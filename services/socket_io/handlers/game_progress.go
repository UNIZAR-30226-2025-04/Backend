package handlers

import (
	models "Nogler/models/postgres"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	"Nogler/utils"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

func HandleProposeBlind(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("[] %s ha empezado a proponer una ciega\n", username)

		if len(args) < 2 {
			log.Printf("[INFO-ERROR] Faltan argumentos para usuario %s", username)
			client.Emit("error", gin.H{"error": "Falta la ciega propuesta o el lobby"})
			return
		}

		proposedBlind := args[0].(int)
		lobby := args[1].(string)

		isInLobby, err := utils.IsPlayerInLobby(db, lobby, username)
		if err != nil {
			fmt.Println("Database error:", err)
			client.Emit("error", gin.H{"error": "Database error"})
			return
		}

		if !isInLobby {
			fmt.Println("User is NOT in lobby:", username, "Lobby:", lobby)
			client.Emit("error", gin.H{"error": "You must join the lobby before proposing blinds"})
			return
		}

		// TODO ?????? BroadcastMessageToLobby(redisClient, client, db, username, sio)

		currentBlind, err := redisClient.GetCurrentBlind(lobby)
		if err != nil {
			log.Printf("[DECK-ERROR] Error getting current blind: %v", err)
			client.Emit("error", gin.H{"error": "Error al obtener la ciega"})
			return
		}

		//TODO Manage when all players have given a blind or timeout is due round starts
		if proposedBlind > currentBlind {
			err := redisClient.SetCurrentBlind(lobby, proposedBlind)
			if err != nil {
				log.Printf("[INFO-ERROR] Could not update current blind")
				return
			}
		}
	}
}

// Function we should call when
func Send_chosen_blind(lobbyID string, rc *redis.RedisClient, sio *socketio_types.SocketServer) {
	lobby, err := rc.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("Error obtaining lobby to broadcast blind: %v", err)
		return
	}
	blind := lobby.CurrentBlind
	sio.Sio_server.To(socket.Room(lobbyID)).Emit("send_chosen_blind", blind)
}

func HandleStarttimeout(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {

		lobbyID := args[0].(string)
		timeout := args[1].(int)

		log.Printf("[TIMEOUT] Starting timeout of %d minutes for lobby %s", timeout, lobbyID)

		// Check if lobby exists
		var lobbyPG *models.GameLobby
		lobbyPG, err := utils.CheckLobbyExists(db, lobbyID)
		if err != nil {
			fmt.Println("Lobby does not exist:", lobbyID)
			client.Emit("error", gin.H{"error": "Lobby does not exist"})
			return
		}

		// Check if the user is in the lobby
		isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
		if err != nil {
			log.Printf("[HAND-ERROR] Database error: %v", err)
			client.Emit("error", gin.H{"error": "Database error"})
			return
		}

		if !isInLobby {
			log.Printf("[HAND-ERROR] User is NOT in lobby: %s, Lobby: %s", username, lobbyID)
			client.Emit("error", gin.H{"error": "You must join the lobby before sending messages"})
			return
		}

		// Check if user is the host
		if username != lobbyPG.CreatorUsername {
			client.Emit("error", gin.H{"error": "Only the host can start the game"})
			return
		}

		// Start the timeout for the lobby
		lobby, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			log.Printf("[TIMEOUT-ERROR] Error obtaining lobby to start timeout: %v", err)
			return
		}

		// Check if the game is already in timeout
		if !lobby.Timeout.IsZero() {
			log.Printf("[TIMEOUT-ERROR] Game is already in timeout: %v", lobby.Timeout)
			return
		}

		// Set the timeout to the current time
		lobby.Timeout = time.Now()
		err = redisClient.SaveGameLobby(lobby)
		if err != nil {
			log.Printf("[TIMEOUT-ERROR] Error setting lobby timeout: %v", err)
			return
		}

		// Sleep for timeout duration
		time.Sleep(time.Minute * time.Duration(timeout))

		// Broadcast the timeout to all players in the lobby
		sio.Sio_server.To(socket.Room(lobbyID)).Emit("started_timeout", gin.H{"message": "Timeout reached"})
		log.Printf("[TIMEOUT] 2 minutes timeout reached %s", lobbyID)

		// Reset the timeout
		lobby.Timeout = time.Time{}
		err = redisClient.SaveGameLobby(lobby)
		if err != nil {
			log.Printf("[TIMEOUT-ERROR] Error resetting lobby timeout: %v", err)
			return
		}
		log.Printf("[TIMEOUT] Timeout reset for lobby %s", lobbyID)
	}
}

func HandleRequestTimeout(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {

		lobbyID := args[0].(string)

		log.Printf("[TIMEOUT] Requesting timeout for lobby %s", lobbyID)

		// Check if the user is in the lobby
		isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
		if err != nil {
			log.Printf("[HAND-ERROR] Database error: %v", err)
			client.Emit("error", gin.H{"error": "Database error"})
			return
		}

		if !isInLobby {
			log.Printf("[HAND-ERROR] User is NOT in lobby: %s, Lobby: %s", username, lobbyID)
			client.Emit("error", gin.H{"error": "You must join the lobby before sending messages"})
			return
		}

		// Get timeout from lobby
		lobby, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			log.Printf("[TIMEOUT-ERROR] Error obtaining lobby to start timeout: %v", err)
			return
		}

		client.Emit("requested_timeout", gin.H{"message": lobby.Timeout})
	}
}
