package game

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

// EventPlayerConnection: oyuncunun bağlantı durumu (api.md; oyun durumunu
// değiştirmez, state_version taşımaz).
const EventPlayerConnection = "game.player_connection"

// DefaultDisconnectGrace, kopan oyuncunun geri dönmesi için beklenen süredir
// (game-design §4).
const DefaultDisconnectGrace = 60 * time.Second

// Disconnects, oyun sürerken bağlantısı kopan oyuncuları izler (RT-007):
//   - son bağlantı kapanınca hayattaki katılımcıya game_participants.
//     disconnected_at yazılır, odaya player_connection{disconnected} gider;
//   - süre içinde dönerse kayıt silinir, {reconnected} gider;
//   - dönmezse oyuncu "Ayrıl" ile aynı yoldan odadan ve oyundan ayrılır
//     (kayıt: disconnect_timeout), {left} gider.
//
// Kaynak veritabanıdır: açılışta (Start) aktif oyunlardaki hayattaki
// oyunculara — bağlantıların hepsi kopmuş olacağından — süre tanınır, önceden
// kopmuşların kalan süresi korunur. Bildirimler sırasız gelebileceği için
// her kararda gerçek bağlantı durumu (presence) denetlenir.
type Disconnects struct {
	db        *persistence.DB
	rooms     *room.Service
	presence  room.Presence
	publisher room.Publisher
	grace     time.Duration
	logger    *slog.Logger
	now       func() time.Time

	mu      sync.Mutex
	timers  map[string]*time.Timer // kullanıcı → süre dolumu
	stopped bool
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewDisconnects, izleyiciyi kurar.
func NewDisconnects(svc *Service, rooms *room.Service, presence room.Presence, logger *slog.Logger) *Disconnects {
	if logger == nil {
		logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Disconnects{db: svc.db, rooms: rooms, presence: presence, publisher: svc.publisher,
		grace: DefaultDisconnectGrace, logger: logger, now: time.Now,
		timers: map[string]*time.Timer{}, ctx: ctx, cancel: cancel}
}

// go_, izlenen bir arka plan işi başlatır (Stop bekler).
func (d *Disconnects) run(fn func(ctx context.Context)) {
	d.mu.Lock()
	if d.stopped {
		d.mu.Unlock()
		return
	}
	d.wg.Add(1)
	d.mu.Unlock()
	go func() {
		defer d.wg.Done()
		ctx, cancel := context.WithTimeout(d.ctx, 10*time.Second)
		defer cancel()
		fn(ctx)
	}()
}

// retry, fn'yi geçici hatalarda artan aralıklarla (0,5 → 8 sn) yeniden dener;
// her deneme kendi bağlamıyla. Durdurulursa bırakır. fn, iş bittiyse
// (başarı veya kalıcı karar) true döner.
func (d *Disconnects) retry(what, userID string, fn func(ctx context.Context) (done bool, err error)) {
	d.run(func(context.Context) {
		wait := 500 * time.Millisecond
		for attempt := 1; ; attempt++ {
			ctx, cancel := context.WithTimeout(d.ctx, 10*time.Second)
			done, err := fn(ctx)
			cancel()
			if done || d.ctx.Err() != nil {
				return
			}
			d.logger.Warn(what+" kaydedilemedi; yeniden denenecek", "user_id", userID, "attempt", attempt,
				"retry_in", wait.String(), "error", err)
			select {
			case <-d.ctx.Done():
				return
			case <-time.After(wait):
			}
			wait = min(wait*2, 8*time.Second)
		}
	})
}

// activeGame, odanın sürmekte olan oyunudur ("" yoksa).
func activeGame(ctx context.Context, q persistence.Querier, roomID string) (string, error) {
	var gameID string
	err := q.QueryRow(ctx, `select g.id::text from rooms r join games g on g.id = r.current_game_id
		where r.id = $1 and r.status = 'in_game' and g.phase <> 'finished'`, roomID).Scan(&gameID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return gameID, err
}

func (d *Disconnects) publish(roomID, gameID, userID, state string) {
	d.publisher.Publish(room.Event{RoomID: roomID, GameID: gameID, Type: EventPlayerConnection,
		Payload: map[string]any{"user_id": userID, "state": state}})
}

// UserDisconnected, realtime.ConnectionListener. Veritabanı hatasında yeniden
// denenir: kayıt kaybolursa kopan oyuncu hiç "ayrıldı" sayılmazdı.
func (d *Disconnects) UserDisconnected(userID, roomID string) {
	d.retry("bağlantı kopması", userID, func(ctx context.Context) (bool, error) {
		if d.presence.Connected(userID) {
			return true, nil // bu arada yeniden bağlandı
		}
		var gameID string
		var at time.Time
		err := d.db.Tx(ctx, func(tx pgx.Tx) error {
			var err error
			if gameID, err = activeGame(ctx, tx, roomID); err != nil || gameID == "" {
				return err
			}
			err = tx.QueryRow(ctx, `update game_participants set disconnected_at = $3
				where game_id = $1 and user_id = $2 and status = 'alive' and disconnected_at is null
				returning disconnected_at`, gameID, userID, d.now().UTC()).Scan(&at)
			if errors.Is(err, pgx.ErrNoRows) {
				gameID = "" // hayatta değil veya zaten kopuk
				return nil
			}
			return err
		})
		if err != nil {
			return false, err
		}
		if gameID != "" {
			d.publish(roomID, gameID, userID, "disconnected")
			d.schedule(userID, roomID, gameID, at.Add(d.grace))
		}
		return true, nil
	})
}

// UserConnected, realtime.ConnectionListener.
func (d *Disconnects) UserConnected(userID, roomID string) {
	d.cancelTimer(userID)
	d.retry("yeniden bağlanma", userID, func(ctx context.Context) (bool, error) {
		gameID, ok, err := d.clear(ctx, userID, roomID)
		if err != nil {
			return false, err
		}
		if ok {
			d.publish(roomID, gameID, userID, "reconnected")
		}
		return true, nil
	})
}

// clear, kopma kaydını siler; silindiyse oyun kimliği ve true.
func (d *Disconnects) clear(ctx context.Context, userID, roomID string) (string, bool, error) {
	var gameID string
	err := d.db.Tx(ctx, func(tx pgx.Tx) error {
		var err error
		if gameID, err = activeGame(ctx, tx, roomID); err != nil || gameID == "" {
			return err
		}
		tag, err := tx.Exec(ctx, `update game_participants set disconnected_at = null
			where game_id = $1 and user_id = $2 and disconnected_at is not null`, gameID, userID)
		if err == nil && tag.RowsAffected() == 0 {
			gameID = ""
		}
		return err
	})
	if err != nil {
		return "", false, err
	}
	return gameID, gameID != "", nil
}

func (d *Disconnects) schedule(userID, roomID, gameID string, at time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.stopped {
		return
	}
	if t, ok := d.timers[userID]; ok {
		t.Stop()
	}
	d.timers[userID] = time.AfterFunc(max(time.Until(at), 0), func() {
		d.mu.Lock()
		delete(d.timers, userID)
		d.mu.Unlock()
		d.run(func(ctx context.Context) { d.expire(ctx, userID, roomID, gameID) })
	})
}

func (d *Disconnects) cancelTimer(userID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if t, ok := d.timers[userID]; ok {
		t.Stop()
		delete(d.timers, userID)
	}
}

// Pending, bekleyen süre dolumu sayısıdır.
func (d *Disconnects) Pending() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.timers)
}

// expire: süre doldu. Oyuncu bu arada bağlandıysa kayıt silinir; hâlâ kopuk,
// hayatta ve oyun sürüyorsa odadan ve oyundan ayrılır.
func (d *Disconnects) expire(ctx context.Context, userID, roomID, gameID string) {
	if d.presence.Connected(userID) {
		// Bağlı: kayıt sessizce silinir (açılış kurtarmasında bağlı olduğu
		// anlaşılan oyuncu için "koptu" hiç yayınlanmamıştı).
		if _, _, err := d.clear(ctx, userID, roomID); err != nil {
			d.logger.Warn("bağlı oyuncunun kopma kaydı silinemedi", "user_id", userID, "error", err)
		}
		return
	}
	var still bool
	err := d.db.Read(ctx, func(q persistence.Querier) error {
		return q.QueryRow(ctx, `select exists (select 1 from game_participants p join games g on g.id = p.game_id
			where p.game_id = $1 and p.user_id = $2 and p.status = 'alive' and p.disconnected_at is not null
			and g.phase <> 'finished')`, gameID, userID).Scan(&still)
	})
	if err != nil {
		d.logger.Warn("bağlantı süresi denetlenemedi; yeniden denenecek", "user_id", userID, "error", err)
		d.schedule(userID, roomID, gameID, d.now().Add(5*time.Second))
		return
	}
	if !still {
		return
	}
	if err := d.rooms.Leave(withLeaveReason(ctx, ActionDisconnectTimeout), auth.Principal{UserID: userID}, roomID); err != nil {
		d.logger.Warn("kopan oyuncu ayrılamadı; yeniden denenecek", "user_id", userID, "error", err)
		d.schedule(userID, roomID, gameID, d.now().Add(5*time.Second))
		return
	}
	d.logger.Info("bağlantısı dönmeyen oyuncu ayrıldı", "user_id", userID, "room_id", roomID, "game_id", gameID)
	d.publish(roomID, gameID, userID, "left")
}

// Start, açılış kurtarmasıdır: aktif oyunlardaki hayattaki oyunculardan kopma
// kaydı olmayanlara şimdiden süre tanınır (sunucu yeni açıldı, kimse bağlı
// değil), tüm kopuklar planlanır. Bu arada bağlanan olursa süre dolumunda
// presence denetimi kaydı siler.
func (d *Disconnects) Start() {
	d.run(func(ctx context.Context) {
		type row struct {
			user, room, game string
			at               time.Time
		}
		var rows []row
		err := d.db.Tx(ctx, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `update game_participants p set disconnected_at = $1
				from games g where g.id = p.game_id and g.phase <> 'finished'
				and p.status = 'alive' and p.disconnected_at is null`, d.now().UTC()); err != nil {
				return err
			}
			r, err := tx.Query(ctx, `select p.user_id::text, g.room_id::text, g.id::text, p.disconnected_at
				from game_participants p join games g on g.id = p.game_id
				where g.phase <> 'finished' and g.room_id is not null and p.status = 'alive' and p.disconnected_at is not null`)
			if err != nil {
				return err
			}
			defer r.Close()
			for r.Next() {
				var x row
				if err := r.Scan(&x.user, &x.room, &x.game, &x.at); err != nil {
					return err
				}
				rows = append(rows, x)
			}
			return r.Err()
		})
		if err != nil {
			d.logger.Error("bağlantı kopması kurtarması başarısız", "error", err)
			return
		}
		for _, x := range rows {
			d.schedule(x.user, x.room, x.game, x.at.Add(d.grace))
		}
		d.logger.Info("bağlantı kopması kurtarması tamam", "waiting_players", len(rows))
	})
}

// Stop, bekleyen süreleri bırakır (veritabanındaki kayıtlar kalır) ve
// çalışan işleri ctx süresince bekler.
func (d *Disconnects) Stop(ctx context.Context) error {
	d.mu.Lock()
	d.stopped = true
	for id, t := range d.timers {
		t.Stop()
		delete(d.timers, id)
	}
	d.mu.Unlock()
	d.cancel()
	done := make(chan struct{})
	go func() { d.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SetGrace, bekleme süresini değiştirir (testler).
func (d *Disconnects) SetGrace(g time.Duration) { d.grace = g }
