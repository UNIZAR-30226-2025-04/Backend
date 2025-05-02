package handlers

import (
	redis_models "Nogler/models/redis"
	"Nogler/services/redis"
	socketio_utils "Nogler/services/socket_io/utils"
	"Nogler/services/socket_io/utils/stages/play_round"
	"Nogler/services/socket_io/utils/stages/shop"
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

		// Get current jokers with sell prices
		var currentJokers poker.Jokers
		var jokersWithPrices []gin.H

		if player.CurrentJokers != nil && len(player.CurrentJokers) > 0 {
			if err := json.Unmarshal(player.CurrentJokers, &currentJokers); err == nil {
				// Calculate sell price for each joker
				for _, jokerID := range currentJokers.Juglares {
					if jokerID != 0 { // Skip empty slots
						jokersWithPrices = append(jokersWithPrices, gin.H{
							"id":         jokerID,
							"sell_price": poker.CalculateJokerSellPrice(jokerID),
						})
					}
				}
			} else {
				log.Printf("[PHASE-INFO-WARNING] Error parsing jokers: %v", err)
			}
		}

		var actualCurrentBet int
		if player.BetMinimumBlind {
			actualCurrentBet = lobby.CurrentBaseBlind
		} else {
			actualCurrentBet = lobby.CurrentHighBlind
		}

		// Create a response with comprehensive game and player state
		response := gin.H{
			// Game state information
			"phase":              lobby.CurrentPhase,
			"timeout":            phaseTimeout,
			"total_players":      lobby.PlayerCount,
			"current_round":      lobby.CurrentRound,
			"current_pot":        play_round.CalculatePotAmount(lobby.CurrentRound),
			"current_high_blind": lobby.CurrentHighBlind,
			"current_base_blind": lobby.CurrentBaseBlind,
			"max_rounds":         lobby.MaxRounds,

			// Player-specific state
			"player_data": gin.H{
				"username":      player.Username,
				"players_money": player.PlayersMoney,
				// NEW: include the blind the user bet to
				"actual_current_bet": actualCurrentBet,
				// TODO: see in_game_player.go
				// "remaining_deck_cards": player.PlayersRemainingCards,
				"current_hand": player.CurrentHand,
				// TODO, include additional joker information (should be enough with sell price as well)
				"current_jokers":    jokersWithPrices,
				"current_points":    player.CurrentRoundPoints,
				"total_points":      player.TotalGamePoints,
				"hand_plays_left":   player.HandPlaysLeft,
				"next_reroll_price": shop.GetRerollPrice(lobby), // player.Rerolls + 2, TODO, revisar si se hace con player o con shop state
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
			// KEY: remove player purchased items from the shop state
			shop.RemovePurchasedItems(lobby.ShopState, player)
			response["shop_items"] = lobby.ShopState
			response["players_finished_shop"] = len(lobby.PlayersFinishedShop)
		// NEW: vouchers phase info
		case redis_models.PhaseVouchers:
			response["player_vouchers"] = player.Modifiers
			response["player_active_vouchers"] = player.ActivatedModifiers
			response["player_received_vouchers"] = player.ReceivedModifiers
			response["players_finished_vouchers"] = len(lobby.PlayersFinishedVouchers)
		}

		log.Printf("[PHASE-INFO] Sending reconnection info to %s for phase %s, info: %v",
			username, lobby.CurrentPhase, response)

		// Send the comprehensive game state
		client.Emit("game_phase_player_info", response)

		log.Printf("[PHASE-INFO] Sent complete game info to %s for phase %s",
			username, lobby.CurrentPhase)
	}
}
