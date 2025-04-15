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
func AnnounceWinner(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {
	log.Printf("[GAME-END] Game ending for lobby %s", lobbyID)

	// Set the game phase to announce winner
	if err := socketio_utils.SetGamePhase(redisClient, lobbyID, redis_models.AnnounceWinner); err != nil {
		log.Printf("[GAME-END-ERROR] Error setting game end phase: %v", err)
	}

	// Get all players to determine winner
	players, err := redisClient.GetAllPlayersInLobby(lobbyID)
	if err != nil {
		log.Printf("[GAME-END-ERROR] Error getting players: %v", err)
		return
	}

	// Track the highest score and all players who achieved it
	var highestPoints = -1
	var winners []*redis_models.InGamePlayer

	// First pass: find the highest score
	for i := range players {
		if players[i].CurrentPoints > highestPoints {
			highestPoints = players[i].CurrentPoints
		}
	}

	// Second pass: collect all players with the highest score
	for i := range players {
		if players[i].CurrentPoints == highestPoints {
			// Make a copy of the player to avoid pointer issues
			playerCopy := players[i]
			winners = append(winners, &playerCopy)
		}
	}

	// Prepare the winners data array
	winnersData := []gin.H{}

	// If no winners were found (should be impossible), provide a default
	if len(winners) == 0 {
		winnersData = append(winnersData, gin.H{
			"winner_username": "", // Empty username
			"points":          0,
			"icon":            1, // ANY icon
		})
	} else {
		// Add all winners to the response
		for _, winner := range winners {
			// Get the winner's icon from PostgreSQL database
			winnerIcon := utils.UserIcon(db, winner.Username)

			winnersData = append(winnersData, gin.H{
				"winner_username": winner.Username,
				"points":          winner.CurrentPoints,
				"icon":            winnerIcon,
			})

			log.Printf("[GAME-END] Winner: %s with %d points and icon %d",
				winner.Username, winner.CurrentPoints, winnerIcon)
		}

		if len(winners) > 1 {
			log.Printf("[GAME-END] The game ended in a %d-way tie with %d points each",
				len(winners), highestPoints)
		} else {
			log.Printf("[GAME-END] Winner is %s with %d points",
				winners[0].Username, winners[0].CurrentPoints)
		}
	}

	// Broadcast game end to all players
	sio.Sio_server.To(socket.Room(lobbyID)).Emit("game_end", gin.H{
		"winners": winnersData, // Now an array of winners
		"tie":     len(winners) > 1,
		"points":  highestPoints,
		"message": "The game has ended!",
	})

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
		if err := tx.Exec("DELETE FROM players_in_lobbies WHERE lobby_id = ?", lobbyID).Error; err != nil {
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
