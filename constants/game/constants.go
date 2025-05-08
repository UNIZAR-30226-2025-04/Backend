package game_constants

const MaxGameRounds = 10
const MaxJokersPerPlayer = 5 // NOTE: This is what frontend uses
const TOTAL_HAND_PLAYS = 3
const TOTAL_DISCARDS = 3
const BASE_BLIND = 10
const ROUND_BLIND_MULTIPLIER = 3
const MAX_BLIND = 1e6

// Shop constants
const (
	// Pack types (1-3) - Used to identify the type of pack
	PACK_TYPE_CARDS    = 1 // Contains regular playing cards
	PACK_TYPE_JOKERS   = 2 // Contains joker cards with special abilities
	PACK_TYPE_VOUCHERS = 3 // Contains game modifiers/vouchers
)

// Modifier type constants
const MODIFIER_TYPE = "modifier"
const JOKER_TYPE = "joker"
const PACK_TYPE = "pack"

// "current_pot":        lobby.CurrentRound + lobby.CurrentRound/2 + 1,
