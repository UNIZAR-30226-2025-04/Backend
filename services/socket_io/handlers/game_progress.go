package handlers

import (
	redis_models "Nogler/models/redis"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	"Nogler/services/socket_io/utils/stages/game_flow"
	"Nogler/utils"
	"fmt"
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

		// Check if blind timeout has already started
		if lobby.BlindTimeout.IsZero() {
			log.Printf("[BLIND-WARNING] Trying to propose blind without active blind phase for lobby %s", lobbyID)
			client.Emit("error", gin.H{"error": "Blind voting phase is not active"})
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

		// Increment the counter of proposed blinds
		lobby.TotalProposedBlinds++
		log.Printf("[BLIND] Player %s proposed blind. Total proposals: %d/%d",
			username, lobby.TotalProposedBlinds, lobby.PlayerCount)

		// Save the updated lobby
		err = redisClient.SaveGameLobby(lobby)
		if err != nil {
			log.Printf("[BLIND-ERROR] Error saving game lobby: %v", err)
			client.Emit("error", gin.H{"error": "Error saving game state"})
			return
		}

		// If all players have proposed, start the round
		if lobby.TotalProposedBlinds >= lobby.PlayerCount {
			log.Printf("[BLIND-COMPLETE] All players have proposed blinds (%d/%d). Starting round.",
				lobby.TotalProposedBlinds, lobby.PlayerCount)

			// Start the round immediately instead of waiting for timeout
			go game_flow.AdvanceToNextRoundPlay(redisClient, db, lobbyID, sio)
		}
	}
}

// Function we should call when
func Send_chosen_blind(lobbyID string, rc *redis.RedisClient, sio *socketio_types.SocketServer) {
	lobby, err := rc.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("Error obtaining lobby to broadcast blind: %v", err)
		return
	}
	blind := lobby.CurrentBlind
	sio.Sio_server.To(socket.Room(lobbyID)).Emit("send_chosen_blind", blind)
}

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

// Helper function to validate lobby and user, returning the lobby if valid
func validateLobbyAndUser(redisClient *redis.RedisClient, client *socket.Socket,
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

func HandleRequestBlindTimeout(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {
		if len(args) < 1 {
			client.Emit("error", gin.H{"error": "Lobby ID is required"})
			return
		}

		lobbyID := args[0].(string)
		log.Printf("[BLIND-TIMEOUT-REQUEST] Requesting blind timeout for lobby %s by user %s", lobbyID, username)

		lobby, err := validateLobbyAndUser(redisClient, client, db, username, lobbyID)
		if err != nil {
			return
		}

		client.Emit("blind_timeout_info", gin.H{
			"timeout": lobby.BlindTimeout,
		})
	}
}

func HandleRequestGameRoundTimeout(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {
		if len(args) < 1 {
			client.Emit("error", gin.H{"error": "Lobby ID is required"})
			return
		}

		lobbyID := args[0].(string)
		log.Printf("[GAME-ROUND-TIMEOUT-REQUEST] Requesting game round timeout for lobby %s by user %s", lobbyID, username)

		lobby, err := validateLobbyAndUser(redisClient, client, db, username, lobbyID)
		if err != nil {
			return
		}

		client.Emit("game_round_timeout_info", gin.H{
			"timeout": lobby.GameRoundTimeout,
		})
	}
}

func HandleRequestShopTimeout(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {
		if len(args) < 1 {
			client.Emit("error", gin.H{"error": "Lobby ID is required"})
			return
		}

		lobbyID := args[0].(string)
		log.Printf("[SHOP-TIMEOUT-REQUEST] Requesting shop timeout for lobby %s by user %s", lobbyID, username)

		lobby, err := validateLobbyAndUser(redisClient, client, db, username, lobbyID)
		if err != nil {
			return
		}

		client.Emit("shop_timeout_info", gin.H{
			"timeout": lobby.ShopTimeout,
		})
	}
}

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
		lobby, err := validateLobbyAndUser(redisClient, client, db, username, lobbyID)
		if err != nil {
			return
		}

		// Increment the finished shop counter
		lobby.TotalPlayersFinishedShop++
		log.Printf("[NEXT-BLIND] Player %s ready for next blind. Total ready: %d/%d",
			username, lobby.TotalPlayersFinishedShop, lobby.PlayerCount)

		// Save the updated lobby
		err = redisClient.SaveGameLobby(lobby)
		if err != nil {
			log.Printf("[NEXT-BLIND-ERROR] Error saving game lobby: %v", err)
			client.Emit("error", gin.H{"error": "Error saving game state"})
			return
		}

		// If all players are ready, broadcast the starting_next_blind event
		if lobby.TotalPlayersFinishedShop >= lobby.PlayerCount {
			log.Printf("[NEXT-BLIND-COMPLETE] All players ready for next blind (%d/%d), round %d.",
				lobby.TotalPlayersFinishedShop, lobby.PlayerCount, lobby.CurrentRound)

			// Advance to the next blind
			game_flow.AdvanceToNextBlind(redisClient, db, lobbyID, sio, false)
		}
	}
}
