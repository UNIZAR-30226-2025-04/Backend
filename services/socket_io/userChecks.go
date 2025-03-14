
package socket_io

import (
	"fmt"
	"gorm.io/gorm"
	"Nogler/utils"
	
	"github.com/zishang520/socket.io/v2/socket"

	"github.com/gin-gonic/gin"
)


func fieldOnAuthdata () {

}

// Check if user is in lobby
func userExists (db *gorm.DB, lobbyID string, username string, client *socket.Socket) (error) {
	_ , err := utils.IsPlayerInLobby(db, lobbyID, username)
	if err != nil {
		fmt.Println("Database error:", err)
		client.Emit("error", gin.H{"error": "Database error"})
	}
	return err
}


func userIsInLobby () {

}


func lobbyExists () {

}




