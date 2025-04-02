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
	// NOTE: higher ping interval and timeout to 1) reduce network load and 2) support slower networks
	c.SetPingInterval(5 * time.Second) // 300 * time.Millisecond
	c.SetPingTimeout(3 * time.Second)  // 200 * time.Millisecond
	c.SetMaxHttpBufferSize(1000000)
	c.SetConnectTimeout(10 * time.Second) // 1000 * time.Millisecond
	c.SetTransports(types.NewSet("polling", "websocket"))
	c.SetCors(&types.Cors{
		Origin:      "*",
		Credentials: true,
	})

	// KEY: inicializar el map, sino panikea
	sio.UserConnections = make(map[string]*socket.Socket)

	sio.Sio_server = socket.NewServer(nil, nil)
	sio.Sio_server.On("connection", func(clients ...interface{}) {
		client := clients[0].(*socket.Socket)

		// Check if the client is authenticated
		success, username, email := socketio_utils.VerifyUserConnection(client, db)
		if !success {
			return
		}

		// Add connection to map
		(*socketio_types.SocketServer)(sio).AddConnection(username, client)

		fmt.Println("Username: ", username)
		fmt.Println("Email: ", email)

		// log oki
		fmt.Println("An individual just connected!: ", username)
		fmt.Println("Current connections: ", sio.UserConnections)

		// Join the user to a room corresponding to a Nogler game lobby
		client.On("join_lobby", handlers.HandleJoinLobby(redisClient, client, db, username, (*socketio_types.SocketServer)(sio)))

		// Exit a lobby voluntarily
		client.On("exit_lobby", handlers.HandleExitLobby(redisClient, client, db, username))

		// Kick a user from a lobby (only for hosts)
		client.On("kick_from_lobby", handlers.HandleKickFromLobby(redisClient, client, db, username, (*socketio_types.SocketServer)(sio)))

		// Get (username,icon) of all users in a lobby and (username,icon) of the lobby host/creator
		client.On("get_lobby_info", handlers.GetLobbyInfo(redisClient, client, db, username))

		// Broadcast a message to all clients in a specific lobby
		client.On("broadcast_to_lobby", handlers.BroadcastMessageToLobby(redisClient, client, db, username, (*socketio_types.SocketServer)(sio)))

		// NOTE: will remove sio connection from map
		client.On("disconnecting", handlers.HandleDisconnecting(username, (*socketio_types.SocketServer)(sio)))

		// Start game
		client.On("start_game", handlers.HandleStartGame(redisClient, client, db, username, (*socketio_types.SocketServer)(sio)))

		// Play a hand and recieve the type of hand and the points scored
		client.On("play_hand", handlers.HandlePlayHand(redisClient, client, db, username))

		client.On("draw_cards", handlers.HandleDrawCards(redisClient, client, db, username))

		client.On("get_full_deck", handlers.HandleGetFullDeck(redisClient, client, db, username))

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
