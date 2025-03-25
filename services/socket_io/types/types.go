package socketio_types

import (
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/socket.io/v2/socket"
)

type SocketServer struct {
	Sio_server          *socket.Server
	Webtransport_server *types.HttpServer
}
