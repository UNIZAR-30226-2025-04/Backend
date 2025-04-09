package handlers

import (
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
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

		// TODO ?????? BroadcastMessageToLobby(redisClient, client, db, username, sio)

		currentBlind, err := redisClient.GetCurrentBlind(lobby)
		if err != nil {
			log.Printf("[DECK-ERROR] Error getting current blind: %v", err)
			client.Emit("error", gin.H{"error": "Error al obtener la ciega"})
			return
		}

		if proposedBlind > currentBlind {
			err := redisClient.SetCurrentBlind(lobby, proposedBlind)
			if err != nil {
				log.Printf("[INFO-ERROR] Could not update current blind")
				return
			}
		}
	}
}
