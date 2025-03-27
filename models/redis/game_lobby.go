package redis

type PlayerStatus string

const (
	StatusOnline  PlayerStatus = "online"
	StatusOffline PlayerStatus = "offline"
	StatusPlaying PlayerStatus = "playing"
	StatusAFK     PlayerStatus = "afk"
)

type PlayerPresence struct {
	Username string       `json:"username"`
	Status   PlayerStatus `json:"status"`
	LastPing int64        `json:"last_ping"` // Unix timestamp
	SocketID string       `json:"socket_id"` // For direct messaging
}
