package redis

import (
	"Nogler/services/poker"
	"database/sql/driver"
	"encoding/json"
	"fmt"
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
	CurrentBlind    int        `json:"current_blind"`    // Matches game_lobbies.current_blind
	NumberOfVotes   int        `json:"number_of_votes"`  // Matches game_lobbies.number_of_votes
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
	PackSeed int64        `json:"pack_seed,omitempty"`
	Content  PackContents `gorm:"type:jsonb" json:"content"` // Directly store PackContents
}

type PackContents struct {
	Cards  []poker.Card   `json:"cards"`
	Jokers []poker.Jokers `json:"jokers"`
}

// Value - Serialize to JSON
func (p PackContents) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan - Deserialize from JSON
func (p *PackContents) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}
	return json.Unmarshal(b, p)
}
