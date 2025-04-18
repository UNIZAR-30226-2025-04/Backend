package socketio_types

import (
	"sync"

	"github.com/zishang520/socket.io/v2/socket"
)

// SocketServer is a struct that contains the socket.io server and a map of socket connections.
// It is used to handle socket.io connections.
type SocketServer struct {
	Sio_server *socket.Server
	// Map to track username -> socket connections
	UserConnections map[string]*socket.Socket
	Mutex           sync.RWMutex
}

func NewSocketServer() *SocketServer {
	return &SocketServer{
		UserConnections: make(map[string]*socket.Socket),
	}
}

// Add methods to manage connections
func (s *SocketServer) AddConnection(username string, socket *socket.Socket) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.UserConnections[username] = socket
}

func (s *SocketServer) RemoveConnection(username string) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	delete(s.UserConnections, username)
}

func (s *SocketServer) GetConnection(username string) (*socket.Socket, bool) {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()
	socket, exists := s.UserConnections[username]
	return socket, exists
}
