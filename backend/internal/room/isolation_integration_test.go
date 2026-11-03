package room

// SAFE-003 / ROOM-006: oda yalıtımı. A odasının üyesi, B odasına ait hiçbir
// rotada B hakkında bilgi alamaz ve B'yi değiştiremez; yanıtı, var olmayan
// bir odanınkiyle aynıdır. roomScopedRoutes'taki HER rota denenir.

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
)

// sampleBodies, her oda rotası için geçerli bir örnek gövde. Yeni rota
// eklenip burada gövdesi verilmezse test başarısız olur.
func sampleBodies(targetUserID string) map[string]string {
	return map[string]string{
		"GET /v1/rooms/{roomId}":            "",
		"POST /v1/rooms/{roomId}/leave":     "",
		"POST /v1/rooms/{roomId}/ready":     `{"ready":true}`,
		"PATCH /v1/rooms/{roomId}/settings": `{"capacity":1}`,
		"POST /v1/rooms/{roomId}/kick":      `{"user_id":"` + targetUserID + `"}`,
		"POST /v1/rooms/{roomId}/start":     ``,
	}
}

// roomState, B odasının karşılaştırılabilir hali (yalıtım ihlali tespiti).
func (a *api) roomState(roomID string) string {
	a.t.Helper()
	var out []byte
	err := a.env.Admin.QueryRow(context.Background(), `select json_build_object(
		'room', (select row_to_json(r) from `+a.env.Schema+`.rooms r where r.id = $1),
		'members', (select json_agg(m order by m.user_id) from `+a.env.Schema+`.room_members m where m.room_id = $1))::text`, roomID).Scan(&out)
	if err != nil {
		a.t.Fatal(err)
	}
	return string(out)
}

// normalize, yanıttan isteğe özgü alanları (request_id) atar.
func normalize(r resp) string {
	b := map[string]any{}
	_ = json.Unmarshal([]byte(r.raw), &b)
	if e, ok := b["error"].(map[string]any); ok {
		delete(e, "request_id")
	}
	out, _ := json.Marshal(b)
	return string(out)
}

func TestIntegrationRoomIsolationMatrix(t *testing.T) {
	a := newAPI(t)
	aHost, roomA, codeA := a.room("Ahmet")
	aMember, _ := a.guest("Aylin")
	a.join(aMember, codeA)

	_, roomB, codeB := a.room("Berk")
	bMember, bMemberID := a.guest("Banu")
	a.join(bMember, codeB)

	before := a.roomState(roomB)
	missing := "00000000-0000-0000-0000-000000000000"
	bodies := sampleBodies(bMemberID)

	for _, rt := range roomScopedRoutes {
		body, ok := bodies[rt.pattern]
		if !ok {
			t.Errorf("%s için örnek gövde yok (sampleBodies'e ekleyin)", rt.pattern)
			continue
		}
		method, path, _ := strings.Cut(rt.pattern, " ")
		for _, attacker := range []struct{ name, tok string }{{"A sahibi", aHost}, {"A üyesi", aMember}} {
			onB := a.do(method, strings.Replace(path, "{roomId}", roomB, 1), attacker.tok, body)
			onMissing := a.do(method, strings.Replace(path, "{roomId}", missing, 1), attacker.tok, body)
			if onB.code >= 200 && onB.code < 300 && onB.code != 204 {
				t.Errorf("%s %s: B'de başarılı yanıt %d %s", attacker.name, rt.pattern, onB.code, onB.raw)
			}
			// B ile var olmayan oda ayırt edilemez.
			if onB.code != onMissing.code || normalize(onB) != normalize(onMissing) {
				t.Errorf("%s %s: B (%d %s) ≠ olmayan oda (%d %s)", attacker.name, rt.pattern, onB.code, onB.raw, onMissing.code, onMissing.raw)
			}
		}
	}
	if after := a.roomState(roomB); after != before {
		t.Fatalf("B odası değişti:\nönce: %s\nsonra: %s", before, after)
	}
	if len(sampleBodies("")) != len(roomScopedRoutes) {
		t.Fatal("sampleBodies'de fazladan rota var")
	}

	// Kendi odasında yetkili olan A sahibi, B'nin üyesini A yolundan atamaz.
	if r := a.do("POST", "/v1/rooms/"+roomA+"/kick", aHost, `{"user_id":"`+bMemberID+`"}`); r.code != 404 {
		t.Fatalf("başka odanın üyesi atılamaz: %d %s", r.code, r.raw)
	}
	if after := a.roomState(roomB); after != before {
		t.Fatal("B odası değişmemeli")
	}
}

func TestIntegrationMembershipHelper(t *testing.T) {
	a := newAPI(t)
	hostTok, roomID, code := a.room("Burak")
	tok, uid := a.guest("Ayşe")
	a.join(tok, code)
	var hostID string
	_ = a.env.Admin.QueryRow(context.Background(), "select host_user_id::text from "+a.env.Schema+".rooms where id = $1", roomID).Scan(&hostID)

	m, err := a.svc.Membership(context.Background(), hostID, roomID)
	if err != nil || !m.IsHost || m.Role != RolePlayer || m.RoomStatus != StatusLobby || m.CurrentGameID != nil {
		t.Fatalf("sahip üyeliği: %+v %v", m, err)
	}
	m, _ = a.svc.Membership(context.Background(), uid, roomID)
	if m.IsHost || m.Role != RolePlayer {
		t.Fatalf("üye: %+v", m)
	}

	outsider, outID := a.guest("Mehmet")
	_ = outsider
	for name, c := range map[string][2]string{
		"üye olmayan": {outID, roomID},
		"bozuk oda":   {uid, "x"},
		"bozuk kişi":  {"x", roomID},
		"olmayan oda": {uid, "00000000-0000-0000-0000-000000000000"},
	} {
		if _, err := a.svc.Membership(context.Background(), c[0], c[1]); apperr.CodeOf(err) != apperr.NotFound {
			t.Errorf("%s: NOT_FOUND bekleniyordu: %v", name, err)
		}
	}
	// Ayrılınca ve oda kapanınca üyelik yok.
	a.leave(tok, roomID)
	if _, err := a.svc.Membership(context.Background(), uid, roomID); apperr.CodeOf(err) != apperr.NotFound {
		t.Fatal("ayrılan üye değil")
	}
	a.exec("update rooms set status = 'closed', closed_at = now() where id = $1", roomID)
	if _, err := a.svc.Membership(context.Background(), hostID, roomID); apperr.CodeOf(err) != apperr.NotFound {
		t.Fatal("kapalı odada üyelik yok")
	}
	_ = hostTok
	_ = reflect.DeepEqual
}
