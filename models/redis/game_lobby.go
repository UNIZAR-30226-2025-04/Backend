package redis

import "time"

// GameLobby represents a game lobby state
type GameLobby struct {
	Id              string        `json:"id"`               // Matches game_lobbies.id
	CreatorUsername string        `json:"creator_username"` // Matches game_lobbies.creator_username
	NumberOfRounds  int           `json:"number_of_rounds"` // Matches game_lobbies.number_of_rounds
	TotalPoints     int           `json:"total_points"`     // Matches game_lobbies.total_points
	CreatedAt       time.Time     `json:"created_at"`       // Matches game_lobbies.created_at
	GameHasBegun    bool          `json:"game_has_begun"`   // Matches game_lobbies.game_has_begun
	IsPublic        bool          `json:"is_public"`        // Matches game_lobbies.is_public
	ChatHistory     []ChatMessage `json:"chat_history"`     // Redis-specific field for real-time chat
}
