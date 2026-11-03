package room

// Oda olayları (api.md §5 "Oda olayları"). Yalnızca transaction commit
// edildikten sonra yayınlanır (ADR-0004); gerçek zamanlı katman (RT) odadaki
// bağlantılara iletir.
const (
	EventMemberJoined    = "room.member_joined"
	EventMemberLeft      = "room.member_left"
	EventMemberKicked    = "room.member_kicked"
	EventReadyChanged    = "room.ready_changed"
	EventSettingsChanged = "room.settings_changed"
	EventHostChanged     = "room.host_changed"
	EventReturnedToLobby = "room.returned_to_lobby"
)

// Event, odadaki herkese giden olaydır. Version, olaydan sonraki oda
// sürümüdür (zarfın state_version alanı).
type Event struct {
	RoomID  string
	Type    string
	Version int64
	Payload any
}

// Publisher, olayları iletir. RT gelene kadar NoPublisher kullanılır.
type Publisher interface {
	Publish(Event)
}

// NoPublisher, olayları atar.
type NoPublisher struct{}

func (NoPublisher) Publish(Event) {}
