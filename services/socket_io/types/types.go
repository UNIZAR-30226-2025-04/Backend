package socketio_types

import (
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/socket.io/v2/socket"
)

// SocketServer is a struct that contains the socket.io server and the webtransport server.
// It is used to handle socket.io connections and webtransport connections.
type SocketServer struct {
	Sio_server          *socket.Server
	Webtransport_server *types.HttpServer
}
