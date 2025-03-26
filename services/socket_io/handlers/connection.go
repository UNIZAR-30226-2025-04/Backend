package handlers

import (
	socketio_types "Nogler/services/socket_io/types"
	"fmt"
)

// Function to handle socket.io client disconnections.
func HandleDisconnecting(username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		// Remove connection from map
		sio.RemoveConnection(username)
		fmt.Println("A user just disconnected: ", username)
		fmt.Println("Current connections: ", sio.UserConnections)
	}
}
