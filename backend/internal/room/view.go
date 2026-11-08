package room

// Presence, kullanıcının şu an WebSocket ile bağlı olup olmadığını söyler
// (RoomView.members[].connected). Gerçek zamanlı katman (RT) sağlar; o
// gelene kadar NoPresence herkesi bağlı değil sayar.
type Presence interface {
	Connected(userID string) bool
}

// NoPresence, bağlantı bilgisi olmayan varsayılandır.
type NoPresence struct{}

func (NoPresence) Connected(string) bool { return false }

// View, istemciye giden oda görünümüdür (api.md "RoomView"). Yalnızca aktif
// üyeler, katılma sırasıyla listelenir.
type View struct {
	ID            string       `json:"id"`
	InviteCode    string       `json:"invite_code"`
	Status        Status       `json:"status"`
	HostUserID    string       `json:"host_user_id"`
	Capacity      int          `json:"capacity"`
	ScenarioID    *string      `json:"scenario_id"` // nil = rastgele
	CurrentGameID *string      `json:"current_game_id"`
	Version       int64        `json:"version"`
	Members       []MemberView `json:"members"`
}

// MemberView, RoomView içindeki üyedir.
type MemberView struct {
	UserID    string `json:"user_id"`
	Nickname  string `json:"nickname"`
	Role      Role   `json:"role"`
	IsReady   bool   `json:"is_ready"`
	Connected bool   `json:"connected"`
}

// NewView, odanın görünümünü üretir.
func NewView(r Room, presence Presence) View {
	if presence == nil {
		presence = NoPresence{}
	}
	v := View{ID: r.ID, InviteCode: r.InviteCode, Status: r.Status, HostUserID: r.HostUserID,
		Capacity: r.Capacity, ScenarioID: r.ScenarioID, CurrentGameID: r.CurrentGameID, Version: r.Version, Members: []MemberView{}}
	for _, m := range r.Active() {
		v.Members = append(v.Members, MemberView{UserID: m.UserID, Nickname: m.Nickname, Role: m.Role,
			IsReady: m.Ready, Connected: presence.Connected(m.UserID)})
	}
	return v
}
