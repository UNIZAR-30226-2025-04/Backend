package handlers

import (
	models "Nogler/models/postgres"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"

	"github.com/zishang520/socket.io/v2/socket"

	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Function to handle socket.io client disconnections.
func HandleDisconnecting(username string, sio *socketio_types.SocketServer,
	db *gorm.DB, redisClient *redis.RedisClient) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("[DISCONNECT] HandleDisconnecting iniciado - Usuario: %s", username)

		// Get the client socket for room operations
		client, exists := sio.GetConnection(username)

		// Find all lobbies the user is in
		var userInLobbies []models.InGamePlayer
		if err := db.Where("username = ?", username).Find(&userInLobbies).Error; err != nil {
			log.Printf("[DISCONNECT-ERROR] Error finding user's lobbies: %v", err)
		} else {
			log.Printf("[DISCONNECT] Usuario %s está en %d lobbies", username, len(userInLobbies))

			// Process each lobby the user is in
			for _, inGamePlayer := range userInLobbies {
				lobbyID := inGamePlayer.LobbyID
				log.Printf("[DISCONNECT] Removiendo usuario %s del lobby %s", username, lobbyID)

				// Start transaction
				tx := db.Begin()
				if tx.Error != nil {
					log.Printf("[DISCONNECT-ERROR] Error iniciando transacción: %v", tx.Error)
					continue
				}

				// Delete from PostgreSQL
				if err := tx.Delete(&inGamePlayer).Error; err != nil {
					tx.Rollback()
					log.Printf("[DISCONNECT-ERROR] Error eliminando usuario de Postgres: %v", err)
					continue
				}

				// Delete from Redis
				if redisClient != nil {
					if err := redisClient.DeleteInGamePlayer(username, lobbyID); err != nil {
						tx.Rollback()
						log.Printf("[DISCONNECT-ERROR] Error eliminando usuario de Redis: %v", err)
						continue
					}
				}

				// Commit transaction
				if err := tx.Commit().Error; err != nil {
					log.Printf("[DISCONNECT-ERROR] Error en commit: %v", err)
					continue
				}

				// Leave the room if client exists
				if exists {
					client.Leave(socket.Room(lobbyID))
				}

				// Notify other players in the lobby about the disconnection
				sio.Sio_server.To(socket.Room(lobbyID)).Emit("player_left", gin.H{
					"username": username,
					"lobby_id": lobbyID,
					"reason":   "disconnected",
				})

				log.Printf("[DISCONNECT-SUCCESS] Usuario %s removido del lobby %s", username, lobbyID)
			}
		}

		// Finally remove connection from map
		sio.RemoveConnection(username)
		log.Printf("[DISCONNECT-DONE] Usuario desconectado: %s", username)
	}
}
