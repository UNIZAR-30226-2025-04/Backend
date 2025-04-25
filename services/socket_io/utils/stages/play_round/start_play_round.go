package play_round

import (
	game_constants "Nogler/constants/game"
	redis_models "Nogler/models/redis"
	poker "Nogler/services/poker"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	"encoding/json"
	"log"
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

func BroadcastRoundStart(sio *socketio_types.SocketServer, redisClient *redis.RedisClient, lobbyID string, round int, blind int, timeout int) {
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
	for _, player := range players {
		// Get player's socket using GetConnection
		playerSocket, exists := sio.GetConnection(player.Username)
		if !exists {
			log.Printf("[SHOP-MULTICAST-WARNING] Player %s has no active connection", player.Username)
			continue
		}

		var deckSize int = 0
		if player.CurrentDeck != nil {
			deck, _ := poker.DeckFromJSON(player.CurrentDeck)
			deckSize = len(deck.TotalCards)
		}

		// Send personalized message to this player
		playerSocket.Emit("starting_round", gin.H{
			"round_number":       round,
			"blind":              blind,
			"timeout":            timeout,
			"timeout_start_date": lobby.GameRoundTimeout.Format(time.RFC3339),
			"total_hand_plays":   game_constants.TOTAL_HAND_PLAYS,
			"total_discards":     game_constants.TOTAL_DISCARDS,
			"current_pot":        lobby.CurrentRound + lobby.CurrentRound/2 + 1, // NOTE: formula specified in constants/game/constants.go
			"current_jokers":     player.CurrentJokers,
			"active_vouchers":    player.ActivatedModifiers,
			"current_deck_size":  deckSize,
		})

		log.Printf("[SHOP-MULTICAST] Sent personalized shop data to player %s", player.Username)
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
			if modifier.LeftUses != 0 {
				remainingModifiers = append(remainingModifiers, modifier)
			} else if modifier.LeftUses == 0 {
				deletedModifiers = append(deletedModifiers, modifier)
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

		var deletedReceiedModifiers []poker.Modifier

		for _, modifier := range activatedModifiers.Modificadores {
			if modifier.LeftUses != 0 {
				remainingReceivedModifiers = append(remainingReceivedModifiers, modifier)
			} else if modifier.LeftUses == 0 {
				deletedReceiedModifiers = append(deletedReceiedModifiers, modifier)
			}
		}

		receivedModifiers.Modificadores = remainingReceivedModifiers
		player.ReceivedModifiers, err = json.Marshal(receivedModifiers)
		if err != nil {
			log.Printf("[HAND-ERROR] Error serializing received modifiers: %v", err)
			return
		}

		// Emit the deleted modifiers to the client
		if len(deletedModifiers) > 0 {
			sio.UserConnections[player.Username].Emit("deleted_modifiers", gin.H{"deleted_activated_modifiers": deletedModifiers})
			log.Printf("[HAND-INFO] Deleted modifiers for user %s: %v", player.Username, deletedModifiers)
		}

		// Emit the deleted received modifiers to the client
		if len(deletedReceiedModifiers) > 0 {
			sio.UserConnections[player.Username].Emit("deleted_modifiers", gin.H{"deleted_received_modifiers": deletedReceiedModifiers})
			log.Printf("[HAND-INFO] Deleted received modifiers for user %s: %v", player.Username, deletedReceiedModifiers)
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
