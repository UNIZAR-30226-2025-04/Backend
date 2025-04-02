package socketio_utils

import (
	"Nogler/middleware"
	models "Nogler/models/postgres"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

// Function that verifies a socket.io client connection using JWT authentication.
// It extracts the email from the JWT token and retrieves the associated username from the database.
func VerifyUserConnection(client *socket.Socket, db *gorm.DB) (success bool, username, email string) {
	// Checks if we have auth data in the connection
	authData, ok := client.Handshake().Auth.(map[string]interface{})
	if !ok {
		fmt.Println("No auth data provided in handshake!")
		client.Emit("error", gin.H{"error": "Authentication failed: missing auth data"})
		return false, "", ""
	}

	// Check if authorization token exists
	_, exists := authData["authorization"].(string)
	if !exists {
		fmt.Println("No authorization token provided in handshake!")
		client.Emit("error", gin.H{"error": "Authentication failed: missing authorization token"})
		return false, "", ""
	}

	// Decode JWT to get email
	fmt.Println("Provided JWT: ", authData["authorization"].(string))
	email, err := middleware.Socketio_JWT_decoder(authData)
	if err != nil {
		fmt.Println("Error decoding JWT:", err)
		client.Emit("error", gin.H{
			"error": "Authentication failed: invalid JWT. Remember to set it on the 'Authorization' field and with the 'Bearer ' prefix.",
		})
		return false, "", ""
	}

	// Fetch username from database using the email
	var user models.User
	result := db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		fmt.Println("Error fetching user from database:", result.Error)
		client.Emit("error", gin.H{"error": "Authentication failed: could not find user"})
		return false, "", email
	}

	username = user.ProfileUsername
	return true, username, email
}
