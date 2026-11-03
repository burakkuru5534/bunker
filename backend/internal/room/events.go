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

// Event, odadaki herkese (To boş) veya yalnızca bir kullanıcının
// bağlantılarına (To dolu; gizli bilgi, ör. game.your_cards) giden olaydır.
// Version, zarfın state_version alanıdır: lobide oda sürümü, oyun sürerken
// oyun sürümü (api.md §5).
type Event struct {
	RoomID  string
	GameID  string // oyun olaylarında dolu (zamanlayıcı bunu izler); istemciye gitmez
	Type    string
	Version int64
	To      string
	// Exclude, olayı almayacak kullanıcılar (ör. göndereni engelleyenler).
	Exclude []string
	Payload any
}

// Publisher, olayları iletir. RT gelene kadar NoPublisher kullanılır.
type Publisher interface {
	Publish(Event)
}

// NoPublisher, olayları atar.
type NoPublisher struct{}

func (NoPublisher) Publish(Event) {}
