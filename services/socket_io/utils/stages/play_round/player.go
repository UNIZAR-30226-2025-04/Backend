package play_round

import (
	redis_models "Nogler/models/redis"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
)

// Separate function to handle player eliminations based on blind achievement
// TODO: REVISE!!
func HandlePlayerEliminations(redisClient *redis.RedisClient, lobbyID string, sio *socketio_types.SocketServer) ([]string, error) {
	// List to track eliminated players
	var eliminatedPlayers []string

	// Get the lobby
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		return nil, fmt.Errorf("error getting lobby: %v", err)
	}

	highestBlindProposer := lobby.HighestBlindProposer
	currentHighBlind := lobby.CurrentHighBlind
	baseBlind := lobby.CurrentBaseBlind

	if highestBlindProposer == "" {
		log.Printf("[ELIMINATION-INFO] No blind proposer found for lobby %s, skipping eliminations", lobbyID)
		return nil, nil
	}

	// Get all players in the lobby
	players, err := redisClient.GetAllPlayersInLobby(lobbyID)
	if err != nil {
		return nil, fmt.Errorf("error getting players: %v", err)
	}

	// Find the highest blind proposer player
	var proposerPlayer *redis_models.InGamePlayer
	for i := range players {
		if players[i].Username == highestBlindProposer {
			proposerPlayer = &players[i]
			break
		}
	}

	// Apply elimination rules
	if proposerPlayer != nil {
		proposerReachedBlind := proposerPlayer.CurrentPoints >= currentHighBlind

		if !proposerReachedBlind {
			// Only eliminate the highest blind proposer who failed to reach their proposed blind
			eliminatedPlayers = append(eliminatedPlayers, highestBlindProposer)
			log.Printf("[ELIMINATION] Player %s eliminated for not reaching their proposed blind of %d (scored %d)",
				highestBlindProposer, currentHighBlind, proposerPlayer.CurrentPoints)
		} else {
			// Check each player based on their BetMinimumBlind status
			for _, player := range players {
				// Determine the target blind the player needs to reach
				playerTargetBlind := currentHighBlind
				if player.BetMinimumBlind {
					// If player bet minimum, they only need to reach the base blind
					playerTargetBlind = baseBlind
					log.Printf("[ELIMINATION-CHECK] Player %s bet minimum, needs to reach %d points",
						player.Username, baseBlind)
				} else {
					log.Printf("[ELIMINATION-CHECK] Player %s bet higher, needs to reach %d points",
						player.Username, currentHighBlind)
				}

				// Eliminate player if they didn't reach their target blind
				if player.CurrentPoints < playerTargetBlind {
					eliminatedPlayers = append(eliminatedPlayers, player.Username)
					log.Printf("[ELIMINATION] Player %s eliminated for not reaching their target blind of %d (scored %d)",
						player.Username, playerTargetBlind, player.CurrentPoints)
				} else {
					log.Printf("[ELIMINATION-SAFE] Player %s safe with %d points vs target of %d",
						player.Username, player.CurrentPoints, playerTargetBlind)
				}
			}
		}

		// Remove eliminated players from Redis and update game state
		if len(eliminatedPlayers) > 0 {
			// Remove eliminated players from Redis
			for _, username := range eliminatedPlayers {
				if err := redisClient.DeleteInGamePlayer(username, lobbyID); err != nil {
					log.Printf("[ELIMINATION-ERROR] Error removing player %s: %v", username, err)
				}
			}

			// Update player count
			lobby.PlayerCount -= len(eliminatedPlayers)
			if lobby.PlayerCount < 0 {
				lobby.PlayerCount = 0
			}

			if err := redisClient.SaveGameLobby(lobby); err != nil {
				log.Printf("[ELIMINATION-ERROR] Error updating player count: %v", err)
			}

			// Broadcast the eliminated players
			sio.Sio_server.To(socket.Room(lobbyID)).Emit("players_eliminated", gin.H{
				"eliminated_players": eliminatedPlayers,
				"reason":             "blind_check",
				"high_blind_value":   currentHighBlind,
				"base_blind":         baseBlind,
			})
		}
	}

	return eliminatedPlayers, nil
}
