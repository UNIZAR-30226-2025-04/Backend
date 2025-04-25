package play_round

import (
	postgres_models "Nogler/models/postgres"
	redis_models "Nogler/models/redis"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

// Separate function to handle player eliminations based on blind achievement
// TODO: REVISE!!
func HandlePlayerEliminations(redisClient *redis.RedisClient, lobbyID string, sio *socketio_types.SocketServer, db *gorm.DB) ([]string, error) {
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

	if proposerPlayer == nil {
		log.Printf("[ELIMINATION-ERROR] Highest blind proposer %s not found in players list", highestBlindProposer)
		return nil, fmt.Errorf("highest blind proposer not found")
	}

	// Check if the highest blind proposer reached their proposed blind
	proposerReachedBlind := proposerPlayer.CurrentPoints >= currentHighBlind
	var reachedText string
	if proposerReachedBlind {
		reachedText = "reached"
	} else {
		reachedText = "failed to reach"
	}

	log.Printf("[ELIMINATION-CHECK] Highest proposer %s %s their proposed blind of %d with %d points",
		highestBlindProposer,
		reachedText,
		currentHighBlind,
		proposerPlayer.CurrentPoints)

	// First, handle all minimum-blind players (they ALWAYS get eliminated if they don't reach the base blind)
	for _, player := range players {
		if player.BetMinimumBlind && player.CurrentPoints < baseBlind {
			// Minimum-betting players are eliminated if below base blind, regardless of proposer's success
			eliminatedPlayers = append(eliminatedPlayers, player.Username)
			log.Printf("[ELIMINATION] Player %s eliminated for betting minimum and not reaching base blind of %d (scored %d)",
				player.Username, baseBlind, player.CurrentPoints)
		}
	}

	// Now handle the highest blind proposer and remaining players based on proposer's success
	if !proposerReachedBlind {
		// Proposer failed: Only eliminate proposer and min-blind-betting players who failed (already handled)
		eliminatedPlayers = append(eliminatedPlayers, highestBlindProposer)
		log.Printf("[ELIMINATION] Proposer %s eliminated for not reaching their proposed blind of %d (scored %d)",
			highestBlindProposer, currentHighBlind, proposerPlayer.CurrentPoints)

		// Players who bet higher than minimum are safe in this scenario
		for _, player := range players {
			if !player.BetMinimumBlind && player.Username != highestBlindProposer {
				log.Printf("[ELIMINATION-SAFE] Player %s safe due to proposer failure (scored %d)",
					player.Username, player.CurrentPoints)
			}
		}
	} else {
		// Proposer succeeded: Everyone must reach their respective targets
		for _, player := range players {
			// Skip players already processed (min-blind players and the proposer)
			if player.BetMinimumBlind || player.Username == highestBlindProposer {
				continue
			}

			// Non-minimum players must reach the high blind
			if player.CurrentPoints < currentHighBlind {
				eliminatedPlayers = append(eliminatedPlayers, player.Username)
				log.Printf("[ELIMINATION] Player %s eliminated for not reaching high blind of %d (scored %d)",
					player.Username, currentHighBlind, player.CurrentPoints)
			} else {
				log.Printf("[ELIMINATION-SAFE] Player %s safe by reaching high blind with %d points",
					player.Username, player.CurrentPoints)
			}
		}
	}

	// Remove eliminated players from Redis and update game state
	if len(eliminatedPlayers) > 0 {
		// Remove eliminated players from Redis and PostgreSQL
		for _, username := range eliminatedPlayers {
			// Delete from Redis
			if err := redisClient.DeleteInGamePlayer(username, lobbyID); err != nil {
				log.Printf("[ELIMINATION-ERROR] Error removing player %s from Redis: %v", username, err)
			}

			// Delete from PostgreSQL
			if err := db.Where("lobby_id = ? AND username = ?", lobbyID, username).Delete(&postgres_models.InGamePlayer{}).Error; err != nil {
				log.Printf("[ELIMINATION-ERROR] Error removing player %s from PostgreSQL: %v", username, err)
			} else {
				log.Printf("[ELIMINATION] Successfully removed player %s from PostgreSQL", username)
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
			"proposer_succeeded": proposerReachedBlind,
		})
	}

	// Just log if all players were eliminated - AnnounceWinners will handle this case
	if lobby.PlayerCount == 0 {
		log.Printf("[ELIMINATION-NOTICE] All players eliminated in lobby %s", lobbyID)
	}

	return eliminatedPlayers, nil
}
