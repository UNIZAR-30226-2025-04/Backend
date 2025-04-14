package socketio_utils

import (
	redis_models "Nogler/models/redis"
	"Nogler/services/redis"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
)

// ValidateGamePhase checks if the current game phase matches the expected phase
func ValidateGamePhase(redisClient *redis.RedisClient, client *socket.Socket, lobbyID string, expectedPhase string) (bool, error) {
	// Get the game lobby from Redis to check the current phase
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[PHASE-ERROR] Error getting lobby: %v", err)
		if client != nil {
			client.Emit("error", gin.H{"error": "Error checking game phase"})
		}
		return false, err
	}

	// Check if we're in the expected phase
	if lobby.CurrentPhase != expectedPhase {
		log.Printf("[PHASE-ERROR] Action attempted during wrong phase: %s (required: %s)",
			lobby.CurrentPhase, expectedPhase)
		if client != nil {
			client.Emit("error", gin.H{
				"error": fmt.Sprintf("This action is only allowed during the %s phase (current phase: %s)",
					expectedPhase, lobby.CurrentPhase),
			})
		}
		return false, nil
	}

	return true, nil
}

// ValidatePlayRoundPhase specifically validates that the game is in the play round phase
func ValidatePlayRoundPhase(redisClient *redis.RedisClient, client *socket.Socket, lobbyID string) (bool, error) {
	return ValidateGamePhase(redisClient, client, lobbyID, redis_models.PhasePlayRound)
}

// ValidateShopPhase specifically validates that the game is in the shop phase
func ValidateShopPhase(redisClient *redis.RedisClient, client *socket.Socket, lobbyID string) (bool, error) {
	return ValidateGamePhase(redisClient, client, lobbyID, redis_models.PhaseShop)
}

// ValidateBlindPhase specifically validates that the game is in the blind phase
func ValidateBlindPhase(redisClient *redis.RedisClient, client *socket.Socket, lobbyID string) (bool, error) {
	return ValidateGamePhase(redisClient, client, lobbyID, redis_models.PhaseBlind)
}

// ValidateModifiersPhase specifically validates that the game is in the modifiers phase
func ValidateModifiersPhase(redisClient *redis.RedisClient, client *socket.Socket, lobbyID string) (bool, error) {
	return ValidateGamePhase(redisClient, client, lobbyID, redis_models.PhaseModifiers)
}
