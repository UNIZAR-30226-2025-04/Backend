package redis

import (
	"Nogler/services/poker"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// String consts to represent the different phases of the game
const (
	PhaseNone      = "none"
	PhaseBlind     = "blind"
	PhasePlayRound = "play_round"
	PhaseShop      = "shop"
	PhaseVouchers  = "vouchers"
	AnnounceWinner = "announce_winner"
)

// GameLobby represents a game lobby state
type GameLobby struct {
	Id                   string     `json:"id"`                     // Matches game_lobbies.id
	CreatorUsername      string     `json:"creator_username"`       // Matches game_lobbies.creator_username
	MaxRounds            int        `json:"number_of_rounds"`       // Matches game_lobbies.number_of_rounds
	TotalPoints          int        `json:"total_points"`           // Matches game_lobbies.total_points
	CreatedAt            time.Time  `json:"created_at"`             // Matches game_lobbies.created_at
	GameHasBegun         bool       `json:"game_has_begun"`         // Matches game_lobbies.game_has_begun
	IsPublic             bool       `json:"is_public"`              // Matches game_lobbies.is_public
	ShopState            *LobbyShop `json:"shop_state"`             // Matches game_lobbies.shop_state
	CurrentHighBlind     int        `json:"current_blind"`          // Matches game_lobbies.current_blind
	NumberOfVotes        int        `json:"number_of_votes"`        // Matches game_lobbies.number_of_votes
	HighestBlindProposer string     `json:"highest_blind_proposer"` // New field to track who proposed the highest blind

	// New fields
	CurrentRound int `json:"current_round"`

	// Replace counters with maps of usernames to track who has completed each action
	ProposedBlinds          map[string]bool `json:"proposed_blinds"`           // Map of usernames who have proposed blinds
	PlayersFinishedRound    map[string]bool `json:"players_finished_round"`    // Map of usernames who have finished the round
	PlayersFinishedShop     map[string]bool `json:"players_finished_shop"`     // Map of usernames who have finished shopping
	PlayersFinishedVouchers map[string]bool `json:"players_finished_vouchers"` // Map of usernames who have finished the vouchers phase

	PlayerCount int `json:"player_count"` // New field to track number of players

	// Replace single Timeout with specific timeouts
	BlindTimeout     time.Time `json:"blind_timeout"`
	GameRoundTimeout time.Time `json:"game_round_timeout"`
	ShopTimeout      time.Time `json:"shop_timeout"`
	VouchersTimeout  time.Time `json:"vouchers_timeout"`

	// Add current phase tracking
	CurrentPhase string `json:"current_phase"` // One of: none, blind, play_round, shop

	// Current base blind proposed by the game
	CurrentBaseBlind int `json:"current_base_blind"`
}

// CRITICAL: if maps were not initialized, they would be nil and cause panic
func (l *GameLobby) EnsureMapsInitialized() {
	if l.ProposedBlinds == nil {
		l.ProposedBlinds = make(map[string]bool)
	}
	if l.PlayersFinishedRound == nil {
		l.PlayersFinishedRound = make(map[string]bool)
	}
	if l.PlayersFinishedShop == nil {
		l.PlayersFinishedShop = make(map[string]bool)
	}
	if l.PlayersFinishedVouchers == nil {
		l.PlayersFinishedVouchers = make(map[string]bool)
	}
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
