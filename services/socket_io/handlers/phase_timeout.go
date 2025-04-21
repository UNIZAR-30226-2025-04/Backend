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

// HandleGetPhaseTimeout responds with the current game phase and its associated timeout
func HandleGetPhaseTimeout(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {
		// Check if we have enough arguments
		if len(args) < 1 {
			log.Printf("[PHASE-TIMEOUT-ERROR] Missing lobby ID for user %s", username)
			client.Emit("error", gin.H{"error": "Missing lobby ID"})
			return
		}

		lobbyID := args[0].(string)
		log.Printf("[PHASE-TIMEOUT] User %s requesting phase timeout for lobby %s", username, lobbyID)

		// Validate the user and lobby
		lobby, err := socketio_utils.ValidateLobbyAndUser(redisClient, client, db, username, lobbyID)
		if err != nil {
			// Error already emitted in ValidateLobbyAndUser
			return
		}

		// Determine which timeout to return based on the current phase
		var phaseTimeout string
		switch lobby.CurrentPhase {
		case redis_models.PhaseBlind:
			phaseTimeout = lobby.BlindTimeout.Format(time.RFC3339)
		case redis_models.PhasePlayRound:
			phaseTimeout = lobby.GameRoundTimeout.Format(time.RFC3339)
		case redis_models.PhaseShop:
			phaseTimeout = lobby.ShopTimeout.Format(time.RFC3339)
		default:
			phaseTimeout = "" // Empty string for other phases
		}

		// Send the phase and timeout information to the client
		client.Emit("phase_timeout_info", gin.H{
			"phase":         lobby.CurrentPhase,
			"timeout":       phaseTimeout,
			"current_round": lobby.CurrentRound,
		})

		log.Printf("[PHASE-TIMEOUT] Sent phase timeout info to user %s: phase=%s, timeout=%s",
			username, lobby.CurrentPhase, phaseTimeout)
	}
}
