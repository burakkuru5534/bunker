package room

import (
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
)

var t0 = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

func member(id, nick string, role Role, ready bool, joinedMin int) Member {
	return Member{UserID: id, Nickname: nick, Role: role, Ready: ready, JoinedAt: t0.Add(time.Duration(joinedMin) * time.Minute)}
}

// lobby, host = u1, n oyuncu (u1..un), hepsi hazır değil.
func lobby(n int) Room {
	r := Room{ID: "r", HostUserID: "u1", Capacity: 2, Status: StatusLobby}
	names := []string{"Burak", "Ayşe", "Mehmet", "Zeynep", "Can", "Elif", "Emre", "Deniz", "Işık", "Ali", "Fazla"}
	for i := range n {
		r.Members = append(r.Members, member("u"+string(rune('0'+i+1)), names[i], RolePlayer, false, i))
	}
	return r
}

func code(err error) apperr.Code { return apperr.CodeOf(err) }

func TestJoin(t *testing.T) {
	r := lobby(3)
	out, err := Join(r, "new", "Yeni", false)
	if err != nil || out.Role != RolePlayer || out.AlreadyMember {
		t.Fatalf("lobiye oyuncu: %+v %v", out, err)
	}
	if out, _ := Join(r, "u2", "Ayşe", false); !out.AlreadyMember || out.Role != RolePlayer {
		t.Fatal("zaten üye: değişiklik yok")
	}
	// Türkçe büyük/küçük harf duyarsız takma ad çakışması.
	if _, err := Join(r, "new", "AYŞE", false); code(err) != apperr.NicknameTakenInRoom {
		t.Fatalf("takma ad çakışması: %v", err)
	}
	if _, err := Join(lobby(10), "new", "Yeni", false); code(err) != apperr.RoomFull {
		t.Fatalf("10 oyuncu dolu: %v", err)
	}

	// Atılan tekrar katılamaz.
	k := lobby(3)
	now := t0
	k.Members[2].LeftAt, k.Members[2].KickedAt = &now, &now
	if _, err := Join(k, "u3", "Mehmet", false); code(err) != apperr.Kicked {
		t.Fatalf("atılan: %v", err)
	}
	// Ayrılan (atılmayan) lobiye geri dönebilir; eski takma adı serbest.
	l := lobby(3)
	l.Members[2].LeftAt = &now
	if out, err := Join(l, "u3", "Mehmet", false); err != nil || out.Role != RolePlayer {
		t.Fatalf("ayrılan geri döner: %+v %v", out, err)
	}

	g := lobby(4)
	g.Status = StatusInGame
	if _, err := Join(g, "new", "Yeni", false); code(err) != apperr.GameInProgress {
		t.Fatalf("oyun sürerken yabancı: %v", err)
	}
	if out, err := Join(g, "new", "Yeni", true); err != nil || out.Role != RoleSpectator {
		t.Fatalf("eski katılımcı izleyici döner: %+v %v", out, err)
	}

	f := lobby(4)
	f.Status = StatusFinished
	if out, err := Join(f, "new", "Yeni", false); err != nil || out.Role != RolePlayer {
		t.Fatalf("bitmiş odaya oyuncu (tekrar oyna için): %+v %v", out, err)
	}
	c := lobby(1)
	c.Status = StatusClosed
	if _, err := Join(c, "new", "Yeni", false); code(err) != apperr.InviteCodeNotFound {
		t.Fatalf("kapalı oda: %v", err)
	}
}

func TestLeaveAndHostTransfer(t *testing.T) {
	r := lobby(3)
	out, err := Leave(r, "u2")
	if err != nil || !out.DeleteMembership || out.NewHost != nil || out.CloseRoom {
		t.Fatalf("lobide sıradan üye: %+v %v", out, err)
	}
	out, _ = Leave(r, "u1")
	if out.NewHost == nil || *out.NewHost != "u2" {
		t.Fatalf("sahiplik en eski üyeye: %+v", out)
	}
	if _, err := Leave(r, "yok"); code(err) != apperr.NotAMember {
		t.Fatalf("üye olmayan: %v", err)
	}
	if out, _ := Leave(lobby(1), "u1"); !out.CloseRoom || out.NewHost != nil {
		t.Fatalf("son üye odayı kapatır: %+v", out)
	}

	// Oyunda: üyelik silinmez (left_at), sahiplik oyuncuya geçer.
	g := lobby(3)
	g.Status = StatusInGame
	g.Members[1].Role = RoleSpectator // u2 izleyici (en eski)
	out, _ = Leave(g, "u1")
	if out.DeleteMembership || out.NewHost == nil || *out.NewHost != "u3" {
		t.Fatalf("oyunda sahiplik oyuncuya: %+v", out)
	}
	// Yalnızca izleyiciler kaldıysa en eskiye.
	if h := NextHost([]Member{member("a", "Aaa", RoleSpectator, false, 2), member("b", "Bbb", RoleSpectator, false, 1)}); h.UserID != "a" {
		t.Fatalf("Active() sıralı verir; ilk eleman: %s", h.UserID)
	}
}

func TestActiveOrdering(t *testing.T) {
	r := Room{Members: []Member{
		member("c", "Ccc", RolePlayer, false, 5),
		member("b", "Bbb", RolePlayer, false, 1),
		member("a", "Aaa", RolePlayer, false, 1),
	}}
	got := r.Active()
	if got[0].UserID != "a" || got[1].UserID != "b" || got[2].UserID != "c" {
		t.Fatalf("katılma zamanı, eşitlikte kimlik: %v", []string{got[0].UserID, got[1].UserID, got[2].UserID})
	}
}

func TestSetReady(t *testing.T) {
	r := lobby(3)
	if changed, err := SetReady(r, "u2", true); err != nil || !changed {
		t.Fatalf("hazır: %v %v", changed, err)
	}
	if changed, _ := SetReady(r, "u2", false); changed {
		t.Fatal("aynı değer: değişiklik yok")
	}
	if _, err := SetReady(r, "yok", true); code(err) != apperr.NotAMember {
		t.Fatal("üye olmayan")
	}
	g := lobby(3)
	g.Status = StatusInGame
	if _, err := SetReady(g, "u2", true); code(err) != apperr.InvalidPhase {
		t.Fatal("oyunda hazır olunamaz")
	}
	s := lobby(3)
	s.Members[1].Role = RoleSpectator
	if _, err := SetReady(s, "u2", true); code(err) != apperr.Forbidden {
		t.Fatal("izleyici hazır olamaz")
	}
}

func TestHostOnlyLobbyActions(t *testing.T) {
	r := lobby(4)
	if changed, err := ChangeCapacity(r, "u1", 1); err != nil || !changed {
		t.Fatalf("kapasite: %v %v", changed, err)
	}
	if changed, _ := ChangeCapacity(r, "u1", 2); changed {
		t.Fatal("aynı kapasite")
	}
	if _, err := ChangeCapacity(r, "u1", 3); code(err) != apperr.ValidationFailed {
		t.Fatal("kapasite 1 veya 2")
	}
	if _, err := ChangeCapacity(r, "u2", 1); code(err) != apperr.Forbidden {
		t.Fatal("yalnızca oda sahibi")
	}
	if _, err := ChangeCapacity(r, "yok", 1); code(err) != apperr.NotAMember {
		t.Fatal("üye olmayan")
	}
	g := lobby(4)
	g.Status = StatusInGame
	if _, err := ChangeCapacity(g, "u1", 1); code(err) != apperr.InvalidPhase {
		t.Fatal("oyunda değiştirilemez")
	}

	if err := Kick(r, "u1", "u3"); err != nil {
		t.Fatal(err)
	}
	if code(Kick(r, "u2", "u3")) != apperr.Forbidden || code(Kick(r, "u1", "u1")) != apperr.ValidationFailed ||
		code(Kick(r, "u1", "yok")) != apperr.NotFound || code(Kick(g, "u1", "u3")) != apperr.InvalidPhase {
		t.Fatal("atma kuralları")
	}
}

func TestCanStart(t *testing.T) {
	reason := func(err error) string {
		var e *apperr.Error
		if ae, ok := err.(*apperr.Error); ok {
			e = ae
		}
		if e == nil || e.Code != apperr.StartConditionsNotMet {
			t.Fatalf("START_CONDITIONS_NOT_MET bekleniyordu: %v", err)
		}
		return e.Details["reason"].(string)
	}
	allReady := func(r Room) Room {
		for i := range r.Members {
			r.Members[i].Ready = true
		}
		return r
	}

	if reason(CanStart(allReady(lobby(3)), "u1")) != string(StartTooFewPlayers) {
		t.Fatal("3 oyuncu yetmez")
	}
	if reason(CanStart(lobby(4), "u1")) != string(StartNotAllReady) {
		t.Fatal("herkes hazır olmalı")
	}
	r := allReady(lobby(4))
	r.Members[0].Ready = false // oda sahibi de hazır olmalı
	if reason(CanStart(r, "u1")) != string(StartNotAllReady) {
		t.Fatal("oda sahibi dahil")
	}
	for _, n := range []int{4, 10} {
		if err := CanStart(allReady(lobby(n)), "u1"); err != nil {
			t.Fatalf("%d oyuncu başlamalı: %v", n, err)
		}
	}
	over := allReady(lobby(11))
	if reason(CanStart(over, "u1")) != string(StartTooManyPlayers) {
		t.Fatal("11 oyuncu (savunma)")
	}
	// İzleyiciler sayılmaz.
	s := allReady(lobby(5))
	s.Members[4].Role, s.Members[4].Ready = RoleSpectator, false
	if err := CanStart(s, "u1"); err != nil {
		t.Fatalf("izleyici hazır sayımına girmez: %v", err)
	}
	if code(CanStart(allReady(lobby(4)), "u2")) != apperr.Forbidden {
		t.Fatal("yalnızca oda sahibi başlatır")
	}
}

func TestRematch(t *testing.T) {
	f := lobby(4)
	f.Status = StatusFinished
	if err := Rematch(f, "u1"); err != nil {
		t.Fatal(err)
	}
	if code(Rematch(f, "u2")) != apperr.Forbidden || code(Rematch(lobby(4), "u1")) != apperr.InvalidPhase {
		t.Fatal("tekrar oyna kuralları")
	}
}
