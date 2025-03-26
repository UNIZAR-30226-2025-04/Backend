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

// Function to handle the act of joining a lobby. The code will check whether the user has
// requested to join the lobby via the API (querying the Postgres database) and if the lobby
// exists in Redis. If both checks are positive, the client will be automatically joined to the
// socket.io room corresponding to that lobby, and the Redis info about the player will be updated
// (a new `InGamePlayer` object will be inserted).
func HandleJoinLobby(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("[JOIN] HandleJoinLobby iniciado - Usuario: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())

		if len(args) < 1 {
			log.Printf("[JOIN-ERROR] Faltan argumentos para usuario %s", username)
			client.Emit("error", gin.H{"error": "Falta el ID del lobby"})
			return
		}

		// TODO: y si el mismo usuario vuelve a intentar conectarse otra vez sin haberse desconectado?

		lobbyID := args[0].(string)
		log.Printf("[JOIN] Procesando lobby ID: %s para usuario: %s", lobbyID, username)

		err := utils.UserExists(db, lobbyID, username, client)
		if err != nil {
			return
		}

		// 0. Check if user is in lobby (Postgres)
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

		// TODO: integrar el uso de la redis
		// A VER QUE NARICES HACEMOS CON ESTO NIÑO
		// 1. Verificar si existe el lobby en Redis
		/*lobby, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			log.Println("Error al buscar lobby:", err)
			client.Emit("error", gin.H{"error": "Lobby no encontrado en Redis"})
			return
		}
		log.Println("Lobby encontrado:", lobby)

		// 2. Verificar si el jugador ya está en el lobby
		currentLobby, err := redisClient.GetPlayerCurrentLobby(username)
		if err == nil && currentLobby == lobbyID {
			client.Emit("error", gin.H{"error": "Ya estás en este lobby"})
			return
		}
		log.Println("Jugador no está en el lobby:", username, currentLobby)
		// 3. Crear objeto InGamePlayer con valores iniciales
		player := &redis.InGamePlayer{
			Username:       username,
			LobbyId:        lobbyID,
			PlayersMoney:   1000,
			CurrentDeck:    json.RawMessage(`{"cards":[]}`),
			Modifiers:      json.RawMessage(`{}`),
			CurrentJokers:  json.RawMessage(`{}`),
			MostPlayedHand: json.RawMessage(`{}`),
		}
		log.Println("Jugador creado:", player)
		// 4. Guardar estado del jugador en Redis
		if err := redisClient.SaveInGamePlayer(player); err != nil {
			log.Println("Error al guardar jugador en Redis:", err)
			client.Emit("error", gin.H{"error": "Error al unirse al lobby"})
			return
		}
		log.Println("Jugador guardado en Redis")*/
		// 5. Unir al socket room
		client.Join(socket.Room(lobbyID))
		log.Println("Jugador unido al room:", lobbyID)
		// 6. Notificar éxito
		log.Printf("[JOIN-SUCCESS] Usuario %s unido exitosamente al lobby %s", username, lobbyID)
		client.Emit("joined_lobby", gin.H{
			"lobby_id": lobbyID,
			"message":  "¡Bienvenido al lobby!",
			// Pero vamos a ver, como que TotalPoints y NumberOfRounds si
			// ni ha empezado la partida tio???
			/*"total_points":     lobby.TotalPoints,
			"number_of_rounds": lobby.NumberOfRounds,*/
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

		// Notify success
		log.Printf("[EXIT-SUCCESS] Usuario %s ha salido exitosamente del lobby %s", username, lobbyID)
		client.Emit("exited_lobby", gin.H{
			"lobby_id": lobbyID,
			"message":  "Has salido del lobby exitosamente",
		})

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

		// Send the message to all clients in the lobby
		sio.Sio_server.To(socket.Room(lobbyID)).Emit("new_lobby_message", gin.H{"lobby_id": lobbyID, "message": message})
	}
}
