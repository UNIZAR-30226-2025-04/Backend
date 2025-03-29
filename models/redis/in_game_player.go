package redis

import "encoding/json"

// InGamePlayer represents a player's state during a game
type InGamePlayer struct {
	Username       string          `json:"username"`         // Matches game_profiles.username
	LobbyId        string          `json:"lobby_id"`         // Matches game_lobbies.id
	PlayersMoney   int             `json:"players_money"`    // Matches in_game_players.players_money
	CurrentDeck    json.RawMessage `json:"current_deck"`     // Temporary Redis field
	Modifiers      json.RawMessage `json:"modifiers"`        // Temporary Redis field
	CurrentJokers  json.RawMessage `json:"current_jokers"`   // Temporary Redis field
	MostPlayedHand json.RawMessage `json:"most_played_hand"` // Matches in_game_players.most_played_hand
	Winner         bool            `json:"winner"`           // Matches in_game_players.winner
}
