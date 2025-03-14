
package socket_io

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"gorm.io/gorm"
	"Nogler/utils"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/engine.io/v2/engine"
	"github.com/zishang520/engine.io/v2/log"
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/engine.io/v2/webtransport"
	"github.com/zishang520/socket.io/v2/socket"
)


func fieldOnAuthdata () {

}

// Check if user is in lobby
func userExists (db *gorm.DB, lobbyID string, username string, client *socket.Socket) (string, error) {
	isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
	if err != nil {
		fmt.Println("Database error:", err)
		client.Emit("error", gin.H{"error": "Database error"})
		return
	}
}


func userIsInLobby () {

}


func lobbyExists () {

}





		// Check if the data in auth is indeed a username
		username, exists := authData["username"].(string)
		if !exists {
		    fmt.Println("No username provided in handshake!")
		    client.Emit("error", gin.H{"error": "Authentication failed: missing username"})
		    return
		}

		// TODO: check if user is a user?

		// log oki
		fmt.Println("A individual just connected!: ", username)

		// TODO: separate events in multiple files
		// TODO: ws token authentication?
		client.On("join_lobby", func(args ...interface{}) {
			lobbyID := args[0].(string) // needed string sanitize?

			// Check if lobby is indeed a real lobby
			lobby, err := utils.CheckLobbyExists(db, lobbyID)
			if err != nil {
				fmt.Println("Lobby does not exist:", lobbyID)
				client.Emit("error", gin.H{"error": "Lobby does not exist"})
				return
			}				

			// TODO: check if user is indeed in lobby (ON POSTGRES AND REDDIS). 
			// Comment: creator is always in lobby by design,
			// but right now we don't add other users to lobby. 

			// Was tested by adding manually to db, and we have a function
			// in utils to check if the user is on a lobby on postgres
			// therefore this check passes with Nico and yago

			// Check if user is in lobby
			isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
    		if err != nil {
        		fmt.Println("Database error:", err)
        		client.Emit("error", gin.H{"error": "Database error"})
        		return
    		}

    		if !isInLobby {
    		    fmt.Println("User is NOT in lobby:", username, "Lobby:", lobbyID)
     		   	client.Emit("error", gin.H{"error": "You must join the lobby before sending messages"})
     		   	return
    		}

			
			client.Join(socket.Room(args[0].(string)))
			fmt.Println("Client joined lobby:", lobby)
        	client.Emit("lobby_joined", gin.H{"lobby_id": lobby, "message": "Welcome to the lobby!"})
		})

		// Broadcast a message to all clients in a specific lobby
    	client.On("broadcast_to_lobby", func(args ...interface{}) {
			lobbyID := args[0].(string)

			// check if lobby exists. We could maby have a "global" lobby check,
			// so that a connection is associated with a valid lobby only checked once
			_, err := utils.CheckLobbyExists(db, lobbyID)
			if err != nil {
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

			message := args[1].(string) // sanitize string?

			// Check if user is in lobby
			isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
    		if err != nil {
        		fmt.Println("Database error:", err)
        		client.Emit("error", gin.H{"error": "Database error"})
        		return
    		}

    		if !isInLobby {
    		    fmt.Println("User is NOT in lobby:", username, "Lobby:", lobbyID)
     		   	client.Emit("error", gin.H{"error": "You must join the lobby before sending messages"})
     		   	return
    		}

       	 	fmt.Println("Broadcasting to lobby:", lobbyID, "Message:", message)

	        // Send the message to all clients in the lobby
    		sio.sio_server.To(socket.Room(lobbyID)).Emit("new_lobby_message", gin.H{"lobby_id": lobbyID, "message": message})
  	  	})
	})
