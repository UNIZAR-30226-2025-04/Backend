package redis

import "encoding/json"

// InGamePlayer represents a player's state during a game
type InGamePlayer struct {
	Username           string          `json:"username"`            // Matches game_profiles.username
	LobbyId            string          `json:"lobby_id"`            // Matches game_lobbies.id
	PlayersMoney       int             `json:"players_money"`       // Matches in_game_players.players_money
	CurrentDeck        json.RawMessage `json:"current_deck"`        // Temporary Redis field
	Modifiers          json.RawMessage `json:"modifiers"`           // Temporary Redis field
	ActivatedModifiers json.RawMessage `json:"activated_modifiers"` // Temporary Redis field
	ReceivedModifiers  json.RawMessage `json:"received_modifiers"`  // Temporary Redis field
	CurrentJokers      json.RawMessage `json:"current_jokers"`      // Temporary Redis field
	MostPlayedHand     json.RawMessage `json:"most_played_hand"`    // Matches in_game_players.most_played_hand
	Winner             bool            `json:"winner"`              // Matches in_game_players.winner
	// CurrentHand    int             `json:"current_hand"`     // Matches in_game_players.current_hand
	CurrentPoints int `json:"current_points"`  // Matches in_game_players.current_points
	TotalPoints   int `json:"total_points"`    // Matches in_game_players.total_points
	HandPlaysLeft int `json:"hand_plays_left"` // Matches in_game_players.hand_plays_left
	DiscardsLeft  int `json:"discards_left"`   // Matches in_game_players.discards_left
}
