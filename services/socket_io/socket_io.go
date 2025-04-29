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
			client.Emit("connection_error", gin.H{
				"message": "Authentication failed",
			})
			return
		}

		// Add connection to map
		// TODO: declare variable of type *socketio_types.SocketServer to avoid casting constantly
		sio_casted := (*socketio_types.SocketServer)(sio)
		sio_casted.AddConnection(username, client)

		fmt.Println("Username: ", username)
		fmt.Println("Email: ", email)

		// log oki
		fmt.Println("An individual just connected!: ", username)
		fmt.Println("Current connections: ", sio.UserConnections)

		client.Emit("connection_success", gin.H{
			"message":  "Connection successful",
			"username": username,
			"email":    email,
		})

		// Join the user to a room corresponding to a Nogler game lobby
		client.On("join_lobby", handlers.HandleJoinLobby(redisClient, client, db, username, sio_casted))

		// Exit a lobby voluntarily
		client.On("exit_lobby", handlers.HandleExitLobby(redisClient, client, db, username))

		// Kick a user from a lobby (only for hosts)
		client.On("kick_from_lobby", handlers.HandleKickFromLobby(redisClient, client, db, username, sio_casted))

		// Get (username,icon) of all users in a lobby and (username,icon) of the lobby host/creator
		client.On("get_lobby_info", handlers.GetLobbyInfo(redisClient, client, db, username))

		// Broadcast a message to all clients in a specific lobby
		client.On("broadcast_to_lobby", handlers.BroadcastMessageToLobby(redisClient, client, db, username, sio_casted))

		// NOTE: will remove sio connection from map
		client.On("disconnecting", handlers.HandleDisconnecting(username, sio_casted))

		// Start game
		client.On("start_game", handlers.HandleStartGame(redisClient, client, db, username, sio_casted))

		// Play a hand and recieve the type of hand and the points scored
		client.On("play_hand", handlers.HandlePlayHand(redisClient, client, db, username, sio_casted))

		client.On("get_cards", handlers.HandleGetCards(redisClient, client, db, username, sio_casted))

		client.On("discard_cards", handlers.HandleDiscardCards(redisClient, client, db, username, sio_casted))

		client.On("get_full_deck", handlers.HandleGetFullDeck(redisClient, client, db, username))

		client.On("propose_blind", handlers.HandleProposeBlind(redisClient, client, db, username, sio_casted))

		client.On("request_game_phase_player_info", handlers.HandleRequestGamePhaseInfo(redisClient, client, db, username))

		client.On("continue_to_next_blind", handlers.HandleContinueToNextBlind(redisClient, client, db, username, sio_casted))

		client.On("activate_modifiers", handlers.HandleActivateModifiers(redisClient, client, db, username, sio_casted))

		client.On("send_modifiers", handlers.HandleSendModifiers(redisClient, client, db, username, sio_casted))

		client.On("continue_to_vouchers", handlers.HandleContinueToVouchers(redisClient, client, db, username, sio_casted))

		client.On("get_phase_timeout", handlers.HandleGetPhaseTimeout(redisClient, client, db, username))

		// TODO, NOTE: should be already covered with activate_modifiers and send_modifiers
		//// client.On("play_voucher", handlers.HandlePlayVoucher(redisClient, client, db, username, sio_casted))

		client.On("buy_joker", handlers.HandleBuyJoker(redisClient, client, db, username, sio_casted))

		client.On("buy_voucher", handlers.HandleBuyVoucher(redisClient, client, db, username, sio_casted))

		client.On("buy_pack", handlers.HandlePurchasePack(redisClient, client, db, username))

		client.On("choose_pack_items", handlers.HandlePackSelection(redisClient, client, db, username, sio_casted))

		//client.On("get_from_pack", handlers.HandleGetFromPack(redisClient, client, db, username))

		client.On("rerroll_shop", handlers.HandleRerollShop(redisClient, client, db, username, sio_casted))

		// TODO: sell_joker
		client.On("sell_joker", handlers.HandleSellJoker(redisClient, client, db, username))
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
