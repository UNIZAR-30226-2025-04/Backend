package handlers

import (
	game_constants "Nogler/constants/game"
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

		// Assert as float64 first
		proposedBlindFloat, ok := args[0].(float64)
		if !ok {
			log.Printf("[BLIND-ERROR] Invalid type for proposed blind: expected number, got %T", args[0])
			client.Emit("error", gin.H{"error": "Invalid proposed blind value"})
			return
		}
		// Convert float64 to int
		proposedBlind := int(proposedBlindFloat)

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

		// Get player data to update BetMinimumBlind field
		player, err := redisClient.GetInGamePlayer(username)
		if err != nil {
			log.Printf("[BLIND-ERROR] Error getting player data: %v", err)
			client.Emit("error", gin.H{"error": "Error getting player data"})
			return
		}

		// Check if proposed blind exceeds MAX_BLIND
		if proposedBlind > game_constants.MAX_BLIND {
			log.Printf("[BLIND] Player %s proposed blind %d exceeding MAX_BLIND, capping at %d",
				username, proposedBlind, game_constants.MAX_BLIND)
			proposedBlind = game_constants.MAX_BLIND
			player.BetMinimumBlind = false
		} else if proposedBlind < lobby.CurrentBaseBlind {
			// If below base blind, set BetMinimumBlind to true
			log.Printf("[BLIND] Player %s proposed blind %d below base blind %d, marking as min blind better",
				username, proposedBlind, lobby.CurrentBaseBlind)
			player.BetMinimumBlind = true
		} else {
			// Otherwise, they're not betting the minimum
			player.BetMinimumBlind = false
		}

		// Save player data
		if err := redisClient.SaveInGamePlayer(player); err != nil {
			log.Printf("[BLIND-ERROR] Error saving player data: %v", err)
			client.Emit("error", gin.H{"error": "Error saving player data"})
			return
		}

		currentBlind, err := redisClient.GetCurrentBlind(lobbyID)
		if err != nil {
			log.Printf("[BLIND-ERROR] Error getting current blind: %v", err)
			client.Emit("error", gin.H{"error": "Error getting current blind"})
			return
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

		// If all players have proposed, start the round (no need to read the lobby again after calling redisClient.SetCurrentBlind)
		if len(lobby.ProposedBlinds) >= lobby.PlayerCount {
			log.Printf("[BLIND-COMPLETE] All players have proposed blinds (%d/%d). Starting round.",
				len(lobby.ProposedBlinds), lobby.PlayerCount)

			// Start the round immediately instead of waiting for timeout
			go game_flow.AdvanceToNextRoundPlayIfUndone(redisClient, db, lobbyID, sio, lobby.CurrentRound)
		}
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
