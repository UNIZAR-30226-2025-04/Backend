package handlers

import (
	"Nogler/utils"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

func BroadcastToLobbyHandler(sio *socket.Server, client *socket.Socket, db *gorm.DB, args ...interface{}) {

	lobbyID, ok := args[0].(string)
	if !ok {
		client.Emit("error", gin.H{"error": "Invalid lobby ID"})
		return
	}

	message, ok := args[1].(string)
	if !ok {
		client.Emit("error", gin.H{"error": "Invalid message format"})
		return
	}

	username, err := utils.GetUsernameFromClient(client)
	if err != nil {
		client.Emit("error", gin.H{"error": err.Error()})
		return
	}

	if utils.LobbyExists(db, lobbyID, client) != nil {
		fmt.Println("Lobby does not exist:", lobbyID)
		client.Emit("error", gin.H{"error": "Lobby does not exist"})
		return
	}

	// same as above, it might be better to check this on a higher level to
	// avoid repeated check. It isn't really that bad to check twice tho.
	authData, ok := client.Handshake().Auth.(map[string]interface{})
	if !ok {
		fmt.Println("Handshake auth data is missing or invalid!")
		client.Emit("error", gin.H{"error": "Authentication failed: missing auth data"})
		return
	}

	username, exists := authData["username"].(string)
	if !exists {
		fmt.Println("No username provided in handshake!")
		client.Emit("error", gin.H{"error": "Authentication failed: missing username"})
		return
	}

	if utils.Userisinlobby(db, lobbyID, username, client) != nil {
		fmt.Println("User is NOT in lobby:", username, "Lobby:", lobbyID)
		client.Emit("error", gin.H{"error": "You must join the lobby before sending messages"})
		return
	}

	fmt.Println("Broadcasting to lobby:", lobbyID, "Message:", message)

	// Send the message to all clients in the lobby
	sio.To(socket.Room(lobbyID)).Emit("new_lobby_message", gin.H{"lobby_id": lobbyID, "message": message})

}
