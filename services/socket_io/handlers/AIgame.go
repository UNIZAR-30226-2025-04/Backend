package handlers

import (
	game_constants "Nogler/constants/game"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	socketio_utils "Nogler/services/socket_io/utils"
	"log"
	"math/rand"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
)

func ProposeBlindAI(redisClient *redis.RedisClient, client *socket.Socket, lobbyID string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		username := "Noglerinho" // AI username

		log.Printf("[AI-BLIND] %s is proposing a blind", username)

		// Get the lobby from Redis
		lobby, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			log.Printf("[AI-BLIND-ERROR] Error getting game lobby: %v", err)
			client.Emit("error", gin.H{"error": "Error getting game lobby"})
			return
		}

		// Validate blind phase
		valid, err := socketio_utils.ValidateBlindPhase(redisClient, client, lobbyID)
		if err != nil || !valid {
			// Error already emitted in ValidateBlindPhase
			return
		}

		AI, err := redisClient.GetInGamePlayer(username)
		if err != nil {
			log.Printf("[AI-BLIND-ERROR] Error getting player data: %v", err)
			client.Emit("error", gin.H{"error": "Error getting player data"})
			return
		}

		// Generate a random blind
		AIMoney := AI.PlayersMoney
		proposedBlind := AIMoney/2 + rand.Intn(AIMoney-AIMoney/2+1)

		// Check if proposed blind exceeds MAX_BLIND
		if proposedBlind > game_constants.MAX_BLIND {
			log.Printf("[AI-BLIND] Player %s proposed blind %d exceeding MAX_BLIND, capping at %d",
				username, proposedBlind, int(game_constants.MAX_BLIND))
			proposedBlind = game_constants.MAX_BLIND
			AI.BetMinimumBlind = false
		} else if proposedBlind < lobby.CurrentBaseBlind {
			// If below base blind, set BetMinimumBlind to true
			log.Printf("[AI-BLIND] Player %s proposed blind %d below base blind %d, marking as min blind better",
				username, proposedBlind, lobby.CurrentBaseBlind)
			AI.BetMinimumBlind = true
		} else {
			// Otherwise, they're not betting the minimum
			AI.BetMinimumBlind = false
		}

		// Save player data
		if err := redisClient.SaveInGamePlayer(AI); err != nil {
			log.Printf("[AI-BLIND-ERROR] Error saving player data: %v", err)
			client.Emit("error", gin.H{"error": "Error saving player data"})
			return
		}

		currentBlind, err := redisClient.GetCurrentBlind(lobbyID)
		if err != nil {
			log.Printf("[AI-BLIND-ERROR] Error getting current blind: %v", err)
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
			log.Printf("[AI-BLIND-ERROR] Error saving game lobby: %v", err)
			client.Emit("error", gin.H{"error": "Error saving game state"})
			return
		}

		// Update current blind if this proposal is higher
		if proposedBlind > currentBlind {
			err := redisClient.SetCurrentHighBlind(lobbyID, proposedBlind, username)
			if err != nil {
				log.Printf("[AI-BLIND-ERROR] Could not update current blind: %v", err)
				client.Emit("error", gin.H{"error": "Error updating blind"})
				return
			}

			// Broadcast the new blind value to everyone in the lobby
			client.Emit("AI_blind_updated", gin.H{
				"old_max_blind": currentBlind,
				"new_blind":     proposedBlind,
				"proposed_by":   username,
			})
		}
	}
}
