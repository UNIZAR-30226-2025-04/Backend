package play_round

import (
	redis_models "Nogler/models/redis"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	"Nogler/services/socket_io/utils/stages/shop"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

// Function to handle the end of a round
func HandleRoundEnd(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {
	log.Printf("[ROUND-END] Handling end of round for lobby %s", lobbyID)

	// Get the lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-END-ERROR] Error getting lobby: %v", err)
		return
	}

	// If GameRoundTimeout is zero, it means the round has already ended
	if lobby.GameRoundTimeout.IsZero() {
		log.Printf("[ROUND-END-INFO] Round already ended for lobby %s, skipping", lobbyID)
		return
	}

	// Reset the game round timeout to indicate round has ended
	lobby.GameRoundTimeout = time.Time{}

	// Update the current phase
	lobby.CurrentPhase = redis_models.PhaseShop

	// CRITICAL: save game lobby, we'll save it again in handlePlayerEliminations,
	// and before broadcasting `starting_shop` event to the players (avoid concurrency problems)
	err = redisClient.SaveGameLobby(lobby)
	if err != nil {
		log.Printf("[ROUND-END-ERROR] Error saving lobby with updated GameRoundTimeout and CurrentPhase: %v", err)
		return
	}

	// Process eliminations based on blind achievement
	_, err = handlePlayerEliminations(redisClient, lobbyID, sio)
	if err != nil {
		log.Printf("[ELIMINATION-ERROR] Error handling player eliminations: %v", err)
	}

	// Get updated lobby (player count might have changed)
	lobby, err = redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-END-ERROR] Error getting updated lobby: %v", err)
		return
	}

	// Initialize the shop phase
	shop, err := shop.InitializeShop(lobbyID, lobby.CurrentRound)
	if err != nil {
		log.Printf("[SHOP-INIT-ERROR] Error initializing shop: %v", err)
	} else {
		// Store shop state in lobby
		lobby.ShopState = shop

		// Reset shop-related counters
		lobby.TotalPlayersFinishedShop = 0

		// Save the updated lobby
		if err := redisClient.SaveGameLobby(lobby); err != nil {
			log.Printf("[ROUND-END-ERROR] Error saving lobby: %v", err)
		}

		// Broadcast shop start to all players
		sio.Sio_server.To(socket.Room(lobbyID)).Emit("starting_shop", gin.H{
			"shop": shop,
		})

		// Start the shop timeout
		startShopTimeout(redisClient, db, lobbyID, sio)
	}

	log.Printf("[ROUND-END] Round ended for lobby %s", lobbyID)
}
