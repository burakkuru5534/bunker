// Package maintenance, veri saklama ve temizlik işidir (SAFE-009, ROOM-010).
// Sunucu içinde düzenli aralıkla çalışır (harici zamanlayıcı yok); her
// çalıştırma küçük partilerle ilerler ve üst sınırda durur, kalan bir sonraki
// çalıştırmaya kalır. İşlemler idempotenttir; iki instance aynı anda
// çalıştırsa da sonuç değişmez. Sorgular ve indeksleri: docs/queries.md
// "Bakım işleri".
package maintenance

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/account"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

// Retention, saklama süreleri ve parti sınırlarıdır (docs/security.md §6).
type Retention struct {
	Chat           time.Duration // sohbet mesajları
	AbandonedRoom  time.Duration // lobide/bitmiş, hareketsiz ve kimsenin bağlı olmadığı oda kapanır
	ClosedRoom     time.Duration // kapalı oda silinir (oyunlar ve raporlar kalır)
	ResolvedReport time.Duration // sonuçlanmış rapor silinir (açık raporlar kalır)
	InactiveUser   time.Duration // bu kadar kullanılmayan, oturumu kalmamış hesap silinir (SAFE-014)
	Batch          int           // parti büyüklüğü
	MaxBatches     int           // iş başına çalıştırma başına en fazla parti
}

// DefaultRetention, üretim değerleridir.
var DefaultRetention = Retention{
	Chat: 30 * 24 * time.Hour, AbandonedRoom: 6 * time.Hour, ClosedRoom: 7 * 24 * time.Hour,
	ResolvedReport: 180 * 24 * time.Hour, InactiveUser: 90 * 24 * time.Hour, Batch: 500, MaxBatches: 20,
}

// Stats, bir çalıştırmanın sonucudur.
type Stats struct {
	ChatMessages, Sessions, AbandonedRooms, ClosedRooms, Reports, InactiveUsers, Invites int
}

func (s Stats) total() int {
	return s.ChatMessages + s.Sessions + s.AbandonedRooms + s.ClosedRooms + s.Reports + s.InactiveUsers + s.Invites
}

// Job, temizlik işidir.
type Job struct {
	db       *persistence.DB
	presence room.Presence
	logger   *slog.Logger
	ret      Retention
	now      func() time.Time

	mu     sync.Mutex
	stop   chan struct{}
	done   chan struct{}
	runCtx context.Context
	cancel context.CancelFunc
}

// New, işi kurar. presence nil ise odalarda bağlı kimse yok sayılır.
func New(db *persistence.DB, presence room.Presence, ret Retention, logger *slog.Logger) *Job {
	if logger == nil {
		logger = slog.Default()
	}
	if presence == nil {
		presence = room.NoPresence{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Job{db: db, presence: presence, logger: logger, ret: ret, now: time.Now, runCtx: ctx, cancel: cancel}
}

// RunOnce, tüm temizlik adımlarını bir kez çalıştırır. Bir adımın hatası
// diğerlerini durdurmaz; ilk hata döner.
func (j *Job) RunOnce(ctx context.Context) (Stats, error) {
	now := j.now().UTC()
	var st Stats
	var firstErr error
	keep := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	var err error
	st.ChatMessages, err = j.batches(ctx, `delete from chat_messages where id = any(array(
		select id from chat_messages where created_at < $1 order by created_at limit $2))`, now.Add(-j.ret.Chat))
	keep(err)
	st.Sessions, err = j.batches(ctx, `delete from sessions where id = any(array(
		select id from sessions where expires_at < $1 order by expires_at limit $2))`, now)
	keep(err)
	st.AbandonedRooms, err = j.closeAbandoned(ctx, now.Add(-j.ret.AbandonedRoom))
	keep(err)
	// Oturumlar ve terk edilmiş odalar önce temizlenir: aday hesabın oturumu
	// ve aktif odası kalmamış olmalı.
	st.InactiveUsers, err = j.deleteInactiveUsers(ctx, now.Add(-j.ret.InactiveUser))
	keep(err)
	// Oda silinince üyelikler ve mesajlar da silinir (cascade); oyunlar ve
	// raporlar kalır, odaya bağları boşalır (set null).
	st.ClosedRooms, err = j.batches(ctx, `delete from rooms where id = any(array(
		select id from rooms where status = 'closed' and closed_at < $1 order by closed_at limit $2))`, now.Add(-j.ret.ClosedRoom))
	keep(err)
	// Süresi dolmuş lobi davetleri (social.InviteTTL); bir gün beklenir.
	st.Invites, err = j.batches(ctx, `delete from room_invites where (room_id, to_user_id) in (
		select room_id, to_user_id from room_invites where expires_at < $1 order by expires_at limit $2)`, now.Add(-24*time.Hour))
	keep(err)
	st.Reports, err = j.batches(ctx, `delete from reports where id = any(array(
		select id from reports where status in ('actioned', 'dismissed') and resolved_at < $1
		order by resolved_at limit $2))`, now.Add(-j.ret.ResolvedReport))
	keep(err)
	return st, firstErr
}

// batches, toplu silmeyi partilerle (her parti ayrı transaction) çalıştırır.
func (j *Job) batches(ctx context.Context, sql string, cutoff time.Time) (int, error) {
	total := 0
	for range j.ret.MaxBatches {
		var n int64
		err := j.db.Tx(ctx, func(tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, sql, cutoff, j.ret.Batch)
			n = tag.RowsAffected()
			return err
		})
		if err != nil {
			return total, err
		}
		total += int(n)
		if int(n) < j.ret.Batch {
			break
		}
	}
	return total, nil
}

// closeAbandoned, lobide veya bitmiş durumda, cutoff'tan beri hareketsiz ve
// aktif üyelerinden kimsenin bağlı olmadığı odaları kapatır. Üyelikler de
// kapanır: aksi halde "tek aktif oda" kısıtı üyelerin yeni oda kurmasını
// engellerdi. Oyundaki odalar dokunulmaz (oyun, bağlantı kopması kuralıyla
// kendiliğinden biter).
func (j *Job) closeAbandoned(ctx context.Context, cutoff time.Time) (int, error) {
	closed := 0
	for range j.ret.MaxBatches {
		var ids []string
		err := j.db.Read(ctx, func(q persistence.Querier) error {
			rows, err := q.Query(ctx, `select id::text from rooms where status <> 'closed' and status <> 'in_game'
				and updated_at < $1 order by updated_at limit $2`, cutoff, j.ret.Batch)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err != nil {
					return err
				}
				ids = append(ids, id)
			}
			return rows.Err()
		})
		if err != nil {
			return closed, err
		}
		progressed := false
		for _, id := range ids {
			ok, err := j.closeRoom(ctx, id, cutoff)
			if err != nil {
				return closed, err
			}
			if ok {
				closed++
				progressed = true
			}
		}
		// Parti tam değilse veya hiç kapanan yoksa (hepsinde bağlı biri var)
		// bu çalıştırmada başka aday yok.
		if len(ids) < j.ret.Batch || !progressed {
			break
		}
	}
	return closed, nil
}

// closeRoom, odayı kilitleyip koşulları yeniden denetler (bu arada biri
// katılmış/başlatmış olabilir) ve kapatır.
func (j *Job) closeRoom(ctx context.Context, roomID string, cutoff time.Time) (bool, error) {
	var ok bool
	err := j.db.Tx(ctx, func(tx pgx.Tx) error {
		var status string
		var updated time.Time
		err := tx.QueryRow(ctx, `select status, updated_at from rooms where id = $1 for update`, roomID).Scan(&status, &updated)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if status == "closed" || status == "in_game" || !updated.Before(cutoff) {
			return nil
		}
		rows, err := tx.Query(ctx, `select user_id::text from room_members where room_id = $1 and left_at is null`, roomID)
		if err != nil {
			return err
		}
		var members []string
		for rows.Next() {
			var u string
			if err := rows.Scan(&u); err != nil {
				rows.Close()
				return err
			}
			members = append(members, u)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for _, u := range members {
			if j.presence.Connected(u) {
				return nil // biri hâlâ bağlı: dokunma
			}
		}
		if _, err := tx.Exec(ctx, `update room_members set left_at = now(), is_ready = false
			where room_id = $1 and left_at is null`, roomID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `update rooms set status = 'closed', closed_at = now(), updated_at = now()
			where id = $1`, roomID); err != nil {
			return err
		}
		ok = true
		return nil
	})
	return ok, err
}

// inactiveUsersSQL, silinecek hesap adaylarıdır: cutoff'tan beri kullanılmamış,
// hiç oturumu kalmamış (misafir hesaba artık hiçbir cihazdan erişilemez) ve
// aktif odası olmayan. Oturum en geç son kullanımdan 90 gün sonra dolar
// (security.md §1), bu yüzden 90 günlük cutoff'ta oturum zaten kalmamıştır.
const inactiveUsersSQL = `select id::text from users u
	where deleted_at is null and last_seen_at < $1
	  and not exists (select 1 from sessions s where s.user_id = u.id)
	  and not exists (select 1 from room_members m where m.user_id = u.id and m.left_at is null)
	order by last_seen_at limit $2`

// deleteInactiveUsers, kullanılmayan misafir hesapları DELETE /v1/me ile aynı
// şekilde anonimleştirir (SAFE-014). Her hesap ayrı transaction'da, satır
// kilidiyle koşullar yeniden denetlenerek silinir.
func (j *Job) deleteInactiveUsers(ctx context.Context, cutoff time.Time) (int, error) {
	deleted := 0
	for range j.ret.MaxBatches {
		var ids []string
		err := j.db.Read(ctx, func(q persistence.Querier) error {
			rows, err := q.Query(ctx, inactiveUsersSQL, cutoff, j.ret.Batch)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err != nil {
					return err
				}
				ids = append(ids, id)
			}
			return rows.Err()
		})
		if err != nil {
			return deleted, err
		}
		progressed := false
		for _, id := range ids {
			ok, err := j.deleteInactiveUser(ctx, id, cutoff)
			if err != nil {
				return deleted, err
			}
			if ok {
				deleted++
				progressed = true
			}
		}
		if len(ids) < j.ret.Batch || !progressed {
			break
		}
	}
	return deleted, nil
}

// deleteInactiveUser, kullanıcı satırını kilitleyip koşulları yeniden
// denetler (seçimden sonra kullanılmış olabilir) ve anonimleştirir.
func (j *Job) deleteInactiveUser(ctx context.Context, userID string, cutoff time.Time) (bool, error) {
	var ok bool
	err := j.db.Tx(ctx, func(tx pgx.Tx) error {
		var eligible bool
		err := tx.QueryRow(ctx, `select u.deleted_at is null and u.last_seen_at < $2
			and not exists (select 1 from sessions s where s.user_id = u.id)
			and not exists (select 1 from room_members m where m.user_id = u.id and m.left_at is null)
			from users u where u.id = $1 for update`, userID, cutoff).Scan(&eligible)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && !eligible) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := account.Anonymize(ctx, tx, userID); err != nil {
			return err
		}
		ok = true
		return nil
	})
	return ok, err
}

// Start, işi arka planda çalıştırır: ilk çalıştırma firstDelay sonra
// (açılış yükünü artırmasın diye), sonra her interval'de.
func (j *Job) Start(firstDelay, interval time.Duration) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.stop != nil {
		return
	}
	stop, done := make(chan struct{}), make(chan struct{})
	j.stop, j.done = stop, done
	// Döngü kanalları yerel kopyadan okur: Stop alanları boşaltır (ikinci
	// Stop güvenli olsun diye); alandan okumak nil kanalda sonsuza beklerdi.
	go func() {
		defer close(done)
		wait := firstDelay
		for {
			select {
			case <-stop:
				return
			case <-time.After(wait):
			}
			wait = interval
			ctx, cancel := context.WithTimeout(j.runCtx, 5*time.Minute)
			start := time.Now()
			st, err := j.RunOnce(ctx)
			cancel()
			switch {
			case err != nil:
				j.logger.Warn("temizlik işi hatayla bitti", "error", err, "stats", st)
			case st.total() > 0:
				j.logger.Info("temizlik işi", "chat_messages", st.ChatMessages, "sessions", st.Sessions,
					"abandoned_rooms", st.AbandonedRooms, "closed_rooms", st.ClosedRooms, "reports", st.Reports, "inactive_users", st.InactiveUsers, "invites", st.Invites,
					"duration", time.Since(start).String())
			}
		}
	}()
}

// Stop, işi durdurur; süren çalıştırmayı iptal edip bitmesini bekler.
func (j *Job) Stop(ctx context.Context) error {
	j.mu.Lock()
	stop, done := j.stop, j.done
	j.stop = nil // ikinci Stop kapalı kanalı yeniden kapatmasın
	j.mu.Unlock()
	if stop == nil {
		return nil
	}
	close(stop)
	j.cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
