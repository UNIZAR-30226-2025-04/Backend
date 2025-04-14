package handlers

import (
	redis_models "Nogler/models/redis"
	"Nogler/services/redis"
	socketio_utils "Nogler/services/socket_io/utils"
	"log"
	"time"

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

		// Create a response with comprehensive game and player state
		response := gin.H{
			// Game state information
			"phase":         lobby.CurrentPhase,
			"timeout":       phaseTimeout,
			"total_players": lobby.PlayerCount,
			"current_round": lobby.CurrentRound,
			"current_blind": lobby.CurrentBlind,

			// Player-specific state
			"player_data": gin.H{
				"username":        player.Username,
				"players_money":   player.PlayersMoney,
				"current_deck":    player.CurrentDeck,
				"modifiers":       player.Modifiers,
				"current_jokers":  player.CurrentJokers,
				"current_points":  player.CurrentPoints,
				"total_points":    player.TotalPoints,
				"hand_plays_left": player.HandPlaysLeft,
				"discards_left":   player.DiscardsLeft,
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
		}

		// Send the comprehensive game state
		client.Emit("game_phase_info", response)

		log.Printf("[PHASE-INFO] Sent complete game info to %s for phase %s",
			username, lobby.CurrentPhase)
	}
}
