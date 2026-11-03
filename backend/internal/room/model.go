// Package room, odaların kurallarıdır (ROOM-001). Bu dosyadaki model saftır:
// veritabanı ve HTTP bilmez; her işlem odanın o anki halinden bir karar
// (Outcome) veya kural hatası (apperr) üretir. Servis katmanı (ROOM-003…)
// odayı "for update" ile okur, kararı uygular ve sürümü artırır (ADR-0004).
// Kaynak: docs/api.md "Oda", docs/game-design.md §1, §4.
package room

import (
	"slices"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/auth"
)

// Oyuncu sayısı (game-design.md §1).
const (
	MinPlayers = 4
	MaxPlayers = 10
)

// Status, oda durumudur.
type Status string

const (
	StatusLobby    Status = "lobby"
	StatusInGame   Status = "in_game"
	StatusFinished Status = "finished"
	StatusClosed   Status = "closed"
)

// Role, üyenin odadaki rolüdür.
type Role string

const (
	RolePlayer    Role = "player"
	RoleSpectator Role = "spectator"
)

// Member, oda üyeliğidir. Aktif üye: LeftAt boş.
type Member struct {
	UserID   string
	Nickname string
	Role     Role
	Ready    bool
	JoinedAt time.Time
	LeftAt   *time.Time
	KickedAt *time.Time
}

// Active, üye hâlâ odada mı?
func (m Member) Active() bool { return m.LeftAt == nil }

// Room, odanın anlık halidir (aktif ve ayrılmış/atılmış tüm üyelerle).
type Room struct {
	ID            string
	InviteCode    string
	HostUserID    string
	Capacity      int
	Status        Status
	CurrentGameID *string
	Version       int64
	Members       []Member
}

// Member, kullanıcının üyelik kaydını döndürür (aktif olmasa da).
func (r Room) Member(userID string) (Member, bool) {
	i := slices.IndexFunc(r.Members, func(m Member) bool { return m.UserID == userID })
	if i < 0 {
		return Member{}, false
	}
	return r.Members[i], true
}

// Active, aktif üyeleri katılma sırasıyla döndürür.
func (r Room) Active() []Member {
	var out []Member
	for _, m := range r.Members {
		if m.Active() {
			out = append(out, m)
		}
	}
	slices.SortStableFunc(out, func(a, b Member) int {
		if c := a.JoinedAt.Compare(b.JoinedAt); c != 0 {
			return c
		}
		if a.UserID < b.UserID {
			return -1
		}
		return 1
	})
	return out
}

// Players, aktif oyuncuları döndürür.
func (r Room) Players() []Member {
	return slices.DeleteFunc(r.Active(), func(m Member) bool { return m.Role != RolePlayer })
}

func (r Room) isActiveMember(userID string) bool {
	m, ok := r.Member(userID)
	return ok && m.Active()
}

// --- Katılma -----------------------------------------------------------------

// JoinOutcome, katılma kararıdır.
type JoinOutcome struct {
	// AlreadyMember: kullanıcı zaten bu odanın aktif üyesi; değişiklik yok,
	// RoomView döner (istemci tekrarı / yeniden açılış).
	AlreadyMember bool
	Role          Role
}

// Join, kullanıcının odaya katılıp katılamayacağına karar verir
// (api.md "Katılma kuralları"). wasParticipant: kullanıcı odanın şu anki
// oyununun katılımcısıydı (ayrılmış). "Başka aktif odada" denetimi
// veritabanındaki tekil indeksle yapılır (ALREADY_IN_ROOM).
func Join(r Room, userID, nickname string, wasParticipant bool) (JoinOutcome, error) {
	if r.Status == StatusClosed {
		return JoinOutcome{}, apperr.New(apperr.InviteCodeNotFound, "oda kapalı")
	}
	if m, ok := r.Member(userID); ok {
		if m.KickedAt != nil {
			return JoinOutcome{}, apperr.New(apperr.Kicked, "bu odadan atıldın")
		}
		if m.Active() {
			return JoinOutcome{AlreadyMember: true, Role: m.Role}, nil
		}
	}
	if r.Status == StatusInGame {
		// Oyunun eski katılımcısı yalnızca izleyici olarak döner (game-design §4).
		if wasParticipant {
			return JoinOutcome{Role: RoleSpectator}, nil
		}
		return JoinOutcome{}, apperr.New(apperr.GameInProgress, "oyun sürüyor")
	}
	// lobby veya finished: oyuncu olarak.
	if len(r.Players()) >= MaxPlayers {
		return JoinOutcome{}, apperr.New(apperr.RoomFull, "oda dolu")
	}
	for _, m := range r.Active() {
		if auth.SameNickname(m.Nickname, nickname) {
			return JoinOutcome{}, apperr.New(apperr.NicknameTakenInRoom, "odada aynı takma ad var")
		}
	}
	return JoinOutcome{Role: RolePlayer}, nil
}

// --- Ayrılma ve sahiplik -----------------------------------------------------

// LeaveOutcome, ayrılma kararıdır.
type LeaveOutcome struct {
	// DeleteMembership: lobide/sonuçta üyelik satırı silinir.
	// Aksi halde (oyunda) left_at yazılır; oyun katılımcısı oyun modülünde
	// "ayrıldı" olur (game-design §4).
	DeleteMembership bool
	// NewHost: ayrılan oda sahibiyse sahiplik en eski aktif üyeye geçer.
	NewHost *string
	// CloseRoom: son aktif üye ayrıldı → oda kapanır.
	CloseRoom bool
}

// Leave, kullanıcının odadan ayrılmasına karar verir.
func Leave(r Room, userID string) (LeaveOutcome, error) {
	if !r.isActiveMember(userID) {
		return LeaveOutcome{}, apperr.New(apperr.NotAMember, "oda üyesi değil")
	}
	out := LeaveOutcome{DeleteMembership: r.Status != StatusInGame}
	var remaining []Member
	for _, m := range r.Active() {
		if m.UserID != userID {
			remaining = append(remaining, m)
		}
	}
	if len(remaining) == 0 {
		out.CloseRoom = true
		return out, nil
	}
	if r.HostUserID == userID {
		next := NextHost(remaining).UserID
		out.NewHost = &next
	}
	return out, nil
}

// NextHost, sahipliğin geçeceği üyedir: önce oyuncular, aralarında en eski
// katılan (api.md: "en eski üyeye geçer"). active boş olmamalıdır.
func NextHost(active []Member) Member {
	for _, m := range active {
		if m.Role == RolePlayer {
			return m
		}
	}
	return active[0]
}

// --- Lobi işlemleri ----------------------------------------------------------

func requireLobby(r Room) error {
	if r.Status != StatusLobby {
		return apperr.New(apperr.InvalidPhase, "yalnızca lobide yapılabilir")
	}
	return nil
}

func requireHost(r Room, actorID string) error {
	if !r.isActiveMember(actorID) {
		return apperr.New(apperr.NotAMember, "oda üyesi değil")
	}
	if r.HostUserID != actorID {
		return apperr.New(apperr.Forbidden, "yalnızca oda sahibi yapabilir")
	}
	return nil
}

// SetReady, oyuncunun hazır durumunu değiştirmesine karar verir.
// Aynı değer tekrar gelirse değişiklik yoktur (changed = false).
func SetReady(r Room, userID string, ready bool) (changed bool, err error) {
	m, ok := r.Member(userID)
	if !ok || !m.Active() {
		return false, apperr.New(apperr.NotAMember, "oda üyesi değil")
	}
	if err := requireLobby(r); err != nil {
		return false, err
	}
	if m.Role != RolePlayer {
		return false, apperr.New(apperr.Forbidden, "izleyici hazır olamaz")
	}
	return m.Ready != ready, nil
}

// ChangeCapacity, oda sahibinin kapasiteyi (1 veya 2) değiştirmesine karar verir.
func ChangeCapacity(r Room, actorID string, capacity int) (changed bool, err error) {
	if capacity != 1 && capacity != 2 {
		return false, apperr.Field("capacity", apperr.FieldNotAllowed)
	}
	if err := requireHost(r, actorID); err != nil {
		return false, err
	}
	if err := requireLobby(r); err != nil {
		return false, err
	}
	return r.Capacity != capacity, nil
}

// Kick, oda sahibinin bir üyeyi lobiden atmasına karar verir. Atılan aynı
// odaya tekrar katılamaz (SAFE-007).
func Kick(r Room, actorID, targetID string) error {
	if err := requireHost(r, actorID); err != nil {
		return err
	}
	if err := requireLobby(r); err != nil {
		return err
	}
	if targetID == actorID {
		return apperr.Field("user_id", apperr.FieldInvalid)
	}
	if !r.isActiveMember(targetID) {
		return apperr.New(apperr.NotFound, "üye bulunamadı")
	}
	return nil
}

// StartBlocker, oyunun neden başlatılamadığıdır (istemci düğme metni için
// details.reason).
type StartBlocker string

const (
	StartTooFewPlayers  StartBlocker = "too_few_players"
	StartTooManyPlayers StartBlocker = "too_many_players"
	StartNotAllReady    StartBlocker = "not_all_ready"
)

// CanStart, oda sahibinin oyunu başlatıp başlatamayacağına karar verir:
// lobide, 4–10 oyuncu ve oda sahibi dahil herkes hazır.
func CanStart(r Room, actorID string) error {
	if err := requireHost(r, actorID); err != nil {
		return err
	}
	if err := requireLobby(r); err != nil {
		return err
	}
	players := r.Players()
	blocked := func(b StartBlocker) error {
		return apperr.New(apperr.StartConditionsNotMet, "başlatma koşulları sağlanmadı").
			WithDetail("reason", string(b)).WithDetail("players", len(players))
	}
	switch {
	case len(players) < MinPlayers:
		return blocked(StartTooFewPlayers)
	case len(players) > MaxPlayers:
		return blocked(StartTooManyPlayers)
	}
	for _, m := range players {
		if !m.Ready {
			return blocked(StartNotAllReady)
		}
	}
	return nil
}

// Rematch, oyun bitince oda sahibinin lobiye dönmesine karar verir: tüm
// aktif üyeler oyuncu olur ve hazırlar sıfırlanır (api.md "rematch").
func Rematch(r Room, actorID string) error {
	if err := requireHost(r, actorID); err != nil {
		return err
	}
	if r.Status != StatusFinished {
		return apperr.New(apperr.InvalidPhase, "yalnızca oyun bitince yapılabilir")
	}
	return nil
}
