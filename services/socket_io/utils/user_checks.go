package socketio_utils

import (
	"Nogler/middleware"
	models "Nogler/models/postgres"
	redis_models "Nogler/models/redis"
	"Nogler/services/redis"
	"Nogler/utils"
	"fmt"
	"log"

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
	token, exists := authData["authorization"].(string)
	if !exists {
		fmt.Println("No authorization token provided in handshake!")
		client.Emit("error", gin.H{"error": "Authentication failed: missing authorization token"})
		return false, "", ""
	}

	// Decode JWT to get the user's email
	fmt.Println("Provided JWT: ", token)
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

// Helper function to validate lobby and user, returning the lobby if valid
func ValidateLobbyAndUser(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, lobbyID string) (*redis_models.GameLobby, error) {

	log.Printf("[TIMEOUT-REQUEST] Validating lobby %s and user %s", lobbyID, username)

	// Check if the user is in the lobby
	isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
	if err != nil {
		log.Printf("[TIMEOUT-ERROR] Database error: %v", err)
		client.Emit("error", gin.H{"error": "Database error"})
		return nil, err
	}

	if !isInLobby {
		log.Printf("[TIMEOUT-ERROR] User is NOT in lobby: %s, Lobby: %s", username, lobbyID)
		client.Emit("error", gin.H{"error": "You must join the lobby before requesting timeout info"})
		return nil, fmt.Errorf("user not in lobby")
	}

	// Get lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[TIMEOUT-ERROR] Error obtaining lobby: %v", err)
		client.Emit("error", gin.H{"error": "Error obtaining lobby information"})
		return nil, err
	}

	return lobby, nil
}
