package redis

import (
	"Nogler/services/poker"
	"time"
)

// GameLobby represents a game lobby state
type GameLobby struct {
	Id              string     `json:"id"`               // Matches game_lobbies.id
	CreatorUsername string     `json:"creator_username"` // Matches game_lobbies.creator_username
	NumberOfRounds  int        `json:"number_of_rounds"` // Matches game_lobbies.number_of_rounds
	TotalPoints     int        `json:"total_points"`     // Matches game_lobbies.total_points
	CreatedAt       time.Time  `json:"created_at"`       // Matches game_lobbies.created_at
	GameHasBegun    bool       `json:"game_has_begun"`   // Matches game_lobbies.game_has_begun
	IsPublic        bool       `json:"is_public"`        // Matches game_lobbies.is_public
	ShopState       *LobbyShop `json:"shop_state"`       // Matches game_lobbies.shop_state
}

type LobbyShop struct {
	Rerolls         int        `json:"reroll_count"`
	FixedPacks      []ShopItem `json:"fixed_packs"`
	FixedModifiers  []ShopItem `json:"fixed_modifiers"`
	RerollableItems []ShopItem `json:"rerollable_items"`
}

type ShopItem struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "card", "joker", "pack", "modifier"
	Price    int          `json:"price"`
	PackSeed int64        `json:"pack_seed,omitempty"` // For deterministic generation
	Content  PackContents `json:"content,omitempty"`
}

type PackContents struct {
	Cards  []poker.Card `json:"cards"`
	Jokers []string     `json:"jokers"`
}
