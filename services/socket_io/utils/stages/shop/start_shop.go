package shop

import (
	"Nogler/models/redis"
	socketio_types "Nogler/services/socket_io/types"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
)

// ---------------------------------------------------------------
// Functions that are executed to start the shop phase
// ---------------------------------------------------------------

func BroadcastStartingShop(sio *socketio_types.SocketServer, lobbyID string, shopItems *redis.LobbyShop) {
	log.Printf("[SHOP-BROADCAST] Broadcasting shop start for lobby %s", lobbyID)

	// Broadcast shop start to all players
	sio.Sio_server.To(socket.Room(lobbyID)).Emit("starting_shop", gin.H{
		"shop": shopItems,
	})

	log.Printf("[SHOP-BROADCAST] Shop start broadcast sent to lobby %s", lobbyID)
}
