package handlers

import (
	redis_models "Nogler/models/redis"
	"Nogler/services/redis"
	socketio_utils "Nogler/services/socket_io/utils"
	"encoding/json"
	"log"
	"time"

	"Nogler/services/poker"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

func HandleRequestGamePhaseInfo(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {
		if len(args) < 1 {
			client.Emit("error", gin.H{"error": "Lobby ID is required"})
			return
		}

		lobbyID := args[0].(string)
		log.Printf("[PHASE-INFO-REQUEST] Requesting phase info for lobby %s by user %s", lobbyID, username)

		// Validate the user and lobby
		lobby, err := socketio_utils.ValidateLobbyAndUser(redisClient, client, db, username, lobbyID)
		if err != nil {
			return
		}

		// Get player-specific data from Redis
		player, err := redisClient.GetInGamePlayer(username)
		if err != nil {
			log.Printf("[PHASE-INFO-ERROR] Error getting player data: %v", err)
			client.Emit("error", gin.H{"error": "Error retrieving player data"})
			return
		}

		// Get current deck
		var deck *poker.Deck
		if player.CurrentDeck != nil {
			deck, err = poker.DeckFromJSON(player.CurrentDeck)
			if err != nil {
				log.Printf("[DISCARD-ERROR] Error parsing deck: %v", err)
				client.Emit("error", gin.H{"error": "Error al procesar el mazo"})
				return
			}
		}

		// Determine which timeout to return based on the current phase
		var phaseTimeout time.Time
		switch lobby.CurrentPhase {
		case redis_models.PhaseBlind:
			phaseTimeout = lobby.BlindTimeout
		case redis_models.PhasePlayRound:
			phaseTimeout = lobby.GameRoundTimeout
		case redis_models.PhaseShop:
			phaseTimeout = lobby.ShopTimeout
		default:
			phaseTimeout = time.Time{} // Zero time for other phases
		}

		// Get the current hand from the player
		var currentHand []poker.Card
		err = json.Unmarshal(player.CurrentHand, &currentHand)
		if err != nil {
			log.Printf("[HAND-ERROR] Error unmarshaling current hand: %v", err)
			client.Emit("error", gin.H{"error": "Error processing current hand"})
			return
		}

		// Create a response with comprehensive game and player state
		response := gin.H{
			// Game state information
			"phase":              lobby.CurrentPhase,
			"timeout":            phaseTimeout,
			"total_players":      lobby.PlayerCount,
			"current_round":      lobby.CurrentRound,
			"current_pot":        lobby.CurrentRound + lobby.CurrentRound/2 + 1,
			"current_high_blind": lobby.CurrentHighBlind,
			"current_base_blind": lobby.CurrentBaseBlind,
			"max_rounds":         lobby.MaxRounds,

			// Player-specific state
			"player_data": gin.H{
				"username":      player.Username,
				"players_money": player.PlayersMoney,
				// TODO: see in_game_player.go
				// "remaining_deck_cards": player.PlayersRemainingCards,
				"current_hand":      player.CurrentHand,
				"modifiers":         player.Modifiers,
				"current_jokers":    player.CurrentJokers,
				"current_points":    player.CurrentPoints,
				"total_points":      player.TotalPoints,
				"hand_plays_left":   player.HandPlaysLeft,
				"discards_left":     player.DiscardsLeft,
				"played_cards":      len(deck.PlayedCards),
				"unplayed_cards":    len(deck.TotalCards) + len(currentHand),
				"vouchers":          player.Modifiers,
				"active_vouchers":   player.ActivatedModifiers,
				"received_vouchers": player.ReceivedModifiers,
			},
		}

		// Add phase-specific information
		switch lobby.CurrentPhase {
		case redis_models.PhaseBlind:
			response["total_proposals"] = len(lobby.ProposedBlinds)
		case redis_models.PhasePlayRound:
			response["players_finished_round"] = len(lobby.PlayersFinishedRound)
		case redis_models.PhaseShop:
			response["shop_items"] = lobby.ShopState
			response["players_finished_shop"] = len(lobby.PlayersFinishedShop)
		// NEW: vouchers phase info
		case redis_models.PhaseVouchers:
			response["player_vouchers"] = player.Modifiers
			response["player_active_vouchers"] = player.ActivatedModifiers
			response["player_received_vouchers"] = player.ReceivedModifiers
			response["players_finished_vouchers"] = len(lobby.PlayersFinishedVouchers)
		}

		// Send the comprehensive game state
		client.Emit("game_phase_player_info", response)

		log.Printf("[PHASE-INFO] Sent complete game info to %s for phase %s",
			username, lobby.CurrentPhase)
	}
}
