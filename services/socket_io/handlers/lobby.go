package handlers

import (
	models "Nogler/models/postgres"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	"Nogler/utils"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

/*
 * IMPORTANT NOTE: all the checks on whether a lobby exists, a user is in a lobby or not, etc
 * must be made through requests to the Postgres database, NOT to the redis database. Check
 * /controllers/lobby.go endpoints to see it.
 * The redis database is used specifically to store the state of the game.
 */

// Function to get information about all users in a lobby.
func GetLobbyInfo(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("[INFO] GetLobbyInfo iniciado - Usuario: %s, Args: %v", username, args)

		// Validate arguments
		if len(args) < 1 {
			log.Printf("[INFO-ERROR] Faltan argumentos para usuario %s", username)
			client.Emit("error", gin.H{"error": "Falta el ID del lobby"})
			return
		}

		lobbyID := args[0].(string)
		log.Printf("[INFO] Obteniendo información del lobby ID: %s para usuario: %s", lobbyID, username)

		// Check if lobby exists
		var lobby models.GameLobby
		if err := db.Where("id = ?", lobbyID).First(&lobby).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				client.Emit("error", gin.H{"error": "Lobby not found"})
			} else {
				client.Emit("error", gin.H{"error": "Database error"})
			}
			return
		}

		// Get all players in the lobby
		var players []models.InGamePlayer
		if err := db.Where("lobby_id = ?", lobbyID).Find(&players).Error; err != nil {
			client.Emit("error", gin.H{"error": "Error retrieving players"})
			return
		}

		// Get icons for all players
		playerInfos := make([]gin.H, 0, len(players))
		for _, player := range players {
			// Get player icon
			var gameProfile models.GameProfile
			if err := db.Where("username = ?", player.Username).First(&gameProfile).Error; err != nil {
				log.Printf("[INFO-WARNING] No se pudo obtener el perfil para %s", player.Username)
				continue
			}

			playerInfos = append(playerInfos, gin.H{
				"username":  player.Username,
				"user_icon": gameProfile.UserIcon,
			})
		}

		// Get creator information
		var creatorProfile models.GameProfile
		if err := db.Where("username = ?", lobby.CreatorUsername).First(&creatorProfile).Error; err != nil {
			client.Emit("error", gin.H{"error": "Error retrieving creator info"})
			return
		}

		// Return the complete lobby info
		client.Emit("lobby_info", gin.H{
			"players": playerInfos,
			"creator": gin.H{
				"username":  lobby.CreatorUsername,
				"user_icon": creatorProfile.UserIcon,
			},
			"lobby_id": lobbyID,
		})

		log.Printf("[INFO-SUCCESS] Información del lobby %s enviada a usuario %s", lobbyID, username)
	}
}

// Function to handle the act of joining a lobby. The code will check whether the user has
// requested to join the lobby via the API (querying the Postgres database) and if the lobby
// exists in Redis. If both checks are positive, the client will be automatically joined to the
// socket.io room corresponding to that lobby, and the Redis info about the player will be updated
// (a new `InGamePlayer` object will be inserted).
func HandleJoinLobby(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("[JOIN] HandleJoinLobby iniciado - Usuario: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())

		if len(args) < 1 {
			log.Printf("[JOIN-ERROR] Faltan argumentos para usuario %s", username)
			client.Emit("error", gin.H{"error": "Falta el ID del lobby"})
			return
		}

		// 1. Parse lobby ID from arguments
		lobbyID := args[0].(string)
		log.Printf("[JOIN] Procesando lobby ID: %s para usuario: %s", lobbyID, username)

		// 2. Check if lobby exists in Postgres
		isInLobby, err := utils.UserExistsInLobby(db, lobbyID, username, client)
		if err != nil {
			return
		}

		// NOTE: si el mismo usuario vuelve a intentar conectarse otra vez sin haberse desconectado
		// se llamará otra vez a client.Join(socket.Room(lobbyID)) y en principio no habrá problema
		// (la llamada no tendrá ningún efecto, idempotencia)
		if !isInLobby {
			fmt.Println("User is NOT in lobby:", username, "Lobby:", lobbyID)
			client.Emit("error", gin.H{"error": "You must join the lobby before sending messages"})
			return
		}

		// 3. Join the socket to the room corresponding to the lobby id
		client.Join(socket.Room(lobbyID))
		log.Println("Jugador unido al room:", lobbyID)

		// 4. Notify success
		log.Printf("[JOIN-SUCCESS] Usuario %s unido exitosamente al lobby %s", username, lobbyID)

		// 5. Get user icon from PostgreSQL
		icon := utils.UserIcon(db, username)

		// 6. Emit success event to the client
		log.Printf("[JOIN-SUCCESS] Usuario %s unido exitosamente al lobby %s", username, lobbyID)
		client.Emit("joined_lobby", gin.H{
			"lobby_id":  lobbyID,
			"username":  username,
			"user_icon": icon,
			"message":   "¡Bienvenido al lobby!",
		})

		var profile models.GameProfile
		if err := db.Where("username = ?", username).First(&profile).Error; err != nil {
			log.Println("Error al obtener GameProfile:", err)
			client.Emit("error", gin.H{"error": "Error al obtener el perfil del jugador"})
			return
		}

		sio.Sio_server.To(socket.Room(lobbyID)).Emit("new_user_in_lobby", gin.H{
			"lobby_id": lobbyID,
			"username": username,
			"icon":     profile.UserIcon,
		})
	}
}

func HandleExitLobby(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("[EXIT] HandleExitLobby iniciado - Usuario: %s, Args: %v", username, args)

		// Validate arguments
		if len(args) < 1 {
			log.Printf("[EXIT-ERROR] Faltan argumentos para usuario %s", username)
			client.Emit("error", gin.H{"error": "Falta el ID del lobby"})
			return
		}

		lobbyID := args[0].(string)
		log.Printf("[EXIT] Procesando salida del lobby ID: %s para usuario: %s", lobbyID, username)

		// Check if lobby exists
		var lobby models.GameLobby
		result := db.Where("id = ?", lobbyID).First(&lobby)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				client.Emit("error", gin.H{"error": "Lobby not found"})
			} else {
				client.Emit("error", gin.H{"error": "Database error"})
			}
			return
		}

		// Check if user is in lobby
		var userInLobby models.InGamePlayer
		result = db.Where(
			"lobby_id = ? AND username = ?",
			lobbyID, username,
		).First(&userInLobby)

		if result.RowsAffected == 0 {
			client.Emit("error", gin.H{"error": "User is not in that lobby"})
			return
		}

		// Start transaction
		tx := db.Begin()
		if tx.Error != nil {
			client.Emit("error", gin.H{"error": "Database error starting transaction"})
			return
		}

		// Delete the player from lobby in PostgreSQL
		if err := tx.Delete(&userInLobby).Error; err != nil {
			tx.Rollback()
			client.Emit("error", gin.H{"error": "Error removing user from lobby"})
			return
		}

		// Remove player from Redis if exists
		if redisClient != nil {
			if err := redisClient.DeleteInGamePlayer(username, lobbyID); err != nil {
				tx.Rollback()
				client.Emit("error", gin.H{"error": "Error removing user from Redis"})
				return
			}
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			client.Emit("error", gin.H{"error": "Error committing transaction"})
			return
		}

		// Leave the socket.io room
		client.Leave(socket.Room(lobbyID))

		// Check if there are no more players in the lobby
		var playersInLobby []models.InGamePlayer
		if err := db.Where("lobby_id = ?", lobbyID).Find(&playersInLobby).Error; err != nil {
			client.Emit("error", gin.H{"error": "Error retrieving players in lobby"})
			return
		}

		// Notify success
		log.Printf("[EXIT-SUCCESS] Usuario %s ha salido exitosamente del lobby %s", username, lobbyID)
		client.Emit("exited_lobby", gin.H{
			"lobby_id": lobbyID,
			"message":  "Has salido del lobby exitosamente",
		})

		// If there are no more players, delete the lobby from PostgreSQL and Redis
		if len(playersInLobby) == 0 {
			log.Printf("[EXIT] No players left in lobby %s. Deleting lobby...", lobbyID)
			// Delete lobby from PostgreSQL
			if err := db.Delete(&lobby).Error; err != nil {
				client.Emit("error", gin.H{"error": "Error deleting lobby"})
				return
			}

			// Delete lobby from Redis
			if redisClient != nil {
				if err := redisClient.DeleteGameLobby(lobbyID); err != nil {
					client.Emit("error", gin.H{"error": "Error deleting lobby from Redis"})
					return
				}
			}
		}

		// Broadcast to other players in the lobby that this player left
		client.To(socket.Room(lobbyID)).Emit("player_left", gin.H{
			"username": username,
			"lobby_id": lobbyID,
		})
	}
}

func HandleKickFromLobby(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("[KICK] HandleKickFromLobby iniciado - Usuario: %s, Args: %v", username, args)

		// Validate arguments
		if len(args) < 2 {
			log.Printf("[KICK-ERROR] Faltan argumentos para usuario %s", username)
			client.Emit("error", gin.H{"error": "Falta el ID del lobby o el usuario a expulsar"})
			return
		}

		lobbyID := args[0].(string)
		usernameToKick := args[1].(string)
		log.Printf("[KICK] Procesando expulsión del usuario %s del lobby %s por %s",
			usernameToKick, lobbyID, username)

		// Find the lobby
		var lobby models.GameLobby
		if err := db.Where("id = ?", lobbyID).First(&lobby).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				client.Emit("error", gin.H{"error": "Lobby not found"})
			} else {
				client.Emit("error", gin.H{"error": "Database error"})
			}
			return
		}

		// Check if the requesting user is the host
		if username != lobby.CreatorUsername {
			client.Emit("error", gin.H{"error": "Only the host can kick players"})
			return
		}

		// Check if the user to kick exists in the lobby
		var userInLobby models.InGamePlayer
		result := db.Where(
			"lobby_id = ? AND username = ?",
			lobbyID, usernameToKick,
		).First(&userInLobby)

		if result.RowsAffected == 0 {
			client.Emit("error", gin.H{"error": "User is not in the lobby"})
			return
		}

		// Cannot kick yourself (the host)
		if usernameToKick == username {
			client.Emit("error", gin.H{"error": "Host cannot kick themselves"})
			return
		}

		// Get kicked user's socket connection
		kickedUserSocket, exists := sio.GetConnection(usernameToKick)
		if !exists {
			log.Printf("[KICK-WARNING] No active socket connection found for user %s", usernameToKick)
			// Continue with kick process even if user isn't connected
		}

		// Start transaction
		tx := db.Begin()
		if tx.Error != nil {
			client.Emit("error", gin.H{"error": "Database error starting transaction"})
			return
		}

		// Delete the player from lobby in PostgreSQL
		if err := tx.Delete(&userInLobby).Error; err != nil {
			tx.Rollback()
			client.Emit("error", gin.H{"error": "Error kicking user from lobby"})
			return
		}

		playerLobby, err := redisClient.GetPlayerCurrentLobby(usernameToKick)
		if err == nil {
			fmt.Println("Current player lobby: ", playerLobby)
		}

		// Remove player from Redis if exists
		if redisClient != nil {
			if err := redisClient.DeleteInGamePlayer(usernameToKick, lobbyID); err != nil {
				tx.Rollback()
				client.Emit("error", gin.H{"error": "Error removing user from Redis"})
				return
			}
		}

		playerLobby2, err := redisClient.GetPlayerCurrentLobby(usernameToKick)
		if err == nil {
			fmt.Println("Current player lobby: ", playerLobby2)
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			client.Emit("error", gin.H{"error": "Error committing transaction"})
			return
		}

		// Make kicked user leave the room if they're connected
		if exists {
			kickedUserSocket.Leave(socket.Room(lobbyID))
			// Send direct message to kicked user's socket
			kickedUserSocket.Emit("you_were_kicked", gin.H{
				"lobby_id": lobbyID,
				"by_user":  username,
			})
		}

		// Emit success event to the kicker
		client.Emit("kick_success", gin.H{
			"message":     "Player kicked successfully",
			"kicked_user": usernameToKick,
			"lobby_id":    lobbyID,
		})

		// Broadcast to all users in the lobby that a player was kicked
		client.To(socket.Room(lobbyID)).Emit("player_kicked", gin.H{
			"kicked_user": usernameToKick,
			"by_user":     username,
			"lobby_id":    lobbyID,
		})

		log.Printf("[KICK-SUCCESS] Usuario %s expulsado exitosamente del lobby %s por %s",
			usernameToKick, lobbyID, username)
		// NOTE: no cerramos forzosamente la conexión, así que no eliminamos el
		// objeto del map de conexiones
	}
}

// Function to broadcast a message to all clients in a specific lobby.
func BroadcastMessageToLobby(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		lobbyID := args[0].(string)

		// check if lobby exists. We could maby have a "global" lobby check,
		// so that a connection is associated with a valid lobby only checked once
		_, err := utils.CheckLobbyExists(db, lobbyID)
		if err != nil {
			fmt.Println("Lobby does not exist:", lobbyID)
			client.Emit("error", gin.H{"error": "Lobby does not exist"})
			return
		}

		// NOTE: now decoding username at top level (when connection is established, just once)

		message := args[1].(string) // sanitize string?

		// Check if user is in lobby
		isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
		if err != nil {
			fmt.Println("Database error:", err)
			client.Emit("error", gin.H{"error": "Database error"})
			return
		}

		if !isInLobby {
			fmt.Println("User is NOT in lobby:", username, "Lobby:", lobbyID)
			client.Emit("error", gin.H{"error": "You must join the lobby before sending messages"})
			return
		}

		fmt.Println("Broadcasting to lobby:", lobbyID, "Message:", message)

		// Get user icon from PostgreSQL
		icon := utils.UserIcon(db, username)

		// Send the message to all clients in the lobby
		sio.Sio_server.To(socket.Room(lobbyID)).Emit("new_lobby_message", gin.H{"lobby_id": lobbyID, "username": username, "user_icon": icon, "message": message})
	}
}

func HandleStartGame(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("[STARTING-GAME] HandleStartGame started - Usuario: %s, Args: %v", username, args)

		if len(args) < 1 {
			log.Printf("[START-ERROR] Arguments are missing %s", username)
			client.Emit("error", gin.H{"error": "Lobby ID is missing"})
			return
		}

		lobbyID := args[0].(string)

		// Check if lobby exists in the database
		var lobby *models.GameLobby
		lobby, err := utils.CheckLobbyExists(db, lobbyID)
		if err != nil {
			fmt.Println("Lobby does not exist:", lobbyID)
			client.Emit("error", gin.H{"error": "Lobby does not exist"})
			return
		}

		// Check if user is the host
		if username != lobby.CreatorUsername {
			client.Emit("error", gin.H{"error": "Only the host can start the game"})
			return
		}

		// Update lobby state to "in progress" in PostgreSQL
		if err := db.Model(&models.GameLobby{}).Where("id = ?", lobbyID).Update("game_has_begun", true).Error; err != nil {
			client.Emit("error", gin.H{"error": "Error updating lobby state"})
			return
		}

		// Update Redis state to "in progress"
		if redisClient != nil {
			if err := redisClient.CloseLobby(lobbyID); err != nil {
				client.Emit("error", gin.H{"error": "Error updating Redis state"})
				return
			}
		}

		// Broadcast to all users in the lobby that the game is starting
		sio.Sio_server.To(socket.Room(lobbyID)).Emit("game_starting", gin.H{
			"lobby_id": lobbyID,
			"message":  "The game is starting!",
		})

		log.Printf("[START-SUCCESS] The game started succesfully %s by %s", lobbyID, username)
	}
}
