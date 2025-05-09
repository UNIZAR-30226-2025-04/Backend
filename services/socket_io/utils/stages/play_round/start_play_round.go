package play_round

import (
	game_constants "Nogler/constants/game"
	redis_models "Nogler/models/redis"
	poker "Nogler/services/poker"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------
// Functions that are executed to start the next game round
// ---------------------------------------------------------------

func PrepareRoundStart(redisClient *redis.RedisClient, lobbyID string) (*redis_models.GameLobby, int, error) {
	log.Printf("[ROUND-PREPARE] Preparing round start state for lobby %s", lobbyID)

	// Get the game lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-PREPARE-ERROR] Error getting lobby info: %v", err)
		return nil, 0, err
	}

	// Reset players finished round map in redis
	lobby.PlayersFinishedRound = make(map[string]bool)

	// Reset the blind timeout to indicate round has started
	lobby.BlindTimeout = time.Time{}

	// Set the current phase to play round
	lobby.CurrentPhase = redis_models.PhasePlayRound

	// Get the blind value
	blind := lobby.CurrentHighBlind

	// NEW: mark the current blind phase as completed
	lobby.BlindsCompleted[lobby.CurrentRound] = true

	// CRITICAL: Save the updated lobby state BEFORE broadcasting
	err = redisClient.SaveGameLobby(lobby)
	if err != nil {
		log.Printf("[ROUND-PREPARE-ERROR] Error updating lobby state: %v", err)
		return nil, 0, err
	}

	log.Printf("[ROUND-PREPARE-SUCCESS] Lobby %s prepared for round start with blind %d",
		lobbyID, blind)

	return lobby, blind, nil
}

func ResetPlayerAndBroadcastRoundStart(sio *socketio_types.SocketServer, redisClient *redis.RedisClient, lobbyID string, round int, blind int, timeout int) {
	log.Printf("[ROUND-BROADCAST] Broadcasting round start event for lobby %s", lobbyID)

	// Get the game lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-BROADCAST-ERROR] Error getting lobby info: %v", err)
		return
	}

	players, err := redisClient.GetAllPlayersInLobby(lobbyID)
	if err != nil {
		log.Printf("[SHOP-MULTICAST-ERROR] Error getting players: %v", err)
		return
	}

	// Send personalized message to each player
	for i, player := range players {
		// Reset player state for the new round

		// Reset current points (TotalPoints are kept for stats, although not used)
		player.CurrentRoundPoints = 0

		// Reset hand plays and discards limits
		player.HandPlaysLeft = game_constants.TOTAL_HAND_PLAYS
		player.DiscardsLeft = game_constants.TOTAL_DISCARDS

		// Reset current hand to empty array
		emptyHand := []poker.Card{}
		emptyHandJSON, err := json.Marshal(emptyHand)
		if err != nil {
			log.Printf("[ROUND-RESET-ERROR] Error creating empty hand for player %s: %v",
				player.Username, err)
			continue
		}
		// TODO, should be an empty hand
		player.CurrentHand = emptyHandJSON

		// Create new deck with standard cards + purchased pack cards
		playersCurrentDeck := poker.NewStandardDeck()

		// Add purchased cards to the deck if any exist
		if player.PurchasedPackCards != nil && len(player.PurchasedPackCards) > 0 {
			var purchasedCards []poker.Card
			if err := json.Unmarshal(player.PurchasedPackCards, &purchasedCards); err != nil {
				log.Printf("[ROUND-RESET-ERROR] Error parsing purchased cards for player %s: %v",
					player.Username, err)
			} else {
				// Add purchased cards to the deck
				playersCurrentDeck.TotalCards = append(playersCurrentDeck.TotalCards, purchasedCards...)
			}
		}

		// Shuffle the deck
		playersCurrentDeck.Shuffle()

		// Update player's deck
		player.CurrentDeck = playersCurrentDeck.ToJSON()

		// Save the updated player state to Redis
		if err := redisClient.SaveInGamePlayer(&player); err != nil {
			log.Printf("[ROUND-RESET-ERROR] Error saving updated player %s: %v",
				player.Username, err)
			continue
		}

		// Update local reference for broadcast
		players[i] = player

		// Get player's socket using GetConnection
		playerSocket, exists := sio.GetConnection(player.Username)
		if !exists {
			log.Printf("[SHOP-MULTICAST-WARNING] Player %s has no active connection", player.Username)
			continue
		}

		deckSize := len(playersCurrentDeck.TotalCards)

		// Calculate the appropriate blind value based on player's bet choice
		playerBlind := blind // Default to high blind

		// Send personalized message to this player
		playerSocket.Emit("starting_round", gin.H{
			"round_number":       round,
			"max_rounds":         game_constants.MaxGameRounds,
			"players_money":      player.PlayersMoney,
			"blind":              playerBlind,
			"timeout":            timeout,
			"timeout_start_date": lobby.GameRoundTimeout.Format(time.RFC3339),
			"total_hand_plays":   game_constants.TOTAL_HAND_PLAYS,
			"total_discards":     game_constants.TOTAL_DISCARDS,
			"current_pot":        CalculatePotAmount(lobby.CurrentRound),
			"current_jokers":     player.CurrentJokers,
			"active_vouchers":    player.ActivatedModifiers,
			"current_deck_size":  deckSize,
		})

		log.Printf("[ROUND-RESET] Reset player %s state for new round with %d cards in deck",
			player.Username, deckSize)
	}

	log.Printf("[ROUND-BROADCAST] Sent round start event to lobby %s with round %d and blind %d",
		lobbyID, round, blind)
}

// Apply modifiers to all players
func ApplyRoundModifiers(redisClient *redis.RedisClient, lobbyID string, sio *socketio_types.SocketServer) {
	log.Printf("[MODIFIER-APPLY] Applying round modifiers for lobby %s", lobbyID)

	// Get all players in the lobby
	players, err := redisClient.GetAllPlayersInLobby(lobbyID)
	if err != nil {
		log.Printf("[MODIFIER-APPLY-ERROR] Error getting players: %v", err)
		return
	}

	// Apply modifiers to each player
	for _, player := range players {
		// Activated modifiers
		var activatedModifiers poker.Modifiers
		if player.ActivatedModifiers != nil {
			err = json.Unmarshal(player.ActivatedModifiers, &activatedModifiers)
			if err != nil {
				log.Printf("[HAND-ERROR] Error parsing activated modifiers: %v", err)
				return
			}
		}

		// Received modifiers
		var receivedModifiers poker.Modifiers
		if player.ReceivedModifiers != nil {
			err = json.Unmarshal(player.ReceivedModifiers, &receivedModifiers)
			if err != nil {
				log.Printf("[HAND-ERROR] Error parsing activated modifiers: %v", err)
				return
			}
		}

		// ONLY FOR APPLYING RAM, SINCE IT NEEDS THE JOKERS. SORRY FOR UGLY CODE.
		for _, modifierID := range receivedModifiers.Modificadores {
			if modifierID.Value == 3 && len(player.CurrentJokers) > 0 {
				randomIndex := rand.Intn(len(player.CurrentJokers)) // Dont wanna set the seed, we use pseudorandomness
				removedJoker := player.CurrentJokers[randomIndex]
				player.CurrentJokers = append(player.CurrentJokers[:randomIndex], player.CurrentJokers[randomIndex+1:]...) // deletes joker from slice
				log.Printf("Removed joker (fake random): %v", removedJoker)
			}
		}

		currentGold := player.PlayersMoney

		// Apply activated modifiers to the player
		goldActivated := poker.ApplyRoundModifiers(&activatedModifiers, currentGold)

		// Apply received modifiers to the player (Currently there are no received modifiers that affect at the start of the round)
		goldReceived := poker.ApplyRoundModifiers(&receivedModifiers, goldActivated)

		if goldActivated != currentGold {
			// Notify player of gold change
			sio.UserConnections[player.Username].Emit("round_modifier", gin.H{
				"current_gold": goldReceived,
				"extra_gold":   goldReceived - currentGold,
			})
		}

		// Delete modifiers if there are no more plays left of the activated modifiers
		var remainingModifiers []poker.Modifier

		var deletedModifiers []poker.Modifier

		for _, modifier := range activatedModifiers.Modificadores {
			if modifier.Value == 1 || modifier.Value == 3 {
				modifier.LeftUses--
				if modifier.LeftUses != 0 {
					remainingModifiers = append(remainingModifiers, modifier)
				} else if modifier.LeftUses == 0 {
					deletedModifiers = append(deletedModifiers, modifier)
				}
			}
		}

		activatedModifiers.Modificadores = remainingModifiers
		player.ActivatedModifiers, err = json.Marshal(activatedModifiers)
		if err != nil {
			log.Printf("[HAND-ERROR] Error serializing activated modifiers: %v", err)
			return
		}

		// Delete modifiers if there are no more plays left of the received modifiers
		var remainingReceivedModifiers []poker.Modifier

		var deletedReceivedModifiers []poker.Modifier

		for _, modifier := range receivedModifiers.Modificadores {
			if modifier.Value == 1 || modifier.Value == 3 {
				modifier.LeftUses--
				if modifier.LeftUses != 0 {
					remainingReceivedModifiers = append(remainingReceivedModifiers, modifier)
				} else if modifier.LeftUses == 0 {
					deletedReceivedModifiers = append(deletedReceivedModifiers, modifier)
				}
			}
		}

		receivedModifiers.Modificadores = remainingReceivedModifiers
		player.ReceivedModifiers, err = json.Marshal(receivedModifiers)
		if err != nil {
			log.Printf("[HAND-ERROR] Error serializing received modifiers: %v", err)
			return
		}

		// Emit the deleted modifiers to the client
		if len(deletedModifiers) > 0 || len(deletedReceivedModifiers) > 0 {
			sio.UserConnections[player.Username].Emit("deleted_modifiers", gin.H{
				"deleted_activated_modifiers": deletedModifiers,
				"deleted_received_modifiers":  deletedReceivedModifiers,
			})
			log.Printf("[HAND-INFO] Deleted modifiers for user %s: activated: %v, received: %v",
				player.Username, deletedModifiers, deletedReceivedModifiers)
		}

		// Update redis
		player.PlayersMoney = goldReceived
		err = redisClient.SaveInGamePlayer(&player)
		if err != nil {
			log.Printf("[HAND-ERROR] Error saving player data: %v", err)
			return
		}
		log.Printf("[HAND-INFO] Player %s updated with activated modifiers: %v", player.Username, activatedModifiers)

	}

	log.Printf("[MODIFIER-APPLY] Successfully applied modifiers for lobby %s", lobbyID)
}
