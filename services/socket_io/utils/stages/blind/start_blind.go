package blind

import (
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------
// Functions that are executed to start the next blind
// ---------------------------------------------------------------

func BroadcastStartingNextBlind(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer, timeout int) {
	// Get the game lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[NEXT-BLIND-ERROR] Error getting lobby info: %v", err)
		return
	}

	// Broadcast starting_next_blind event to all players in the lobby
	sio.Sio_server.To(socket.Room(lobbyID)).Emit("starting_next_blind", gin.H{
		"lobby_id":     lobbyID,
		"blind_number": lobby.CurrentRound,
		"base_blind":   lobby.CurrentBaseBlind,
		"timeout":      timeout,
		"message":      "Starting the blind proposal phase!",
	})

	log.Printf("[NEXT-BLIND] Broadcast starting_next_blind event to lobby %s for round %d",
		lobbyID, lobby.CurrentRound)
}
