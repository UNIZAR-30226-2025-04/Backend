package handlers

import (
	models "Nogler/models/postgres"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	"fmt"

	"gorm.io/gorm"
)

// Function to handle socket.io client disconnections.
func HandleDisconnecting(username string, sio *socketio_types.SocketServer, redisClient *redis.RedisClient, db *gorm.DB) func(args ...interface{}) {
	return func(args ...interface{}) {
		// Remove connection from map
		sio.RemoveConnection(username)
		fmt.Println("A user just disconnected: ", username)
		fmt.Println("Current connections: ", sio.UserConnections)

		// Get the lobby ID of the disconnected player from Redis
		lobbyID, err := redisClient.GetPlayerCurrentLobby(username)
		if err != nil {
			fmt.Printf("[DISCONNECT-ERROR] Could not get lobby for user %s: %v\n", username, err)
			return
		}

		// Remove the player from Redis
		if err := redisClient.DeleteInGamePlayer(username, lobbyID); err != nil {
			fmt.Printf("[DISCONNECT-ERROR] Could not remove player %s from Redis: %v\n", username, err)
			return
		}

		// Check if there are other players in the lobby in PostgreSQL
		var playersInLobby []models.InGamePlayer
		if err := db.Where("lobby_id = ?", lobbyID).Find(&playersInLobby).Error; err != nil {
			fmt.Printf("[DISCONNECT-ERROR] Could not retrieve players for lobby %s: %v\n", lobbyID, err)
			return
		}

		// If no players are left, delete the lobby from PostgreSQL and Redis
		if len(playersInLobby) == 0 {
			fmt.Printf("[DISCONNECT] No players left in lobby %s. Deleting lobby...\n", lobbyID)

			// Delete lobby from PostgreSQL
			if err := db.Delete(&models.GameLobby{}, "id = ?", lobbyID).Error; err != nil {
				fmt.Printf("[DISCONNECT-ERROR] Could not delete lobby %s from PostgreSQL: %v\n", lobbyID, err)
				return
			}

			// Delete lobby from Redis
			if err := redisClient.DeleteGameLobby(lobbyID); err != nil {
				fmt.Printf("[DISCONNECT-ERROR] Could not delete lobby %s from Redis: %v\n", lobbyID, err)
				return
			}

			fmt.Printf("[DISCONNECT-SUCCESS] Lobby %s deleted successfully.\n", lobbyID)
		}

	}
}
