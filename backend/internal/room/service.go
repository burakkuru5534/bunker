package room

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
)

// DefaultCapacity, oda oluşturulurken kapasite verilmezse (game-design §1).
const DefaultCapacity = 2

// inviteAttempts, açık odalarla davet kodu çakışmasında deneme sayısı
// (887 milyon olasılıkla pratikte hiç tekrar gerekmez).
const inviteAttempts = 5

// Service, oda işlemleridir (ROOM-003…).
type Service struct {
	db        *persistence.DB
	presence  Presence
	publisher Publisher
	logger    *slog.Logger
	newCode   func() (string, error)
}

// NewService, servisi kurar. presence ve publisher nil olabilir (RT öncesi).
func NewService(db *persistence.DB, presence Presence, publisher Publisher, logger *slog.Logger) *Service {
	if presence == nil {
		presence = NoPresence{}
	}
	if publisher == nil {
		publisher = NoPublisher{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{db: db, presence: presence, publisher: publisher, logger: logger, newCode: NewInviteCode}
}

// Create, yeni oda kurar; oluşturan oda sahibi ve oyuncu olur. Kullanıcı
// başka bir aktif odadaysa ALREADY_IN_ROOM (details.room_id).
func (s *Service) Create(ctx context.Context, p auth.Principal, capacity int) (View, error) {
	if capacity != 1 && capacity != 2 {
		return View{}, apperr.Field("capacity", apperr.FieldNotAllowed)
	}
	for attempt := 1; ; attempt++ {
		code, err := s.newCode()
		if err != nil {
			return View{}, err
		}
		var r Room
		err = s.db.Tx(ctx, func(tx pgx.Tx) error {
			var err error
			r, err = insertRoom(ctx, tx, code, p.UserID, p.Nickname, capacity)
			return err
		})
		switch {
		case err == nil:
			s.logger.Info("oda oluşturuldu", "room_id", r.ID, "capacity", capacity)
			return NewView(r, s.presence), nil
		case persistence.UniqueViolation(err, constraintOneActiveRoom):
			return View{}, s.alreadyInRoom(ctx, p.UserID)
		case persistence.UniqueViolation(err, constraintInviteCode) && attempt < inviteAttempts:
			s.logger.Warn("davet kodu çakıştı, yeniden deneniyor", "attempt", attempt)
			continue
		default:
			return View{}, err
		}
	}
}

func (s *Service) alreadyInRoom(ctx context.Context, userID string) error {
	e := apperr.New(apperr.AlreadyInRoom, "kullanıcı başka bir aktif odada")
	if id, err := activeRoomID(ctx, s.db, userID); err == nil && id != "" {
		e.WithDetail("room_id", id)
	}
	return e
}

// Join, davet koduyla odaya katılır (api.md "Katılma kuralları").
func (s *Service) Join(ctx context.Context, p auth.Principal, rawCode string) (View, error) {
	code, err := NormalizeInviteCode(rawCode)
	if err != nil {
		return View{}, err
	}
	var (
		view  View
		event *Event
	)
	err = s.db.Tx(ctx, func(tx pgx.Tx) error {
		roomID, err := openRoomIDByCode(ctx, tx, code)
		if errors.Is(err, errRoomNotFound) {
			return apperr.New(apperr.InviteCodeNotFound, "davet kodu geçersiz veya oda kapalı")
		}
		if err != nil {
			return err
		}
		r, err := loadForUpdate(ctx, tx, roomID)
		if err != nil {
			return err
		}
		participant, err := wasParticipant(ctx, tx, r.CurrentGameID, p.UserID)
		if err != nil {
			return err
		}
		out, err := Join(r, p.UserID, p.Nickname, participant)
		if err != nil {
			return err
		}
		if out.AlreadyMember {
			view = NewView(r, s.presence)
			return nil
		}
		joined, err := upsertMember(ctx, tx, r.ID, p.UserID, p.Nickname, out.Role)
		if err != nil {
			return err
		}
		if r.Version, err = bumpVersion(ctx, tx, r.ID); err != nil {
			return err
		}
		m := Member{UserID: p.UserID, Nickname: p.Nickname, Role: out.Role, JoinedAt: joined}
		r.Members = append(slicesWithout(r.Members, p.UserID), m)
		view = NewView(r, s.presence)
		event = &Event{RoomID: r.ID, Type: EventMemberJoined, Version: r.Version, Payload: map[string]any{
			"member": MemberView{UserID: m.UserID, Nickname: m.Nickname, Role: m.Role, Connected: s.presence.Connected(m.UserID)},
		}}
		return nil
	})
	switch {
	case persistence.UniqueViolation(err, constraintOneActiveRoom):
		return View{}, s.alreadyInRoom(ctx, p.UserID)
	case persistence.UniqueViolation(err, constraintNickname):
		return View{}, apperr.New(apperr.NicknameTakenInRoom, "odada aynı takma ad var")
	case err != nil:
		return View{}, err
	}
	if event != nil {
		s.publisher.Publish(*event) // yalnızca commit sonrası
		s.logger.Info("odaya katıldı", "room_id", view.ID, "role", string(event.Payload.(map[string]any)["member"].(MemberView).Role))
	}
	return view, nil
}

// Get, odanın görünümünü döndürür. Yalnızca aktif üyeler görebilir; diğer
// her durumda (bozuk kimlik dahil) NOT_FOUND — odanın varlığı sızdırılmaz.
func (s *Service) Get(ctx context.Context, p auth.Principal, roomID string) (View, error) {
	notFound := apperr.New(apperr.NotFound, "oda bulunamadı")
	if !validUUID(roomID) {
		return View{}, notFound
	}
	var r Room
	err := s.db.Read(ctx, func(q persistence.Querier) error {
		var err error
		r, err = loadRoom(ctx, q, roomID, false)
		return err
	})
	if errors.Is(err, errRoomNotFound) {
		return View{}, notFound
	}
	if err != nil {
		return View{}, err
	}
	if m, ok := r.Member(p.UserID); !ok || !m.Active() {
		return View{}, notFound
	}
	return NewView(r, s.presence), nil
}

func slicesWithout(ms []Member, userID string) []Member {
	out := make([]Member, 0, len(ms))
	for _, m := range ms {
		if m.UserID != userID {
			out = append(out, m)
		}
	}
	return out
}

// validUUID, 8-4-4-4-12 onaltılık biçimi denetler (sorgudan önce).
func validUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < 36; i++ {
		c := s[i]
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
				return false
			}
		}
	}
	return true
}

// Leave, kullanıcıyı odadan çıkarır (api.md "Ayrılma kuralları").
// İdempotenttir: kullanıcı zaten üye değilse veya oda yoksa değişiklik
// yapmadan başarılı döner (ağ tekrarı; odanın varlığı sızdırılmaz).
//
// Oyun sürerken ayrılma: üyelik left_at ile kapanır; katılımcının oyundaki
// "ayrıldı" durumu oyun modülünde işlenir (GAME).
func (s *Service) Leave(ctx context.Context, p auth.Principal, roomID string) error {
	if !validUUID(roomID) {
		return nil
	}
	var events []Event
	var closed bool
	err := s.db.Tx(ctx, func(tx pgx.Tx) error {
		r, err := loadForUpdate(ctx, tx, roomID)
		if errors.Is(err, errRoomNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		out, err := Leave(r, p.UserID)
		if apperr.CodeOf(err) == apperr.NotAMember {
			return nil // zaten ayrılmış
		}
		if err != nil {
			return err
		}
		if out.DeleteMembership {
			err = deleteMember(ctx, tx, r.ID, p.UserID)
		} else {
			err = markLeft(ctx, tx, r.ID, p.UserID)
		}
		if err != nil {
			return err
		}
		if out.CloseRoom {
			closed = true
			return closeRoom(ctx, tx, r.ID)
		}
		if out.NewHost != nil {
			if err := setHost(ctx, tx, r.ID, *out.NewHost); err != nil {
				return err
			}
		}
		v, err := bumpVersion(ctx, tx, r.ID)
		if err != nil {
			return err
		}
		events = append(events, Event{RoomID: r.ID, Type: EventMemberLeft, Version: v,
			Payload: map[string]any{"user_id": p.UserID}})
		if out.NewHost != nil {
			events = append(events, Event{RoomID: r.ID, Type: EventHostChanged, Version: v,
				Payload: map[string]any{"host_user_id": *out.NewHost}})
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, e := range events {
		s.publisher.Publish(e) // yalnızca commit sonrası
	}
	if closed {
		s.logger.Info("oda kapandı (son üye ayrıldı)", "room_id", roomID)
	}
	return nil
}

// change, bir oda değişikliğinin sonucudur: yayınlanacak olaylar (sürüm
// mutate tarafından doldurulur). Olay yoksa değişiklik yapılmamıştır.
type change struct {
	events []Event
}

// mutate, lobi işlemlerinin ortak yoludur (ADR-0004): odayı kilitler,
// çağıranın aktif üye olduğunu doğrular (değilse NOT_FOUND; varlık
// sızdırılmaz), fn'nin kararını yazar; olay varsa sürümü bir artırır ve
// commit sonrası olayları yayınlar. Odanın son halini döndürür.
func (s *Service) mutate(ctx context.Context, p auth.Principal, roomID string,
	fn func(tx pgx.Tx, r *Room) (change, error)) (Room, error) {
	notFound := apperr.New(apperr.NotFound, "oda bulunamadı")
	if !validUUID(roomID) {
		return Room{}, notFound
	}
	var (
		r  Room
		ch change
	)
	err := s.db.Tx(ctx, func(tx pgx.Tx) error {
		var err error
		r, err = loadForUpdate(ctx, tx, roomID)
		if errors.Is(err, errRoomNotFound) {
			return notFound
		}
		if err != nil {
			return err
		}
		if !r.isActiveMember(p.UserID) {
			return notFound
		}
		if ch, err = fn(tx, &r); err != nil || len(ch.events) == 0 {
			return err
		}
		if r.Version, err = bumpVersion(ctx, tx, r.ID); err != nil {
			return err
		}
		for i := range ch.events {
			ch.events[i].RoomID, ch.events[i].Version = r.ID, r.Version
		}
		return nil
	})
	if err != nil {
		return Room{}, err
	}
	for _, e := range ch.events {
		s.publisher.Publish(e)
	}
	return r, nil
}

func (r *Room) update(userID string, fn func(*Member)) {
	for i := range r.Members {
		if r.Members[i].UserID == userID {
			fn(&r.Members[i])
		}
	}
}

// SetReady, oyuncunun hazır durumunu değiştirir (ROOM-007). Aynı değer
// tekrar gelirse değişiklik ve olay yoktur.
func (s *Service) SetReady(ctx context.Context, p auth.Principal, roomID string, ready bool) error {
	_, err := s.mutate(ctx, p, roomID, func(tx pgx.Tx, r *Room) (change, error) {
		changed, err := SetReady(*r, p.UserID, ready)
		if err != nil || !changed {
			return change{}, err
		}
		if err := setReady(ctx, tx, r.ID, p.UserID, ready); err != nil {
			return change{}, err
		}
		r.update(p.UserID, func(m *Member) { m.Ready = ready })
		return change{events: []Event{{Type: EventReadyChanged,
			Payload: map[string]any{"user_id": p.UserID, "is_ready": ready}}}}, nil
	})
	return err
}

// ChangeCapacity, oda sahibinin kapasiteyi değiştirmesidir (ROOM-008).
func (s *Service) ChangeCapacity(ctx context.Context, p auth.Principal, roomID string, capacity int) (View, error) {
	r, err := s.mutate(ctx, p, roomID, func(tx pgx.Tx, r *Room) (change, error) {
		changed, err := ChangeCapacity(*r, p.UserID, capacity)
		if err != nil || !changed {
			return change{}, err
		}
		if err := setCapacity(ctx, tx, r.ID, capacity); err != nil {
			return change{}, err
		}
		r.Capacity = capacity
		return change{events: []Event{{Type: EventSettingsChanged, Payload: map[string]any{"capacity": capacity}}}}, nil
	})
	if err != nil {
		return View{}, err
	}
	return NewView(r, s.presence), nil
}

// Kick, oda sahibinin bir üyeyi lobiden atmasıdır (SAFE-007). Atılanın
// WebSocket bağlantıları RT katmanında room.member_kicked olayıyla 4003
// koduyla kapatılır.
func (s *Service) Kick(ctx context.Context, p auth.Principal, roomID, targetID string) error {
	if !validUUID(targetID) {
		return apperr.Field("user_id", apperr.FieldInvalid)
	}
	_, err := s.mutate(ctx, p, roomID, func(tx pgx.Tx, r *Room) (change, error) {
		if err := Kick(*r, p.UserID, targetID); err != nil {
			return change{}, err
		}
		if err := kickMember(ctx, tx, r.ID, targetID); err != nil {
			return change{}, err
		}
		s.logger.Info("üye atıldı", "room_id", r.ID, "target_user_id", targetID)
		return change{events: []Event{{Type: EventMemberKicked, Payload: map[string]any{"user_id": targetID}}}}, nil
	})
	return err
}

// Membership, kullanıcının odadaki aktif üyeliğini döndürür (NOT_FOUND
// değilse). Oyun, sohbet ve WS katmanının ortak yetki denetimidir.
func (s *Service) Membership(ctx context.Context, userID, roomID string) (Membership, error) {
	var m Membership
	err := s.db.Read(ctx, func(q persistence.Querier) error {
		var err error
		m, err = LoadMembership(ctx, q, roomID, userID)
		return err
	})
	return m, err
}
