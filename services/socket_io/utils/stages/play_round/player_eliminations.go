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
func handlePlayerEliminations(redisClient *redis.RedisClient, lobbyID string, sio *socketio_types.SocketServer) ([]string, error) {
	// List to track eliminated players
	var eliminatedPlayers []string

	// Get the lobby
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		return nil, fmt.Errorf("error getting lobby: %v", err)
	}

	highestBlindProposer := lobby.HighestBlindProposer
	blind := lobby.CurrentBlind

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
		proposerReachedBlind := proposerPlayer.CurrentPoints >= blind

		if !proposerReachedBlind {
			// Only eliminate the highest blind proposer
			eliminatedPlayers = append(eliminatedPlayers, highestBlindProposer)
			log.Printf("[ELIMINATION] Player %s eliminated for not reaching their proposed blind of %d (scored %d)",
				highestBlindProposer, blind, proposerPlayer.CurrentPoints)
		} else {
			// Eliminate all players who didn't reach the blind
			for _, player := range players {
				if player.CurrentPoints < blind {
					eliminatedPlayers = append(eliminatedPlayers, player.Username)
					log.Printf("[ELIMINATION] Player %s eliminated for not reaching the blind of %d (scored %d)",
						player.Username, blind, player.CurrentPoints)
				}
			}
		}

		// Remove eliminated players from Redis
		for _, username := range eliminatedPlayers {
			if err := redisClient.DeleteInGamePlayer(username, lobbyID); err != nil {
				log.Printf("[ELIMINATION-ERROR] Error removing player %s: %v", username, err)
			}
		}

		// Update player count
		if len(eliminatedPlayers) > 0 {
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
				"blind_value":        blind,
			})
		}
	}

	return eliminatedPlayers, nil
}
