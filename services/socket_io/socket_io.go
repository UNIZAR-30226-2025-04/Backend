package socket_io

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"gorm.io/gorm"
	"Nogler/utils"

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

		// TODO: check if user is a user?

		// log oki
		fmt.Println("A individual just connected!: ", username)

		// TODO: separate events in multiple files
		// TODO: ws token authentication?
		client.On("join_lobby", func(args ...interface{}) {
			lobbyID := args[0].(string) // needed string sanitize?

			// Check if lobby is indeed a real lobby
			lobby, err := utils.CheckLobbyExists(db, lobbyID)
			if err != nil {
				fmt.Println("Lobby does not exist:", lobbyID)
				client.Emit("error", gin.H{"error": "Lobby does not exist"})
				return
			}				

			// TODO: check if user is indeed in lobby (ON POSTGRES AND REDDIS). 
			// Comment: creator is always in lobby by design,
			// but right now we don't add other users to lobby. 

			// Was tested by adding manually to db, and we have a function
			// in utils to check if the user is on a lobby on postgres
			// therefore this check passes with Nico and yago

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

			
			client.Join(socket.Room(args[0].(string)))
			fmt.Println("Client joined lobby:", lobby)
        	client.Emit("lobby_joined", gin.H{"lobby_id": lobby, "message": "Welcome to the lobby!"})
		})

		// Broadcast a message to all clients in a specific lobby
    	client.On("broadcast_to_lobby", func(args ...interface{}) {
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
    		sio.sio_server.To(socket.Room(lobbyID)).Emit("new_lobby_message", gin.H{"lobby_id": lobbyID, "message": message})
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
