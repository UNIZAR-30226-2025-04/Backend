package play_round

import (
	redis_models "Nogler/models/redis"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
)

// ---------------------------------------------------------------
// Functions that are executed to start the next game round
// ---------------------------------------------------------------

func PrepareRoundStart(redisClient *redis.RedisClient, lobbyID string) (*redis_models.GameLobby, int, error) {
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

func BroadcastRoundStart(sio *socketio_types.SocketServer, lobbyID string, round int, blind int) {
	log.Printf("[ROUND-BROADCAST] Broadcasting round start event for lobby %s", lobbyID)

	// Broadcast round start event to all players in the lobby
	sio.Sio_server.To(socket.Room(lobbyID)).Emit("starting_round", gin.H{
		"round_number": round,
		"blind":        blind,
	})

	log.Printf("[ROUND-BROADCAST] Sent round start event to lobby %s with round %d and blind %d",
		lobbyID, round, blind)
}
