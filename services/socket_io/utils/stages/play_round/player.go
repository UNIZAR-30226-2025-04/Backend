package play_round

import (
	postgres_models "Nogler/models/postgres"
	redis_models "Nogler/models/redis"
	"Nogler/services/poker"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

// ValidatePlayerHand checks if the hand is valid for the player
// It validates that:
// 1. All cards in the hand are in the player's current hand
// 2. The gold in the hand matches the player's gold
// 3. All jokers in the hand are in the player's current jokers
func ValidatePlayerHand(player *redis_models.InGamePlayer, hand poker.Hand) (bool, string) {
	// Validate cards
	var currentCards []poker.Card
	err := json.Unmarshal(player.CurrentHand, &currentCards)
	if err != nil {
		return false, fmt.Sprintf("Error processing player's current hand: %v", err)
	}

	// Use the new helper function to validate cards
	if valid, errMsg := ValidatePlayerCards(currentCards, hand.Cards); !valid {
		return false, errMsg
	}

	// Validate gold if it was passed
	if hand.Gold != 0 && hand.Gold != player.PlayersMoney {
		return false, fmt.Sprintf("Gold mismatch: hand has %d, player has %d",
			hand.Gold, player.PlayersMoney)
	}

	// Validate jokers
	var playerJokers poker.Jokers
	err = json.Unmarshal(player.CurrentJokers, &playerJokers)
	if err != nil {
		return false, fmt.Sprintf("Error processing player's jokers: %v", err)
	}

	// Create a map of player's jokers for easy lookup
	jokerMap := make(map[int]bool)
	for _, joker := range playerJokers.Juglares {
		if joker != 0 {
			jokerMap[joker] = true
		}
	}

	// Check each joker in the hand
	for _, joker := range hand.Jokers.Juglares {
		if joker == 0 {
			continue
		}
		if !jokerMap[joker] {
			return false, fmt.Sprintf("Joker %d not available to player", joker)
		}
	}

	return true, ""
}

// ValidatePlayerCards checks if all the specified cards are in the player's hand
func ValidatePlayerCards(playerCards []poker.Card, cardsToValidate []poker.Card) (bool, string) {
	// Create a map of player's cards for efficient lookup
	// Use rank, suit, and enhancement as the composite key
	cardMap := make(map[string]int)
	for _, card := range playerCards {
		key := fmt.Sprintf("%s-%s-%d", card.Rank, card.Suit, card.Enhancement)
		cardMap[key]++
	}

	// Check if each card to validate is in the player's hand
	for _, card := range cardsToValidate {
		key := fmt.Sprintf("%s-%s-%d", card.Rank, card.Suit, card.Enhancement)
		if count, exists := cardMap[key]; !exists || count <= 0 {
			return false, fmt.Sprintf("Card %s-%s with enhancement %d not in player's hand",
				card.Rank, card.Suit, card.Enhancement)
		}
		cardMap[key]-- // Decrement to handle duplicate cards
	}

	return true, ""
}

// Separate function to handle player eliminations based on blind achievement
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

	// Get all players in the lobby
	players, err := redisClient.GetAllPlayersInLobby(lobbyID)
	if err != nil {
		return nil, fmt.Errorf("error getting players: %v", err)
	}

	// Variables for handling proposer logic
	var proposerPlayer *redis_models.InGamePlayer
	proposerReachedBlind := false

	// Only try to find the proposer if one exists
	if highestBlindProposer != "" {
		// Find the highest blind proposer player
		for i := range players {
			if players[i].Username == highestBlindProposer {
				proposerPlayer = &players[i]
				break
			}
		}

		if proposerPlayer != nil {
			// Check if the highest blind proposer reached their proposed blind
			proposerReachedBlind = proposerPlayer.CurrentRoundPoints >= currentHighBlind
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
				proposerPlayer.CurrentRoundPoints)
		} else {
			// Log the issue but don't return an error - proposer might have already been eliminated
			log.Printf("[ELIMINATION-WARN] Highest blind proposer %s not found in players list, continuing with elimination logic",
				highestBlindProposer)
		}
	} else {
		log.Printf("[ELIMINATION-INFO] No blind proposer for lobby %s, only applying base blind eliminations", lobbyID)
	}

	// First, handle all minimum-blind players (they ALWAYS get eliminated if they don't reach the base blind)
	for _, player := range players {
		if player.BetMinimumBlind && player.CurrentRoundPoints < baseBlind {
			// Minimum-betting players are eliminated if below base blind, regardless of proposer's success
			eliminatedPlayers = append(eliminatedPlayers, player.Username)
			log.Printf("[ELIMINATION] Player %s eliminated for betting minimum and not reaching base blind of %d (scored %d)",
				player.Username, baseBlind, player.CurrentRoundPoints)
		}
	}

	// Only process proposer-specific logic if we have a valid proposer
	if highestBlindProposer != "" && proposerPlayer != nil {
		if !proposerReachedBlind {
			// Proposer failed: Only eliminate proposer and min-blind-betting players who failed (already handled)
			eliminatedPlayers = append(eliminatedPlayers, highestBlindProposer)
			log.Printf("[ELIMINATION] Proposer %s eliminated for not reaching their proposed blind of %d (scored %d)",
				highestBlindProposer, currentHighBlind, proposerPlayer.CurrentRoundPoints)

			// Players who bet higher than minimum are safe in this scenario
			for _, player := range players {
				if !player.BetMinimumBlind && player.Username != highestBlindProposer {
					log.Printf("[ELIMINATION-SAFE] Player %s safe due to proposer failure (scored %d)",
						player.Username, player.CurrentRoundPoints)
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
				if player.CurrentRoundPoints < currentHighBlind {
					eliminatedPlayers = append(eliminatedPlayers, player.Username)
					log.Printf("[ELIMINATION] Player %s eliminated for not reaching high blind of %d (scored %d)",
						player.Username, currentHighBlind, player.CurrentRoundPoints)
				} else {
					log.Printf("[ELIMINATION-SAFE] Player %s safe by reaching high blind with %d points",
						player.Username, player.CurrentRoundPoints)
				}
			}
		}
	} else if highestBlindProposer == "" {
		// No high blind proposer: non-minimum players are safe (already handled minimum players)
		for _, player := range players {
			if !player.BetMinimumBlind {
				log.Printf("[ELIMINATION-SAFE] Player %s safe as no high blind was proposed (scored %d)",
					player.Username, player.CurrentRoundPoints)
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

// CalculatePotAmount calculates the pot amount based on the current round
func CalculatePotAmount(currentRound int) int {
	return currentRound + currentRound/2 + 1
}

// DistributePot distributes the current round pot to all non-eliminated players
// Each player receives the full pot amount (not divided)
func DistributePot(redisClient *redis.RedisClient, lobbyID string, sio *socketio_types.SocketServer, db *gorm.DB) error {
	// Get the lobby
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		return fmt.Errorf("error getting lobby: %v", err)
	}

	// Calculate pot amount
	potAmount := CalculatePotAmount(lobby.CurrentRound)

	// Get all surviving players
	players, err := redisClient.GetAllPlayersInLobby(lobbyID)
	if err != nil {
		return fmt.Errorf("error getting players: %v", err)
	}

	if len(players) == 0 {
		log.Printf("[POT-DISTRIBUTION] No players left to distribute pot to in lobby %s", lobbyID)
		return nil
	}

	log.Printf("[POT-DISTRIBUTION] Distributing pot of %d to %d players in lobby %s",
		potAmount, len(players), lobbyID)

	// Update each player's money in Redis and PostgreSQL
	for i := range players {
		// Add full pot amount to each player
		players[i].PlayersMoney += potAmount

		// Save in Redis
		if err := redisClient.SaveInGamePlayer(&players[i]); err != nil {
			log.Printf("[POT-DISTRIBUTION-ERROR] Error updating player %s money in Redis: %v",
				players[i].Username, err)
			continue
		}

		// Update in PostgreSQL
		if err := db.Model(&postgres_models.InGamePlayer{}).
			Where("lobby_id = ? AND username = ?", lobbyID, players[i].Username).
			Update("players_money", players[i].PlayersMoney).Error; err != nil {
			log.Printf("[POT-DISTRIBUTION-ERROR] Error updating player %s money in PostgreSQL: %v",
				players[i].Username, err)
		}
	}

	// Notify players about pot distribution
	sio.Sio_server.To(socket.Room(lobbyID)).Emit("pot_distributed", gin.H{
		"pot_amount":        potAmount,
		"players_remaining": len(players),
	})

	log.Printf("[POT-DISTRIBUTION] Successfully distributed pot for lobby %s", lobbyID)
	return nil
}
