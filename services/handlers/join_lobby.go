package handlers

import (
	"Nogler/utils"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

func JoinLobbyHandler(client *socket.Socket, db *gorm.DB, args ...interface{}) {

	lobbyID, ok := args[0].(string)
	if !ok {
		client.Emit("error", gin.H{"error": "Invalid lobby ID"})
		return
	}

	username, err := utils.GetUsernameFromClient(client)
	if err != nil {
		client.Emit("error", gin.H{"error": err.Error()})
		return
	}

	if utils.UserExists(db, username, client) != nil {
		return
	}

	if utils.Userisinlobby(db, lobbyID, username, client) != nil {
		fmt.Println("User is NOT in lobby:", username, "Lobby:", lobbyID)
		client.Emit("error", gin.H{"error": "You must join the lobby before sending messages"})
		return
	}

	client.Join(socket.Room(lobbyID))
	fmt.Println("Client joined lobby:", lobbyID)
	client.Emit("lobby_joined", gin.H{"lobby_id": lobbyID, "message": "Welcome to the lobby!"})
}
