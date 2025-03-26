package socketio_utils

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
)

// TODO: Esto igual se podría meter en la /utils de la raíz. Si no, mover lo que
// haya en /utils (raíz) de socketio a este fichero.
func fieldOnAuthdata() {

}

func userIsInLobby() {

}

func lobbyExists() {

}

// Function that verifies a socket.io client connection. That is, it checks whether
// the Handshake Auth object exists and contains a username. If either of these
// conditions is not met, the connection is rejected and an error message is sent
// to the client. If both conditions are met, the function returns true and the
// username of the client (connection accepted).
func VerifyUserConnection(client *socket.Socket) (success bool, username string) {
	// Checks if we have auth data in the connection. (need username)
	authData, ok := client.Handshake().Auth.(map[string]interface{})
	if !ok {
		fmt.Println("No username provided in handshake!")
		client.Emit("error", gin.H{"error": "Authentication failed: missing username"})
		return false, ""
	}

	// Check if the data in auth is indeed a username
	username, exists := authData["username"].(string)
	if !exists {
		fmt.Println("No username provided in handshake!")
		client.Emit("error", gin.H{"error": "Authentication failed: missing username"})
		return false, username
	}

	return true, username
}
