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
