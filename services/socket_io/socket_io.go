package socket_io

import (
	"Nogler/constants/event_constants"
	"Nogler/services/handlers"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/engine.io/v2/engine"
	"github.com/zishang520/engine.io/v2/log"
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/engine.io/v2/webtransport"
	"github.com/zishang520/socket.io/v2/socket"
)

type SocketServer struct {
	sio_server          *socket.Server
	webtransport_server *types.HttpServer
}

func (sio *SocketServer) Start(router *gin.Engine, db *gorm.DB) {
	log.DEBUG = true
	c := socket.DefaultServerOptions()
	c.SetServeClient(true)
	// c.SetConnectionStateRecovery(&socket.ConnectionStateRecovery{})
	// c.SetAllowEIO3(true)
	c.SetPingInterval(300 * time.Millisecond)
	c.SetPingTimeout(200 * time.Millisecond)
	c.SetMaxHttpBufferSize(1000000)
	c.SetConnectTimeout(1000 * time.Millisecond)
	c.SetTransports(types.NewSet("polling", "websocket", "webtransport"))
	c.SetCors(&types.Cors{
		Origin:      "*",
		Credentials: true,
	})

	sio.sio_server = socket.NewServer(nil, nil)
	sio.sio_server.On("connection", func(clients ...interface{}) {
		client := clients[0].(*socket.Socket)

		// Checks if we have auth data in the connection. (need username)
		authData, ok := client.Handshake().Auth.(map[string]interface{})
		if !ok {
			fmt.Println("No username provided in handshake!")
			client.Emit("error", gin.H{"error": "Authentication failed: missing username"})
			return
		}

		// Check if the data in auth is indeed a username
		username, exists := authData["username"].(string)
		if !exists {
			fmt.Println("No username provided in handshake!")
			client.Emit("error", gin.H{"error": "Authentication failed: missing username"})
			return
		}

		// log oki
		fmt.Println("A individual just connected!: ", username)

		client.On(event_constants.Join_lobby, func(args ...interface{}) {
			handlers.JoinLobbyHandler(client, db, args...)
		})

		client.On(event_constants.Broadcast_to_lobby, func(args ...interface{}) {
			handlers.BroadcastToLobbyHandler(sio.sio_server, client, db, args...)
		})

		client.On(event_constants.Buy_card, func(args ...interface{}) {
			handlers.BuyCardHandler(client, db, args...)
		})

		client.On(event_constants.Buy_joker, func(args ...interface{}) {
			handlers.BuyJokerHandler(client, db, args...)
		})

		client.On(event_constants.Buy_modifier, func(args ...interface{}) {
			handlers.BuyModifierHandler(client, db, args...)
		})

		client.On(event_constants.Sell_card, func(args ...interface{}) {
			handlers.SellCardHandler(client, db, args...)
		})

		client.On(event_constants.Sell_joker, func(args ...interface{}) {
			handlers.SellJokerHandler(client, db, args...)
		})

		client.On(event_constants.Play_hand, func(args ...interface{}) {
			handlers.PlayHandHandler(client, db, args...)
		})

	})

	sio.sio_server.Of("/custom", nil).On("connection", func(clients ...interface{}) {
		client := clients[0].(*socket.Socket)
		client.Emit("auth", client.Handshake().Auth)
	})

	router.POST("/socket.io/*f", gin.WrapH(sio.sio_server.ServeHandler(c)))
	router.GET("/socket.io/*f", gin.WrapH(sio.sio_server.ServeHandler(c)))

	// WebTransport start
	// WebTransport uses udp, so you need to enable the new service.
	sio.webtransport_server = types.NewWebServer(nil)
	// A certificate is required and cannot be a self-signed certificate.
	wts := sio.webtransport_server.ListenWebTransportTLS(":443", os.Getenv("FULLCHAIN_PATH"), os.Getenv("KEY_PATH"), nil, nil)

	// Here is the core logic of the WebTransport handshake.
	sio.webtransport_server.HandleFunc(sio.sio_server.Path()+"/", func(w http.ResponseWriter, r *http.Request) {
		if webtransport.IsWebTransportUpgrade(r) {
			// You need to call socketio.ServeHandler(nil) before this, otherwise you cannot get the Engine instance.
			sio.sio_server.Engine().(engine.Server).OnWebTransportSession(types.NewHttpContext(w, r), wts)
			fmt.Println("Upgrade to WebTransport")
		} else {
			fmt.Println("Upgrade to WebTransport")
			sio.webtransport_server.DefaultHandler.ServeHTTP(w, r)
		}
	})
	// WebTransport end

	SignalC := make(chan os.Signal)

	signal.Notify(SignalC, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for s := range SignalC {
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				sio.sio_server.Close(nil)
				os.Exit(0)
			}
		}
	}()

	fmt.Println("Socket server started")
}
