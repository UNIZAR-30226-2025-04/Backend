package socket_io

import (
	"Nogler/services/socket_io/handlers"
	socketio_types "Nogler/services/socket_io/types"
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

type MySocketServer socketio_types.SocketServer

func (sio *MySocketServer) Start(router *gin.Engine, db *gorm.DB) {
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

	sio.Sio_server = socket.NewServer(nil, nil)
	sio.Sio_server.On("connection", func(clients ...interface{}) {
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

		// Join the user to a room corresponding to a Nogler game lobby
		client.On("join_lobby", handlers.HandleJoinLobby(nil, client, db, username))

		// Broadcast a message to all clients in a specific lobby
		client.On("broadcast_to_lobby", handlers.BroadcastMessageToLobby(nil, client, db, (*socketio_types.SocketServer)(sio)))
	})

	sio.Sio_server.Of("/custom", nil).On("connection", func(clients ...interface{}) {
		client := clients[0].(*socket.Socket)
		client.Emit("auth", client.Handshake().Auth)
	})

	router.POST("/socket.io/*f", gin.WrapH(sio.Sio_server.ServeHandler(c)))
	router.GET("/socket.io/*f", gin.WrapH(sio.Sio_server.ServeHandler(c)))

	// WebTransport start
	// WebTransport uses udp, so you need to enable the new service.
	sio.Webtransport_server = types.NewWebServer(nil)
	// A certificate is required and cannot be a self-signed certificate.
	sio_port := os.Getenv("SOCKETIO_PORT")
	if sio_port == "" {
		sio_port = "443"
	}
	wts := sio.Webtransport_server.ListenWebTransportTLS(":"+sio_port, os.Getenv("FULLCHAIN_PATH"), os.Getenv("KEY_PATH"), nil, nil)

	// Here is the core logic of the WebTransport handshake.
	sio.Webtransport_server.HandleFunc(sio.Sio_server.Path()+"/", func(w http.ResponseWriter, r *http.Request) {
		if webtransport.IsWebTransportUpgrade(r) {
			// You need to call socketio.ServeHandler(nil) before this, otherwise you cannot get the Engine instance.
			sio.Sio_server.Engine().(engine.Server).OnWebTransportSession(types.NewHttpContext(w, r), wts)
			fmt.Println("Upgrade to WebTransport")
		} else {
			fmt.Println("Upgrade to WebTransport")
			sio.Webtransport_server.DefaultHandler.ServeHTTP(w, r)
		}
	})
	// WebTransport end

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
