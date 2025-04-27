package shop

import (
	game_constants "Nogler/constants/game"
	"Nogler/models/redis"
	"Nogler/services/poker"
	redis_services "Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	"encoding/json"
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
		// KEY: Reset the last purchased pack ID to prevent exploits between rounds
		player.LastPurchasedPackItemId = -1

		// Save the updated player state
		err := redisClient.SaveInGamePlayer(&player)
		if err != nil {
			log.Printf("[SHOP-MULTICAST-WARNING] Failed to reset LastPurchasedPackItemId for player %s: %v",
				player.Username, err)
		}

		// Get player's socket using GetConnection
		playerSocket, exists := sio.GetConnection(player.Username)
		if !exists {
			log.Printf("[SHOP-MULTICAST-WARNING] Player %s has no active connection", player.Username)
			continue
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
				log.Printf("[SHOP-MULTICAST-WARNING] Error parsing jokers: %v", err)
			}
		}

		// Send personalized message to this player
		playerSocket.Emit("starting_shop", gin.H{
			"shop":               shopItems,
			"timeout":            timeout,
			"timeout_start_date": lobby.ShopTimeout.Format(time.RFC3339),
			"current_round":      lobby.CurrentRound,
			"money":              player.PlayersMoney,
			"players_jokers":     jokersWithPrices,
			"max_jokers":         game_constants.MaxJokersPerPlayer,
		})

		log.Printf("[SHOP-MULTICAST] Sent personalized shop data to player %s", player.Username)
	}

	log.Printf("[SHOP-MULTICAST] Shop start broadcast completed for lobby %s", lobbyID)
}
