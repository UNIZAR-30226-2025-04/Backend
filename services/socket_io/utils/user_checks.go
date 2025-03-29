package socketio_utils

import (
	"Nogler/middleware"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
)

// Function that verifies a socket.io client connection. That is, it checks whether
// the Handshake Auth object exists and contains a username. If either of these
// conditions is not met, the connection is rejected and an error message is sent
// to the client. If both conditions are met, the function returns true and the
// username of the client (connection accepted).
func VerifyUserConnection(client *socket.Socket) (success bool, username, email string) {
	// Checks if we have auth data in the connection. (need username)
	authData, ok := client.Handshake().Auth.(map[string]interface{})
	if !ok {
		fmt.Println("No username provided in handshake!")
		client.Emit("error", gin.H{"error": "Authentication failed: missing username"})
		return false, "", ""
	}

	// Check if the data in auth is indeed a username
	username, exists := authData["username"].(string)
	if !exists {
		fmt.Println("No username provided in handshake!")
		client.Emit("error", gin.H{"error": "Authentication failed: missing username"})
		return false, username, ""
	}

	// NUEVO: autenticación mediante JWT
	fmt.Println("Provided JWT: ", authData["authorization"].(string))
	email, err := middleware.Socketio_JWT_decoder(authData)
	if err != nil {
		fmt.Println("Error decoding JWT:", err)
		client.Emit("error", gin.H{
			"error": "Authentication failed: invalid JWT. Remember to set in on the 'Authorization' field and with the 'Bearer ' prefix. Provided JWT: " + authData["authorization"].(string),
		})
		return false, username, ""
	}

	return true, username, email
}
