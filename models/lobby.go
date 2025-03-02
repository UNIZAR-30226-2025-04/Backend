package models

// Lobby represents a game room
type Lobby struct {
	Code       string `json:"code"`
	HostEmail  string `json:"host_email"`
	HostName   string `json:"host_name"`
	Visibility string `json:"visibility"` // "public", "private"
	MaxPlayers int    `json:"max_players"`
	Players    []string `json:"players"`
	IsStarted  bool   `json:"is_started"`
	IsFull     bool   `json:"is_full"`
	CreatedAt  string `json:"created_at"`
}

// LobbyInvitation represents a lobby invitation
type LobbyInvitation struct {
	ID           string `json:"id"`
	LobbyCode    string `json:"lobby_code"`
	SenderEmail  string `json:"sender_email"`
	SenderName   string `json:"sender_name"`
	ReceiverEmail string `json:"receiver_email"`
	CreatedAt    string `json:"created_at"`
}

// LobbyCreation to create a new lobby
type LobbyCreation struct {
	Visibility string `json:"visibility" binding:"required"` // "public", "private"
	MaxPlayers int    `json:"max_players" binding:"required,min=2,max=10"`
} 