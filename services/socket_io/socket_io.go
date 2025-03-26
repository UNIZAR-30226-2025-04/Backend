package socket_io

import (
	"Nogler/services/redis"
	"Nogler/services/socket_io/handlers"
	socketio_types "Nogler/services/socket_io/types"
	socketio_utils "Nogler/services/socket_io/utils"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/engine.io/v2/log"
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/socket.io/v2/socket"
)

type MySocketServer socketio_types.SocketServer

func (sio *MySocketServer) Start(router *gin.Engine, db *gorm.DB, redisClient *redis.RedisClient) {
	log.DEBUG = true
	c := socket.DefaultServerOptions()
	c.SetServeClient(true)
	// c.SetConnectionStateRecovery(&socket.ConnectionStateRecovery{})
	// c.SetAllowEIO3(true)
	c.SetPingInterval(300 * time.Millisecond)
	c.SetPingTimeout(200 * time.Millisecond)
	c.SetMaxHttpBufferSize(1000000)
	c.SetConnectTimeout(1000 * time.Millisecond)
	c.SetTransports(types.NewSet("polling", "websocket"))
	c.SetCors(&types.Cors{
		Origin:      "*",
		Credentials: true,
	})

	sio.Sio_server = socket.NewServer(nil, nil)
	sio.Sio_server.On("connection", func(clients ...interface{}) {
		client := clients[0].(*socket.Socket)

		// Check if the client is authenticated
		success, username, email := socketio_utils.VerifyUserConnection(client)
		if !success {
			return
		}

		// Add connection to map
		(*socketio_types.SocketServer)(sio).AddConnection(username, client)

		fmt.Println("Username: ", username)
		fmt.Println("Email: ", email)

		// log oki
		fmt.Println("A individual just connected!: ", username)

		// Join the user to a room corresponding to a Nogler game lobby
		client.On("join_lobby", handlers.HandleJoinLobby(redisClient, client, db, username))

		// Exit a lobby voluntarily
		client.On("exit_lobby", handlers.HandleExitLobby(redisClient, client, db, username))

		// Kick a user from a lobby (only for hosts)
		client.On("kick_from_lobby", handlers.HandleKickFromLobby(redisClient, client, db, username, (*socketio_types.SocketServer)(sio)))

		// Broadcast a message to all clients in a specific lobby
		client.On("broadcast_to_lobby", handlers.BroadcastMessageToLobby(redisClient, client, db, username, (*socketio_types.SocketServer)(sio)))

		// NOTE: will remove sio connection from map
		client.On("disconnecting", handlers.HandleDisconnecting(username, (*socketio_types.SocketServer)(sio)))
	})

	// NOTE: igual lo usamos en algún momento
	/*sio.Sio_server.Of("/custom", nil).On("connection", func(clients ...interface{}) {
		client := clients[0].(*socket.Socket)
		client.Emit("auth", client.Handshake().Auth)
	})*/

	router.POST("/socket.io/*f", gin.WrapH(sio.Sio_server.ServeHandler(c)))
	router.GET("/socket.io/*f", gin.WrapH(sio.Sio_server.ServeHandler(c)))

	SignalC := make(chan os.Signal, 1)

	signal.Notify(SignalC, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for s := range SignalC {
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				sio.Sio_server.Close(nil)
				os.Exit(0)
			}
		}
	}()

	fmt.Println("Socket server started")
}
