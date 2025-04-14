package socketio_utils

import (
	"Nogler/services/redis"
	"fmt"
	"log"
)

func IncrementGameRound(redisClient *redis.RedisClient, lobbyID string, incrementBy int) (int, error) {
	log.Printf("[ROUND-INCREMENT] Incrementing round for lobby %s by %d", lobbyID, incrementBy)

	// Get the game lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-INCREMENT-ERROR] Error getting lobby: %v", err)
		return 0, fmt.Errorf("error getting lobby: %v", err)
	}

	// Increment the current round
	lobby.CurrentRound += incrementBy

	// Save the updated lobby back to Redis
	err = redisClient.SaveGameLobby(lobby)
	if err != nil {
		log.Printf("[ROUND-INCREMENT-ERROR] Error saving lobby: %v", err)
		return 0, fmt.Errorf("error saving lobby: %v", err)
	}

	log.Printf("[ROUND-INCREMENT-SUCCESS] Lobby %s round incremented to %d",
		lobbyID, lobby.CurrentRound)

	return lobby.CurrentRound, nil
}

func SetGamePhase(redisClient *redis.RedisClient, lobbyID string, newPhase string) error {
	log.Printf("[PHASE-CHANGE] Setting lobby %s phase to %s", lobbyID, newPhase)

	// Get the game lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[PHASE-CHANGE-ERROR] Error getting lobby: %v", err)
		return fmt.Errorf("error getting lobby: %v", err)
	}

	// Check if phase is already set to the requested value
	if lobby.CurrentPhase == newPhase {
		log.Printf("[PHASE-CHANGE-INFO] Lobby %s phase already set to %s", lobbyID, newPhase)
		return nil
	}

	// Update the phase
	lobby.CurrentPhase = newPhase

	// Save the updated lobby back to Redis
	err = redisClient.SaveGameLobby(lobby)
	if err != nil {
		log.Printf("[PHASE-CHANGE-ERROR] Error saving lobby with updated phase: %v", err)
		return fmt.Errorf("error saving lobby with updated phase: %v", err)
	}

	log.Printf("[PHASE-CHANGE-SUCCESS] Lobby %s phase changed to %s", lobbyID, newPhase)
	return nil
}
