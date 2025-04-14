package play_round

import (
	redis_models "Nogler/models/redis"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------
// Functions that are executed to start the next game round
// ---------------------------------------------------------------

func AdvanceToNextRoundPlay(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {
	log.Printf("[ROUND-PLAY-ADVANCE] Advancing to round play phase for lobby %s", lobbyID)

	// Get the lobby to check if round already started
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-PLAY-ADVANCE-ERROR] Error getting lobby info: %v", err)
		return
	}

	// Early return if blind timeout is already reset (round already started)
	if lobby.BlindTimeout.IsZero() {
		log.Printf("[ROUND-PLAY-ADVANCE-INFO] Round already started for lobby %s, skipping", lobbyID)
		return
	}

	// Step 1: Prepare the round state in Redis
	updatedLobby, blind, err := prepareRoundStart(redisClient, lobbyID)
	if err != nil {
		log.Printf("[ROUND-PLAY-ADVANCE-ERROR] Failed to prepare round: %v", err)
		return
	}

	// Step 2: Broadcast round start event
	broadcastRoundStart(sio, lobbyID, updatedLobby.CurrentRound, blind)

	// Step 3: Start the round play timeout
	startRoundPlayTimeout(redisClient, db, lobbyID, sio)

	log.Printf("[ROUND-PLAY-ADVANCE-SUCCESS] Advanced lobby %s to round play phase", lobbyID)
}

func prepareRoundStart(redisClient *redis.RedisClient, lobbyID string) (*redis_models.GameLobby, int, error) {
	log.Printf("[ROUND-PREPARE] Preparing round start state for lobby %s", lobbyID)

	// Get the game lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-PREPARE-ERROR] Error getting lobby info: %v", err)
		return nil, 0, err
	}

	// Reset players finished round counter in redis
	lobby.TotalPlayersFinishedRound = 0

	// Reset the blind timeout to indicate round has started
	lobby.BlindTimeout = time.Time{}

	// Set the current phase to play round
	lobby.CurrentPhase = redis_models.PhasePlayRound

	// Get the blind value
	blind := lobby.CurrentBlind

	// CRITICAL: Save the updated lobby state BEFORE broadcasting
	err = redisClient.SaveGameLobby(lobby)
	if err != nil {
		log.Printf("[ROUND-PREPARE-ERROR] Error updating lobby state: %v", err)
		return nil, 0, err
	}

	log.Printf("[ROUND-PREPARE-SUCCESS] Lobby %s prepared for round start with blind %d",
		lobbyID, blind)

	return lobby, blind, nil
}

func broadcastRoundStart(sio *socketio_types.SocketServer, lobbyID string, round int, blind int) {
	log.Printf("[ROUND-BROADCAST] Broadcasting round start event for lobby %s", lobbyID)

	// Broadcast round start event to all players in the lobby
	sio.Sio_server.To(socket.Room(lobbyID)).Emit("starting_round", gin.H{
		"round_number": round,
		"blind":        blind,
	})

	log.Printf("[ROUND-BROADCAST] Sent round start event to lobby %s with round %d and blind %d",
		lobbyID, round, blind)
}

func startRoundPlayTimeout(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {
	log.Printf("[ROUND-PLAY-TIMEOUT] Starting round play timeout for lobby %s", lobbyID)

	// Get the game lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-PLAY-ERROR] Error getting lobby info: %v", err)
		return
	}

	// Check if the round is already in timeout
	// NOTE: SHOULDN'T HAPPEN
	if !lobby.GameRoundTimeout.IsZero() {
		log.Printf("[ROUND-PLAY-ERROR] Round is already in timeout: %v", lobby.GameRoundTimeout)
		return
	}

	// Reset the round-related counters
	lobby.TotalPlayersFinishedRound = 0

	// Set the game round timeout to the current time
	lobby.GameRoundTimeout = time.Now()
	err = redisClient.SaveGameLobby(lobby)
	if err != nil {
		log.Printf("[ROUND-PLAY-ERROR] Error setting lobby round timeout: %v", err)
		return
	}

	// Start a goroutine to handle the timeout
	go func() {
		// Sleep for the round duration (e.g., 2 minutes)
		time.Sleep(2 * time.Minute)

		// Call the function to handle round end
		HandleRoundEnd(redisClient, db, lobbyID, sio)
	}()

	log.Printf("[ROUND-PLAY-TIMEOUT] Round play timeout started for lobby %s", lobbyID)
}
