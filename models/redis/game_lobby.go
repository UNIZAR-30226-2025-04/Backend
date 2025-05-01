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
	IsPublic             int        `json:"is_public"`              // Matches game_lobbies.is_public
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

	// New maps to keep track of the phases that have been completed and their corresponding rounds
	// Each map key is the round number, and the value is a boolean indicating completion
	// If the corresponding entry doesnt exist, it means the phase has not been completed
	BlindsCompleted     map[int]bool `json:"blinds_completed"`      // Map of rounds where blinds have been completed
	GameRoundsCompleted map[int]bool `json:"game_rounds_completed"` // Map of rounds where game rounds have been completed
	ShopsCompleted      map[int]bool `json:"shop_completed"`        // Map of rounds where shop has been completed
	VouchersCompleted   map[int]bool `json:"vouchers_completed"`    // Map of rounds where vouchers have been completed

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
	if l.BlindsCompleted == nil {
		l.BlindsCompleted = make(map[int]bool)
	}
	if l.GameRoundsCompleted == nil {
		l.GameRoundsCompleted = make(map[int]bool)
	}
	if l.ShopsCompleted == nil {
		l.ShopsCompleted = make(map[int]bool)
	}
	if l.VouchersCompleted == nil {
		l.VouchersCompleted = make(map[int]bool)
	}
}

type RerolledJokers struct {
	Jokers [3]ShopItem `json:"jokers"` // Rerolled jokers
}

type LobbyShop struct {
	Rerolls        int              `json:"reroll_count"`
	Rerolled       []RerolledJokers `json:"rerolled_items"` //Rerolls through the shop
	FixedPacks     []ShopItem       `json:"fixed_packs"`
	FixedModifiers []ShopItem       `json:"fixed_modifiers"`
	// RerollableItems []ShopItem       `json:"rerollable_items"` // IDK if its deprecated or not
	RerollSeed   uint64 `json:"reroll_seed"`
	NextUniqueId int    `json:"next_unique_id"` // Unique ID for the next item to be added to the shop
}

type ShopItem struct {
	ID         int          `json:"id"`
	Type       string       `json:"type"` // "card", "joker", "pack", "modifier"
	Price      int          `json:"price"`
	PackSeed   int64        `json:"pack_seed,omitempty"`
	Content    PackContents `gorm:"type:jsonb" json:"content"` // Directly store PackContents
	JokerId    int          `json:"joker_id,omitempty"`        // Only for joker type
	ModifierId int          `json:"modifier_id,omitempty"`     // Only for modifier type
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
