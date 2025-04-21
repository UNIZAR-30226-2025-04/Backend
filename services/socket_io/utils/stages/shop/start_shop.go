package shop

import (
	"Nogler/models/redis"
	redis_services "Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------
// Functions that are executed to start the shop phase
// ---------------------------------------------------------------

func MulticastStartingShop(sio *socketio_types.SocketServer, redisClient *redis_services.RedisClient, lobbyID string, shopItems *redis.LobbyShop, timeout int) {
	log.Printf("[SHOP-MULTICAST] Broadcasting shop start for lobby %s", lobbyID)

	// Get the lobby to access the current round
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[SHOP-MULTICAST-ERROR] Error getting lobby data: %v", err)
		return
	}

	// Get all players in the lobby
	players, err := redisClient.GetAllPlayersInLobby(lobbyID)
	if err != nil {
		log.Printf("[SHOP-MULTICAST-ERROR] Error getting players: %v", err)
		return
	}

	// Send personalized message to each player
	for _, player := range players {
		// Get player's socket using GetConnection
		playerSocket, exists := sio.GetConnection(player.Username)
		if !exists {
			log.Printf("[SHOP-MULTICAST-WARNING] Player %s has no active connection", player.Username)
			continue
		}

		// Send personalized message to this player
		playerSocket.Emit("starting_shop", gin.H{
			"shop":               shopItems,
			"timeout":            timeout,
			"timeout_start_date": lobby.ShopTimeout.Format(time.RFC3339),
			"current_round":      lobby.CurrentRound,
			"money":              player.PlayersMoney,
			"jokers":             player.CurrentJokers,
		})

		log.Printf("[SHOP-MULTICAST] Sent personalized shop data to player %s", player.Username)
	}

	log.Printf("[SHOP-MULTICAST] Shop start broadcast completed for lobby %s", lobbyID)
}
