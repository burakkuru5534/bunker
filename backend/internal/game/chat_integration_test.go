package game

import (
	"fmt"
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
)

func (a *api) say(l lobby, user, id, text string) resp {
	return a.do("POST", "/v1/rooms/"+l.roomID+"/messages", l.tokenOf(user), fmt.Sprintf(`{"client_msg_id":%q,"text":%q}`, id, text))
}

// Oyun sürerken yalnızca hayattakiler yazar; elenen ve ayrılıp izleyici
// dönen okur, yazamaz.
func TestIntegrationChatDuringGame(t *testing.T) {
	a := newAPI(t, true)
	l, gameID := a.startGame(5, 1)
	if r := a.say(l, l.users[1], newActionID(), "başlıyoruz"); r.code != 201 {
		t.Fatalf("hayattaki: %d %s", r.code, r.raw)
	}
	if got := a.scalar(`select game_id::text from chat_messages where room_id = $1`, l.roomID); got != gameID {
		t.Fatalf("game_id %v", got)
	}
	a.toPhase(gameID, engine.PhaseVoting)
	a.voteAll(l, gameID, map[int]int{0: 4, 1: 4, 2: 4, 3: 4, 4: 0}) // u4 elenir
	if r := a.say(l, l.users[4], newActionID(), "ben de"); r.code != 403 || errCode(r) != "FORBIDDEN" {
		t.Fatalf("elenen: %d %s", r.code, r.raw)
	}
	// Ayrılıp izleyici dönen de yazamaz.
	a.leaveRoom(l, l.users[3])
	code := a.scalar(`select invite_code from rooms where id = $1`, l.roomID).(string)
	a.do("POST", "/v1/rooms/join", l.tokenOf(l.users[3]), `{"invite_code":"`+code+`"}`)
	if r := a.say(l, l.users[3], newActionID(), "izleyici"); r.code != 403 {
		t.Fatalf("izleyici: %d %s", r.code, r.raw)
	}
	if r := a.say(l, l.users[2], newActionID(), "hâlâ buradayım"); r.code != 201 {
		t.Fatalf("hayattaki 2: %d %s", r.code, r.raw)
	}
}
