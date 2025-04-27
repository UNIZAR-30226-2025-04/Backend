package end_game

import (
	redis_models "Nogler/models/redis"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	socketio_utils "Nogler/services/socket_io/utils"
	"Nogler/utils"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

// Function to handle game end and announce winner
func AnnounceWinners(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {
	log.Printf("[GAME-END] Game ending for lobby %s", lobbyID)

	// Set the game phase to announce winner
	if err := socketio_utils.SetGamePhase(redisClient, lobbyID, redis_models.AnnounceWinner); err != nil {
		log.Printf("[GAME-END-ERROR] Error setting game end phase: %v", err)
	}

	// Get all players to determine winner
	players, err := redisClient.GetAllPlayersInLobby(lobbyID)
	if err != nil {
		log.Printf("[GAME-END-ERROR] Error getting players: %v", err)
		// Even if there's an error, continue with empty players list
		players = []redis_models.InGamePlayer{}
	}

	// Get the lobby to check player count
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[GAME-END-ERROR] Error getting lobby: %v", err)
		// Continue with available players
	}

	// Track the highest score and all players who achieved it
	var highestPoints = -1
	var winners []*redis_models.InGamePlayer

	// Prepare the winners data array
	winnersData := []gin.H{}

	// If no players remaining, handle the "all eliminated" case
	if len(players) == 0 || (err == nil && lobby.PlayerCount == 0) {
		log.Printf("[GAME-END] No winners for lobby %s (all players eliminated)", lobbyID)

		// Emit game end event with no winners
		sio.Sio_server.To(socket.Room(lobbyID)).Emit("game_end", gin.H{
			"winners":    winnersData, // Empty array
			"tie":        false,
			"points":     0,
			"no_winners": true, // Flag to indicate all players were eliminated
			"message":    "The game has ended! All players were eliminated.",
		})
	} else {
		// Normal game end with winners

		// First pass: find the highest score
		for i := range players {
			if players[i].CurrentRoundPoints > highestPoints {
				highestPoints = players[i].CurrentRoundPoints
			}
		}

		// Second pass: collect all players with the highest score
		for i := range players {
			if players[i].CurrentRoundPoints == highestPoints {
				// Make a copy of the player to avoid pointer issues
				playerCopy := players[i]
				winners = append(winners, &playerCopy)
			}
		}

		// Add all winners to the response
		for _, winner := range winners {
			// Get the winner's icon from PostgreSQL database
			winnerIcon := utils.UserIcon(db, winner.Username)

			winnersData = append(winnersData, gin.H{
				"winner_username": winner.Username,
				"points":          winner.CurrentRoundPoints,
				"icon":            winnerIcon,
			})

			log.Printf("[GAME-END] Winner: %s with %d points and icon %d",
				winner.Username, winner.CurrentRoundPoints, winnerIcon)
		}

		if len(winners) > 1 {
			// NOTE: RN, the only way to get > 1 winners is to end the game because of round limit
			log.Printf("[GAME-END] The game ended in a %d-way tie with %d points each",
				len(winners), highestPoints)
		} else {
			log.Printf("[GAME-END] Winner is %s with %d points",
				winners[0].Username, winners[0].CurrentRoundPoints)
		}

		// Broadcast game end to all players
		sio.Sio_server.To(socket.Room(lobbyID)).Emit("game_end", gin.H{
			"winners":    winnersData,
			"tie":        len(winners) > 1,
			"points":     highestPoints,
			"no_winners": false,
			"message":    "The game has ended!",
		})
	}

	log.Printf("[GAME-END] Game ended for lobby %s", lobbyID)
}

// CleanupGame deletes all game-related data from Redis and PostgreSQL after the game ends
func CleanupGame(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string) {
	log.Printf("[GAME-CLEANUP] Starting cleanup for lobby %s", lobbyID)

	// 1. Get all players in the lobby from Redis
	players, err := redisClient.GetAllPlayersInLobby(lobbyID)
	if err != nil {
		log.Printf("[GAME-CLEANUP-ERROR] Error getting players: %v", err)
		// Continue with cleanup even if we can't get all players
	}

	// 2. Delete each player's game data from Redis
	for _, player := range players {
		if err := redisClient.DeleteInGamePlayer(player.Username, lobbyID); err != nil {
			log.Printf("[GAME-CLEANUP-ERROR] Error deleting player %s from Redis: %v",
				player.Username, err)
		} else {
			log.Printf("[GAME-CLEANUP] Deleted player %s from Redis", player.Username)
		}
	}

	// 3. Delete the game lobby from Redis
	if err := redisClient.DeleteGameLobby(lobbyID); err != nil {
		log.Printf("[GAME-CLEANUP-ERROR] Error deleting lobby %s from Redis: %v",
			lobbyID, err)
	} else {
		log.Printf("[GAME-CLEANUP] Deleted lobby %s from Redis", lobbyID)
	}

	// 4. Use a transaction to delete PostgreSQL data
	err = db.Transaction(func(tx *gorm.DB) error {
		// First remove all player-lobby relationships
		if err := tx.Exec("DELETE FROM in_game_players WHERE lobby_id = ?", lobbyID).Error; err != nil {
			log.Printf("[GAME-CLEANUP-ERROR] Error deleting player relationships: %v", err)
			return err
		}

		// Then delete the lobby itself
		if err := tx.Exec("DELETE FROM game_lobbies WHERE id = ?", lobbyID).Error; err != nil {
			log.Printf("[GAME-CLEANUP-ERROR] Error deleting lobby from PostgreSQL: %v", err)
			return err
		}

		return nil
	})

	if err != nil {
		log.Printf("[GAME-CLEANUP-ERROR] Transaction failed: %v", err)
	} else {
		log.Printf("[GAME-CLEANUP-SUCCESS] Successfully removed lobby %s and all related data from databases", lobbyID)
	}
}
