package handlers

import (
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	socketio_utils "Nogler/services/socket_io/utils"
	"Nogler/services/socket_io/utils/game_flow"
	"Nogler/utils"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

func HandleProposeBlind(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("[BLIND] %s is proposing a blind", username)

		if len(args) < 2 {
			log.Printf("[BLIND-ERROR] Missing arguments for user %s", username)
			client.Emit("error", gin.H{"error": "Missing proposed blind or lobby"})
			return
		}

		proposedBlind := args[0].(int)
		lobbyID := args[1].(string)

		isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
		if err != nil {
			log.Printf("[BLIND-ERROR] Database error: %v", err)
			client.Emit("error", gin.H{"error": "Database error"})
			return
		}

		if !isInLobby {
			log.Printf("[BLIND-ERROR] User is NOT in lobby: %s, Lobby: %s", username, lobbyID)
			client.Emit("error", gin.H{"error": "You must join the lobby before proposing blinds"})
			return
		}

		// Get the lobby from Redis
		lobby, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			log.Printf("[BLIND-ERROR] Error getting game lobby: %v", err)
			client.Emit("error", gin.H{"error": "Error getting game lobby"})
			return
		}

		// Validate blind phase
		valid, err := socketio_utils.ValidateBlindPhase(redisClient, client, lobbyID)
		if err != nil || !valid {
			// Error already emitted in ValidateBlindPhase
			return
		}

		currentBlind, err := redisClient.GetCurrentBlind(lobbyID)
		if err != nil {
			log.Printf("[BLIND-ERROR] Error getting current blind: %v", err)
			client.Emit("error", gin.H{"error": "Error getting current blind"})
			return
		}

		// Update current blind if this proposal is higher
		if proposedBlind > currentBlind {
			err := redisClient.SetCurrentBlind(lobbyID, proposedBlind, username)
			if err != nil {
				log.Printf("[BLIND-ERROR] Could not update current blind: %v", err)
				client.Emit("error", gin.H{"error": "Error updating blind"})
				return
			}

			// Broadcast the new blind value to everyone in the lobby
			sio.Sio_server.To(socket.Room(lobbyID)).Emit("blind_updated", gin.H{
				"new_blind":   proposedBlind,
				"proposed_by": username,
			})
		}

		// Increment the counter of proposed blinds (NEW, using a map to avoid same user incrementing the counter several times)
		lobby.ProposedBlinds[username] = true
		log.Printf("[BLIND] Player %s proposed blind. Total proposals: %d/%d",
			username, len(lobby.ProposedBlinds), lobby.PlayerCount)

		// Save the updated lobby
		err = redisClient.SaveGameLobby(lobby)
		if err != nil {
			log.Printf("[BLIND-ERROR] Error saving game lobby: %v", err)
			client.Emit("error", gin.H{"error": "Error saving game state"})
			return
		}

		// If all players have proposed, start the round
		if len(lobby.ProposedBlinds) >= lobby.PlayerCount {
			log.Printf("[BLIND-COMPLETE] All players have proposed blinds (%d/%d). Starting round.",
				len(lobby.ProposedBlinds), lobby.PlayerCount)

			// Start the round immediately instead of waiting for timeout
			go game_flow.AdvanceToNextRoundPlayIfUndone(redisClient, db, lobbyID, sio, lobby.CurrentRound)
		}
	}
}

// Function we should call when
/*func Send_chosen_blind(lobbyID string, rc *redis.RedisClient, sio *socketio_types.SocketServer) {
	lobby, err := rc.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("Error obtaining lobby to broadcast blind: %v", err)
		return
	}
	blind := lobby.CurrentBlind
	sio.Sio_server.To(socket.Room(lobbyID)).Emit("send_chosen_blind", blind)
}*/

/*func HandleStarttimeout(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {

		lobbyID := args[0].(string)
		timeout := args[1].(int)

		log.Printf("[TIMEOUT] Starting timeout of %d minutes for lobby %s", timeout, lobbyID)

		// Check if lobby exists
		var lobbyPG *models.GameLobby
		lobbyPG, err := utils.CheckLobbyExists(db, lobbyID)
		if err != nil {
			fmt.Println("Lobby does not exist:", lobbyID)
			client.Emit("error", gin.H{"error": "Lobby does not exist"})
			return
		}

		// Check if the user is in the lobby
		isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
		if err != nil {
			log.Printf("[HAND-ERROR] Database error: %v", err)
			client.Emit("error", gin.H{"error": "Database error"})
			return
		}

		if !isInLobby {
			log.Printf("[HAND-ERROR] User is NOT in lobby: %s, Lobby: %s", username, lobbyID)
			client.Emit("error", gin.H{"error": "You must join the lobby before sending messages"})
			return
		}

		// Check if user is the host
		if username != lobbyPG.CreatorUsername {
			client.Emit("error", gin.H{"error": "Only the host can start the game"})
			return
		}

		// Start the timeout for the lobby
		lobby, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			log.Printf("[TIMEOUT-ERROR] Error obtaining lobby to start timeout: %v", err)
			return
		}

		// Check if the game is already in timeout
		if !lobby.Timeout.IsZero() {
			log.Printf("[TIMEOUT-ERROR] Game is already in timeout: %v", lobby.Timeout)
			return
		}

		// Set the timeout to the current time
		lobby.Timeout = time.Now()
		err = redisClient.SaveGameLobby(lobby)
		if err != nil {
			log.Printf("[TIMEOUT-ERROR] Error setting lobby timeout: %v", err)
			return
		}

		// Sleep for timeout duration
		time.Sleep(time.Minute * time.Duration(timeout))

		// Broadcast the timeout to all players in the lobby
		sio.Sio_server.To(socket.Room(lobbyID)).Emit("blind_timeout_expired", gin.H{"message": "Timeout reached"})
		log.Printf("[TIMEOUT] 2 minutes timeout reached %s", lobbyID)

		// Reset the timeout
		lobby.Timeout = time.Time{}
		err = redisClient.SaveGameLobby(lobby)
		if err != nil {
			log.Printf("[TIMEOUT-ERROR] Error resetting lobby timeout: %v", err)
			return
		}
		log.Printf("[TIMEOUT] Timeout reset for lobby %s", lobbyID)
	}
}*/

func HandleContinueToNextBlind(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		if len(args) < 1 {
			log.Printf("[NEXT-BLIND-ERROR] Missing lobby ID for user %s", username)
			client.Emit("error", gin.H{"error": "Missing lobby ID"})
			return
		}

		lobbyID := args[0].(string)
		log.Printf("[NEXT-BLIND] User %s requesting to continue to next blind in lobby %s", username, lobbyID)

		// Validate the user and lobby
		lobby, err := socketio_utils.ValidateLobbyAndUser(redisClient, client, db, username, lobbyID)
		if err != nil {
			return
		}

		// Validate shop phase
		valid, err := socketio_utils.ValidateShopPhase(redisClient, client, lobbyID)
		if err != nil || !valid {
			// Error already emitted in ValidateShopPhase
			return
		}

		// Increment the finished shop counter (NEW, using maps now)
		lobby.PlayersFinishedShop[username] = true
		log.Printf("[NEXT-BLIND] Player %s ready for next blind. Total ready: %d/%d",
			username, len(lobby.PlayersFinishedShop), lobby.PlayerCount)

		// Save the updated lobby
		err = redisClient.SaveGameLobby(lobby)
		if err != nil {
			log.Printf("[NEXT-BLIND-ERROR] Error saving game lobby: %v", err)
			client.Emit("error", gin.H{"error": "Error saving game state"})
			return
		}

		// If all players are ready, broadcast the starting_next_blind event
		if len(lobby.PlayersFinishedShop) >= lobby.PlayerCount {
			log.Printf("[NEXT-BLIND-COMPLETE] All players ready for next blind (%d/%d), round %d.",
				len(lobby.PlayersFinishedShop), lobby.PlayerCount, lobby.CurrentRound)

			// Advance to the next blind
			game_flow.AdvanceToNextBlindIfUndone(redisClient, db, lobbyID, sio, false, lobby.CurrentRound)
		}
	}
}
