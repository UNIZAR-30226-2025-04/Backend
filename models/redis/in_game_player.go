package redis

import "encoding/json"

// InGamePlayer represents a player's state during a game
type InGamePlayer struct {
	Username     string `json:"username"`      // Matches game_profiles.username
	LobbyId      string `json:"lobby_id"`      // Matches game_lobbies.id
	PlayersMoney int    `json:"players_money"` // Matches in_game_players.players_money
	Rerolls      int    `json:"rerolls"`       // Matches in_game_players.rerolls
	// TODO, see whether we use it or not (we would have to update it every time play_hand or draw_cards is called)
	// PlayersRemainingCards int             `json:"current_remaining_cards"` // Cards remaining in deck (deck size - played cards - discarded cards)
	CurrentDeck        json.RawMessage `json:"current_deck"`        // Temporary Redis field
	CurrentHand        json.RawMessage `json:"current_hand"`        // Temporary Redis field
	Modifiers          json.RawMessage `json:"modifiers"`           // Temporary Redis field
	ActivatedModifiers json.RawMessage `json:"activated_modifiers"` // Temporary Redis field
	ReceivedModifiers  json.RawMessage `json:"received_modifiers"`  // Temporary Redis field
	CurrentJokers      json.RawMessage `json:"current_jokers"`      // Temporary Redis field
	MostPlayedHand     json.RawMessage `json:"most_played_hand"`    // Matches in_game_players.most_played_hand
	Winner             bool            `json:"winner"`              // Matches in_game_players.winner
	CurrentRoundPoints int             `json:"current_points"`      // Matches in_game_players.current_points
	TotalGamePoints    int             `json:"total_points"`        // Matches in_game_players.total_points
	HandPlaysLeft      int             `json:"hand_plays_left"`     // Matches in_game_players.hand_plays_left
	DiscardsLeft       int             `json:"discards_left"`       // Matches in_game_players.discards_left

	// Field to store last purchased pack item ID
	LastPurchasedPackItemId int `json:"last_pack_item_id"`

	// Field to indicate if the player is a bot
	IsBot bool `json:"is_bot"` // Matches in_game_players.is_bot

	// Field to store cards that the player has picked from purchased packs
	PurchasedPackCards json.RawMessage `json:"picked_cards"` // Matches in_game_players.picked_cards

	// Map with <K,V> pairs where each key corresponds to a shop item ID and
	// the value is true <=> the user has purchased that item in the current round
	// Otherwise, the entry might not even exist (or be set to false)
	CurrentShopPurchasedItemIDs map[int]bool
}
