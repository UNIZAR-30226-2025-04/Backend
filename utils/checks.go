package utils

import (
	"errors"
	"fmt"

	models "Nogler/models/postgres"

	"gorm.io/gorm"

	"github.com/zishang520/socket.io/v2/socket"

	"github.com/gin-gonic/gin"
)

// Check if user is in lobby
func UserExists(db *gorm.DB, username string, client *socket.Socket) error {
	var user models.User
	err := db.Where("email = ?", username).First(&user).Error
	return err
}

func Userisinlobby(db *gorm.DB, lobbyid string, username string, client *socket.Socket) error {
	_, err := IsPlayerInLobby(db, lobbyid, username)
	if err != nil {
		fmt.Println("User is NOT in lobby:", username, "Lobby:", lobbyid)
		client.Emit("error", gin.H{"error": "You must join the lobby before sending messages"})
	}
	return err

}

func LobbyExists(db *gorm.DB, lobbyid string, client *socket.Socket) error {

	_, err := CheckLobbyExists(db, lobbyid)
	if err != nil {
		fmt.Println("Lobby does not exist:", lobbyid)
		client.Emit("error", gin.H{"error": "Lobby does not exist"})
	}
	return err

}

func GetUsernameFromClient(client *socket.Socket) (string, error) {
	authData, ok := client.Handshake().Auth.(map[string]interface{})
	if !ok {
		fmt.Println("No username provided in handshake!")
		client.Emit("error", gin.H{"error": "Authentication failed: missing username"})
		return "", errors.New("Authentication data missing")
	}

	username, exists := authData["username"].(string)
	if !exists {
		return "", errors.New("Username not found in authentication")
	}

	return username, nil
}
