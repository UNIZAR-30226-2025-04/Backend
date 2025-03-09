package socket_io

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

func (sio *SocketServer) Start(router *gin.Engine) {
	log.DEBUG = true
	c := socket.DefaultServerOptions()
	c.SetServeClient(true)
	// c.SetConnectionStateRecovery(&socket.ConnectionStateRecovery{})
	// c.SetAllowEIO3(true)
	c.SetPingInterval(300 * time.Millisecond)
	c.SetPingTimeout(200 * time.Millisecond)
	c.SetMaxHttpBufferSize(1000000)
	c.SetConnectTimeout(1000 * time.Millisecond)
	c.SetTransports(types.NewSet("polling", "webtransport"))
	c.SetCors(&types.Cors{
		Origin:      "*",
		Credentials: true,
	})

	sio.sio_server = socket.NewServer(nil, nil)
	sio.sio_server.On("connection", func(clients ...interface{}) {
		client := clients[0].(*socket.Socket)

		fmt.Println("A ****** just connected!: ", client)

		client.On("message", func(args ...interface{}) {
			client.Emit("message-back", args...)
		})
		client.Emit("auth", client.Handshake().Auth)

		client.On("message-with-ack", func(args ...interface{}) {
			ack := args[len(args)-1].(socket.Ack)
			ack(args[:len(args)-1], nil)
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
	wts := sio.webtransport_server.ListenWebTransportTLS(":443", "domain.cer", "domain.key", nil, nil)

	// Here is the core logic of the WebTransport handshake.
	sio.webtransport_server.HandleFunc(sio.sio_server.Path()+"/", func(w http.ResponseWriter, r *http.Request) {
		if webtransport.IsWebTransportUpgrade(r) {
			// You need to call socketio.ServeHandler(nil) before this, otherwise you cannot get the Engine instance.
			sio.sio_server.Engine().(engine.Server).OnWebTransportSession(types.NewHttpContext(w, r), wts)
		} else {
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
}
