package handlers

import (
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

// Function to broadcast a message to all clients in a specific lobby.
func BroadcastMessageToLobby(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, sio *socketio_types.SocketServer) func(args ...interface{}) {
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

		// same as above, it might be better to check this on a higher level to
		// avoid repeated check. It isn't really that bad to check twice tho.
		authData, ok := client.Handshake().Auth.(map[string]interface{})
		if !ok {
			fmt.Println("Handshake auth data is missing or invalid!")
			client.Emit("error", gin.H{"error": "Authentication failed: missing auth data"})
			return
		}

		username, exists := authData["username"].(string)
		if !exists {
			fmt.Println("No username provided in handshake!")
			client.Emit("error", gin.H{"error": "Authentication failed: missing username"})
			return
		}

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
