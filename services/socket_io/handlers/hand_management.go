package handlers

import (
	"Nogler/services/poker"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	socketio_utils "Nogler/services/socket_io/utils"
	"Nogler/services/socket_io/utils/game_flow"
	"Nogler/services/socket_io/utils/stages/play_round"
	"Nogler/utils"
	"encoding/json"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

// Hacer aquí una tablica de relación nombre -> puntos o devolver desde el otro lado
// Un valor directamente. Lo mejor sería que consultemos en un spot (redis pg o dnd  sea)
// El nivel al que tenemo sla mano para saber fichas y mult base
// Ahora mismo está como string en el aproach mencionado sería 2 ints, fichas y mult
func HandlePlayHand(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {

		log.Printf("PlayHand iniciado - Usuario: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())

		if len(args) < 1 {
			log.Printf("[HAND-ERROR] Faltan argumentos para usuario %s", username)
			client.Emit("error", gin.H{"error": "Falta la mano a jugar"})
			return
		}

		// 1. Get player data from Redis to extract lobby ID
		player, err := redisClient.GetInGamePlayer(username)
		if err != nil {
			log.Printf("[HAND-ERROR] Error getting player data: %v", err)
			client.Emit("error", gin.H{"error": "Error al obtener los datos del jugador"})
			return
		}

		lobbyID := player.LobbyId
		if lobbyID == "" {
			log.Printf("[HAND-ERROR] User %s is not in a lobby", username)
			client.Emit("error", gin.H{"error": "You must join a lobby before playing hands"})
			return
		}

		// Check player is in lobby (double check with PostgreSQL)
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

		// Validate play round phase
		valid, err := socketio_utils.ValidatePlayRoundPhase(redisClient, client, lobbyID)
		if err != nil || !valid {
			// Error already emitted in ValidatePlayRoundPhase
			return
		}

		// 2. Check if the player has enough plays left
		if player.HandPlaysLeft <= 0 {
			log.Printf("[HAND-ERROR] No hand plays left %s", username)
			client.Emit("error", gin.H{"error": "No hand plays left"})
			return
		}

		handData := args[0].(map[string]interface{}) // Argument is expected to be a map (which is a generic object)
		handJson, err := json.Marshal(handData)      // Convert the argument to JSON
		if err != nil {
			log.Printf("[HAND-ERROR] Error al convertir la mano a JSON: %v", err)
			client.Emit("error", gin.H{"error": "Error al convertir la mano"})
			return
		}

		// Parse the JSON into the poker.Hand struct
		var hand poker.Hand
		err = json.Unmarshal(handJson, &hand)
		if err != nil {
			log.Printf("[HAND-ERROR] Error al parsear la mano: %v", err)
			client.Emit("error", gin.H{"error": "Error al procesar la mano"})
			return
		}

		if (hand.Cards == nil) || (len(hand.Cards) == 0) {
			log.Printf("[HAND-ERROR] No cards in hand for user %s", username)
			client.Emit("error", gin.H{"error": "No cards in hand"})
			return
		}

		// Validate that the hand is valid for this player
		valid, errMsg := play_round.ValidatePlayerHand(player, hand)
		if !valid {
			log.Printf("[HAND-ERROR] Invalid hand for user %s: %s", username, errMsg)
			client.Emit("error", gin.H{"error": errMsg})
			return
		}

		// If no Gold was passed, set it to the player's money
		if hand.Gold == 0 {
			hand.Gold = player.PlayersMoney
		}

		// 3. Calculate base points
		fichas, mult, handType, scored_cards := poker.BestHand(hand)

		fichas += poker.AddChipsPerCard(scored_cards)

		enhancedFichas, enhancedMult := poker.ApplyEnhancements(fichas, mult, scored_cards)

		// 4. Apply jokers (passing the hand which contains the jokers)
		finalFichas, finalMult, finalGold, jokersTriggered := poker.ApplyJokers(hand, hand.Jokers, enhancedFichas, enhancedMult, hand.Gold)

		// 5. Apply modifiers

		// Get the most played hand from the player
		var mostPlayedHand poker.Hand
		if player.MostPlayedHand != nil {
			err = json.Unmarshal(player.MostPlayedHand, &mostPlayedHand)
			if err != nil {
				log.Printf("[HAND-ERROR] Error parsing most played hand: %v", err)
				client.Emit("error", gin.H{"error": "Error parsing most played hand"})
				return
			}
		}

		// Apply activated modifiers
		var activatedModifiers poker.Modifiers
		if player.ActivatedModifiers != nil {
			err = json.Unmarshal(player.ActivatedModifiers, &activatedModifiers)
			if err != nil {
				log.Printf("[HAND-ERROR] Error parsing activated modifiers: %v", err)
				client.Emit("error", gin.H{"error": "Error parsing activated modifiers"})
				return
			}
		}

		finalFichas, finalMult, finalGold = poker.ApplyModifiers(hand, mostPlayedHand, &activatedModifiers, finalFichas, finalMult, finalGold)
		if err != nil {
			log.Printf("[HAND-ERROR] Error applying modifiers: %v", err)
			client.Emit("error", gin.H{"error": "Error applying modifiers"})
			return
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
			client.Emit("error", gin.H{"error": "Error serializing activated modifiers"})
			return
		}

		// Apply received modifiers
		var receivedModifiers poker.Modifiers
		if player.ActivatedModifiers != nil {
			err = json.Unmarshal(player.ReceivedModifiers, &receivedModifiers)
			if err != nil {
				log.Printf("[HAND-ERROR] Error parsing received modifiers: %v", err)
				client.Emit("error", gin.H{"error": "Error parsing received modifiers"})
				return
			}
		}

		finalFichas, finalMult, finalGold = poker.ApplyModifiers(hand, mostPlayedHand, &activatedModifiers, finalFichas, finalMult, finalGold)
		if err != nil {
			log.Printf("[HAND-ERROR] Error applying modifiers: %v", err)
			client.Emit("error", gin.H{"error": "Error applying modifiers"})
			return
		}

		valorFinal := finalFichas * finalMult

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
			client.Emit("error", gin.H{"error": "Error serializing received modifiers"})
			return
		}

		// 6. Update player data in Redis

		var currentHand []poker.Card
		err = json.Unmarshal(player.CurrentHand, &currentHand)
		if err != nil {
			log.Printf("[HAND-ERROR] Error unmarshaling current hand: %v", err)
			client.Emit("error", gin.H{"error": "Error processing current hand"})
			return
		}
		// Delete the played hand from the current hand
		for _, card := range hand.Cards {
			for i, c := range currentHand {
				if c.Suit == card.Suit && c.Rank == card.Rank {
					currentHand = append(currentHand[:i], currentHand[i+1:]...)
					break
				}
			}
		}

		var deck *poker.Deck
		if player.CurrentDeck != nil {
			deck, err = poker.DeckFromJSON(player.CurrentDeck)
			if err != nil {
				log.Printf("[DECK-ERROR] Error parsing deck: %v", err)
				client.Emit("error", gin.H{"error": "Error al procesar el mazo"})
				return
			}
		} else {
			deck = &poker.Deck{
				TotalCards:  make([]poker.Card, 0),
				PlayedCards: make([]poker.Card, 0),
			}
		}

		// Get new cards from the deck
		newCards := deck.Draw(len(hand.Cards))
		if newCards == nil {
			client.Emit("error", gin.H{"error": "There are not enough cards available in the deck"})
			return
		}

		// TODO Update around here hand in redis

		// Add the new cards to the hand
		currentHand = append(currentHand, newCards...)
		player.CurrentHand, err = json.Marshal(currentHand)
		if err != nil {
			log.Printf("[HAND-ERROR] Error serializing current hand: %v", err)
			client.Emit("error", gin.H{"error": "Error serializing current hand"})
			return
		}
		// Add the played hand to the played cards
		deck.PlayedCards = append(deck.PlayedCards, hand.Cards...)
		// Remove the played hand from the deck
		deck.RemoveCards(newCards)
		player.CurrentDeck = deck.ToJSON()

		// TODO, CRITICAL: reset TotalPoints before each game round?
		player.CurrentPoints = valorFinal
		player.TotalPoints += valorFinal
		player.HandPlaysLeft--
		err = redisClient.UpdateDeckPlayer(*player)
		if err != nil {
			log.Printf("[HAND-ERROR] Error updating player data: %v", err)
			client.Emit("error", gin.H{"error": "Error updating player data"})
			return
		}

		// Log the result
		log.Println("Jugador ha puntuado la friolera de:", valorFinal)
		// 7. Emit success response
		client.Emit("played_hand", gin.H{
			"total_score":         valorFinal,
			"gold":                finalGold,
			"hand_type":           handType,
			"jokersTriggered":     jokersTriggered,
			"left_plays":          player.HandPlaysLeft,
			"activated_modifiers": activatedModifiers,
			"received_modifiers":  receivedModifiers,
			"played_cards":        len(deck.PlayedCards),
			"unplayed_cards":      len(deck.TotalCards) + len(currentHand),
			"new_cards":           newCards,
			"scored_cards":        scored_cards,
			"card_points":         fichas,
			"red_score":           finalMult,
			"blue_score":          finalFichas,
			"message":             "¡Mano jugada con éxito!",
		})

		// 8. If the player has no plays left, emit a message
		if player.HandPlaysLeft <= 0 {
			client.Emit("no_plays_left", gin.H{"message": "No hand plays left"})
			log.Printf("[HAND-NO-PLAYS] User %s has no plays left", username)
		}

		// 9. Emit the deleted modifiers to the client
		if len(deletedModifiers) > 0 {
			client.Emit("deleted_modifiers", gin.H{"deleted_activated_modifiers": deletedModifiers})
			log.Printf("[HAND-INFO] Deleted modifiers for user %s: %v", username, deletedModifiers)
		}

		// 10. Emit the deleted received modifiers to the client
		if len(deletedReceiedModifiers) > 0 {
			client.Emit("deleted_modifiers", gin.H{"deleted_received_modifiers": deletedReceiedModifiers})
			log.Printf("[HAND-INFO] Deleted received modifiers for user %s: %v", username, deletedReceiedModifiers)
		}

		// NOTE: check it outside the `if` sentence, since the player might have reached the blind
		checkPlayerFinishedRound(redisClient, db, username, lobbyID, sio)

		//logear en redis + pg cuanto ha puntuado supongo IMPORTANTEEEEEEEEEEEEEEEEEEEEEE

		// Añadir aquí tajo checks, está en lobby redis + postgres + tod ala pesca. si se pueden hacer en asincrono mejor, así no esperamos a ello.
		// 0. Check if user is in lobby (Postgres)
		/*
			isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
			if err != nil {
				fmt.Println("Database error:", err)
				client.Emit("error", gin.H{"error": "Database error"})
				return
			}

			if !isInLobby {
				fmt.Println("User is NOT in lobby:", username, "Lobby:", lobbyID)
				client.Emit("error", gin.H{"error": "You must join the lobby before sending messages"})
				return
			}
		*/

	}
}

// checkPlayerFinishedRound checks if a player has finished the round and handles it
func checkPlayerFinishedRound(redisClient *redis.RedisClient, db *gorm.DB, username string,
	lobbyID string, sio *socketio_types.SocketServer) {

	log.Printf("[ROUND-CHECK] Checking if player %s has finished round in lobby %s", username, lobbyID)

	// Get player from Redis
	player, err := redisClient.GetInGamePlayer(username)
	if err != nil {
		log.Printf("[ROUND-CHECK-ERROR] Error getting player data: %v", err)
		return
	}

	// Get the lobby to check blind value
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-CHECK-ERROR] Error getting lobby: %v", err)
		return
	}

	// Check if player has no plays and discards left OR has reached/exceeded the blind
	if (player.HandPlaysLeft <= 0) || (player.CurrentPoints >= lobby.CurrentHighBlind) {
		if player.CurrentPoints >= lobby.CurrentHighBlind {
			log.Printf("[ROUND-CHECK] Player %s has reached the blind of %d with %d points",
				username, lobby.CurrentHighBlind, player.CurrentPoints)
		} else {
			log.Printf("[ROUND-CHECK] Player %s has finished their round (no plays or discards left)", username)
		}

		// Mark player as finished in the lobby
		if lobby.PlayersFinishedRound == nil {
			lobby.PlayersFinishedRound = make(map[string]bool)
		}

		lobby.PlayersFinishedRound[username] = true
		log.Printf("[ROUND-CHECK] Incremented finished players count to %d/%d for lobby %s",
			len(lobby.PlayersFinishedRound), lobby.PlayerCount, lobbyID)

		// Save the updated lobby
		err = redisClient.SaveGameLobby(lobby)
		if err != nil {
			log.Printf("[ROUND-CHECK-ERROR] Error saving lobby: %v", err)
			return
		}

		// If all players have finished the round, end it
		if len(lobby.PlayersFinishedRound) >= lobby.PlayerCount {
			log.Printf("[ROUND-CHECK] All players (%d/%d) have finished their round in lobby %s. Ending round.",
				len(lobby.PlayersFinishedRound), lobby.PlayerCount, lobbyID)

			game_flow.HandleRoundPlayEnd(redisClient, db, lobbyID, sio, lobby.CurrentRound)
		}
	}
}

func ApplyJokers(h poker.Hand, fichas int, mult int) int {
	// Given a hand and the points obtained from poker.Hand
	return fichas * mult
}

// Get left cards of a hand
func HandleGetCards(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("GetCards request - User: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())

		// 1. Get player data from Redis to extract lobby ID
		player, err := redisClient.GetInGamePlayer(username)
		if err != nil {
			log.Printf("[GET_CARDS-ERROR] Error getting player data: %v", err)
			client.Emit("error", gin.H{"error": "Error getting player data"})
			return
		}

		lobbyID := player.LobbyId
		if lobbyID == "" {
			log.Printf("[GET_CARDS-ERROR] User %s is not in a lobby", username)
			client.Emit("error", gin.H{"error": "You must join a lobby before getting cards"})
			return
		}

		// Check player is in lobby (double check with PostgreSQL)
		isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
		if err != nil {
			log.Printf("[GET_CARDS-ERROR] Database error: %v", err)
			client.Emit("error", gin.H{"error": "Database error"})
			return
		}

		if !isInLobby {
			log.Printf("[GET_CARDS-ERROR] User is NOT in lobby: %s, Lobby: %s", username, lobbyID)
			client.Emit("error", gin.H{"error": "You must join the lobby before sending messages"})
			return
		}

		// Validate play round phase
		valid, err := socketio_utils.ValidatePlayRoundPhase(redisClient, client, lobbyID)
		if err != nil || !valid {
			// Error already emitted in ValidatePlayRoundPhase
			return
		}

		var deck *poker.Deck
		if player.CurrentDeck != nil {
			deck, err = poker.DeckFromJSON(player.CurrentDeck)
			if err != nil {
				log.Printf("[GET_CARDS-ERROR] Error parsing deck: %v", err)
				client.Emit("error", gin.H{"error": "Error parsing deck"})
				return
			}
		} else {
			deck = &poker.Deck{
				TotalCards:  make([]poker.Card, 0),
				PlayedCards: make([]poker.Card, 0),
			}
		}

		var hand []poker.Card
		err = json.Unmarshal(player.CurrentHand, &hand)
		if err != nil {
			log.Printf("[GET_CARDS-ERROR] Error unmarshaling current hand: %v", err)
			client.Emit("error", gin.H{"error": "Error processing current hand"})
			return
		}

		// 3. Determine how many cards the player needs
		cardsNeeded := 8 - len(hand)
		if cardsNeeded <= 0 {
			client.Emit("error", gin.H{"error": "Player already has enough cards"})
			return
		}

		// 4. Get the necessary cards
		newCards := deck.Draw(cardsNeeded)
		if newCards == nil {
			client.Emit("error", gin.H{"error": "There are not enough cards available in the deck"})
			return
		}

		// Update hand
		hand = append(hand, newCards...)

		// 5. Update the player info in Redis
		deck.RemoveCards(newCards)
		player.CurrentDeck = deck.ToJSON()

		player.CurrentHand, err = json.Marshal(hand)
		if err != nil {
			log.Printf("[GET_CARDS-ERROR] Error serializing current hand: %v", err)
			client.Emit("error", gin.H{"error": "Error serializing current hand"})
			return
		}

		err = redisClient.UpdateDeckPlayer(*player)
		if err != nil {
			log.Printf("[GET_CARDS-ERROR] Error updating player data: %v", err)
			client.Emit("error", gin.H{"error": "Error updating player data"})
			return
		}

		// Get the lobby to retrieve current phase
		lobby, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			log.Printf("[GET_CARDS-ERROR] Error getting lobby: %v", err)
			client.Emit("error", gin.H{"error": "Error retrieving game phase"})
			return
		}

		// 6. Prepare the response with the full deck state
		response := gin.H{
			"new_cards":     newCards,
			"current_hand":  hand,
			"deck_size":     len(deck.TotalCards),
			"current_phase": lobby.CurrentPhase, // Add the current phase
		}

		// 7. Send the response to the client
		client.Emit("got_cards", response)
		log.Printf("[GET_CARDS-SUCCESS] Sent updated deck to user %s (%d total cards, phase: %s)",
			username, response["deck_size"], lobby.CurrentPhase)
	}
}

// Discard cards
func HandleDiscardCards(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("DiscardCards request - User: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())

		if len(args) < 1 {
			log.Printf("[DISCARD-ERROR] Missing arguments for user %s", username)
			client.Emit("error", gin.H{"error": "Missing hand data"})
			return
		}

		// 1. Get player data from Redis to extract lobby ID
		player, err := redisClient.GetInGamePlayer(username)
		if err != nil {
			log.Printf("[DISCARD-ERROR] Error getting player data: %v", err)
			client.Emit("error", gin.H{"error": "Error getting player data"})
			return
		}

		lobbyID := player.LobbyId
		if lobbyID == "" {
			log.Printf("[DISCARD-ERROR] User %s is not in a lobby", username)
			client.Emit("error", gin.H{"error": "You must join a lobby before discarding cards"})
			return
		}

		// Check player is in lobby (double check with PostgreSQL)
		isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
		if err != nil {
			log.Printf("[DISCARD-ERROR] Database error: %v", err)
			client.Emit("error", gin.H{"error": "Database error"})
			return
		}

		if !isInLobby {
			log.Printf("[DISCARD-ERROR] User is NOT in lobby: %s, Lobby: %s", username, lobbyID)
			client.Emit("error", gin.H{"error": "You must join the lobby before sending messages"})
			return
		}

		// Validate play round phase
		valid, err := socketio_utils.ValidatePlayRoundPhase(redisClient, client, lobbyID)
		if err != nil || !valid {
			// Error already emitted in ValidatePlayRoundPhase
			return
		}

		// 2. Check if the user has enough draws left
		if player.DiscardsLeft <= 0 {
			log.Printf("[DISCARD-ERROR] No draws left for user %s", username)
			client.Emit("error", gin.H{"error": "No draws left"})
			return
		}

		var deck *poker.Deck
		if player.CurrentDeck != nil {
			deck, err = poker.DeckFromJSON(player.CurrentDeck)
			if err != nil {
				log.Printf("[DISCARD-ERROR] Error parsing deck: %v", err)
				client.Emit("error", gin.H{"error": "Error al procesar el mazo"})
				return
			}
		} else {
			deck = &poker.Deck{
				TotalCards:  make([]poker.Card, 0),
				PlayedCards: make([]poker.Card, 0),
			}
		}

		discards := args[0].([]interface{})         // Argument is expected to be a map (which is a generic object)
		discardsJson, err := json.Marshal(discards) // Convert the argument to JSON
		if err != nil {
			log.Printf("[DISCARD-ERROR] Error al convertir la mano a JSON: %v", err)
			client.Emit("error", gin.H{"error": "Error al convertir la mano"})
			return
		}

		// Parse the JSON into the poker.Hand struct
		var discard []poker.Card
		err = json.Unmarshal(discardsJson, &discard)
		if err != nil {
			log.Printf("[DISCARD-ERROR] Error al parsear la mano: %v", err)
			client.Emit("error", gin.H{"error": "Error al procesar la mano"})
			return
		}

		// Get the current hand
		var hand []poker.Card
		err = json.Unmarshal(player.CurrentHand, &hand)
		if err != nil {
			log.Printf("[GET_CARDS-ERROR] Error unmarshaling current hand: %v", err)
			client.Emit("error", gin.H{"error": "Error processing current hand"})
			return
		}

		// Validate that all discarded cards are in the player's hand
		valid, errMsg := play_round.ValidatePlayerCards(hand, discard)
		if !valid {
			log.Printf("[DISCARD-ERROR] Invalid discard for user %s: %s", username, errMsg)
			client.Emit("error", gin.H{"error": errMsg})
			return
		}

		// 5. Get new cards from the deck
		newCards := deck.Draw(len(discard))
		if newCards == nil {
			client.Emit("error", gin.H{"error": "There are not enough cards available in the deck"})
			return
		}

		// Also update around here player hand
		// 6. Update the player's info in Redis
		deck.PlayedCards = append(deck.PlayedCards, discard...)
		deck.RemoveCards(newCards)
		player.CurrentDeck = deck.ToJSON()

		// Remove the discarded cards from the hand
		for _, card := range discard {
			for i, c := range hand {
				if c.Suit == card.Suit && c.Rank == card.Rank {
					hand = append(hand[:i], hand[i+1:]...)
					break
				}
			}
		}
		// Add the new cards to the hand
		hand = append(hand, newCards...)
		player.CurrentHand, err = json.Marshal(hand)
		if err != nil {
			log.Printf("[DISCARD-ERROR] Error serializing current hand: %v", err)
			client.Emit("error", gin.H{"error": "Error serializing current hand"})
			return
		}

		// Update discards left
		player.DiscardsLeft--

		err = redisClient.UpdateDeckPlayer(*player)
		if err != nil {
			log.Printf("[DISCARD-ERROR] Error updating player data: %v", err)
			client.Emit("error", gin.H{"error": "Error updating player data"})
			return
		}

		// 7. Prepare the response with the full deck state
		response := gin.H{
			"cards_discarded": discard,
			"current_hand":    player.CurrentHand,
			"left_discards":   player.DiscardsLeft,
			"played_cards":    len(deck.PlayedCards),
			"unplayed_cards":  len(deck.TotalCards) + len(hand),
			"new_cards":       newCards,
		}

		// 8. Send the response to the client
		client.Emit("discarded_cards", response)
		log.Printf("[DISCARD-SUCCESS] Sent updated deck to user %s (%d total cards)", username, response["deck_size"])
	}
}

func HandleGetFullDeck(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("GetFullDeck request - Usuario: %s, Socket ID: %s", username, client.Id())

		// Check if lobby ID is provided
		if len(args) < 1 {
			log.Printf("[DECK-ERROR] Missing lobby ID for user %s", username)
			client.Emit("error", gin.H{"error": "Missing lobby ID"})
			return
		}

		// Get the lobby ID from args
		lobbyID := args[0].(string)

		// Verify that the player is in the lobby
		isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
		if err != nil {
			log.Printf("[DECK-ERROR] Database error when checking lobby membership: %v", err)
			client.Emit("error", gin.H{"error": "Database error"})
			return
		}

		if !isInLobby {
			log.Printf("[DECK-ERROR] User %s is not in lobby %s", username, lobbyID)
			client.Emit("error", gin.H{"error": "You must join a game lobby first"})
			return
		}

		// Validate play round phase
		valid, err := socketio_utils.ValidatePlayRoundPhase(redisClient, client, lobbyID)
		if err != nil || !valid {
			// Error already emitted in ValidatePlayRoundPhase
			return
		}

		// 1. Get player's deck from Redis
		player, err := redisClient.GetInGamePlayer(username)
		if err != nil {
			log.Printf("[DECK-ERROR] Error getting player data: %v", err)
			client.Emit("error", gin.H{"error": "Error al obtener el mazo"})
			return
		}

		var deck *poker.Deck
		if player.CurrentDeck != nil {
			deck, err = poker.DeckFromJSON(player.CurrentDeck)
			if err != nil {
				log.Printf("[DECK-ERROR] Error parsing deck: %v", err)
				client.Emit("error", gin.H{"error": "Error al procesar el mazo"})
				return
			}
		} else {
			deck = &poker.Deck{
				TotalCards:  make([]poker.Card, 0),
				PlayedCards: make([]poker.Card, 0),
			}
		}

		// 3. Prepare response with complete deck state
		response := gin.H{
			"total_cards":  deck.TotalCards,  // Available cards
			"played_cards": deck.PlayedCards, // Discarded/used cards
			"deck_size":    len(deck.TotalCards) + len(deck.PlayedCards),
			"username":     username,
		}

		// 4. Send to client
		client.Emit("full_deck", response)
		log.Printf("Sent full deck to user %s (%d total cards)", username, response["deck_size"])
	}
}

func HandleActivateModifiers(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("ActivateModifier request - User: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())

		// 1. Check if the player is in the game
		lobbyID := args[0].(string)

		// Check player is in lobby
		isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
		if err != nil {
			log.Printf("[MODIFIER-ERROR] Database error: %v", err)
			client.Emit("error", gin.H{"error": "Database error"})
			return
		}

		if !isInLobby {
			log.Printf("[MODIFIER-ERROR] User is NOT in lobby: %s, Lobby: %s", username, lobbyID)
			client.Emit("error", gin.H{"error": "You must join the lobby before sending messages"})
			return
		}

		// Validate modifiers phase
		valid, err := socketio_utils.ValidateVouchersPhase(redisClient, client, lobbyID)
		if err != nil || !valid {
			return
		}

		if len(args) < 1 {
			log.Printf("[MODIFIER-ERROR] Missing arguments for user %s", username)
			client.Emit("error", gin.H{"error": "Missing modifier or lobby"})
			return
		}

		modifiers := args[1].([]poker.Modifier)

		player, err := redisClient.GetInGamePlayer(username)
		if err != nil {
			log.Printf("[MODIFIER-ERROR] Error getting player data: %v", err)
			client.Emit("error", gin.H{"error": "Error getting player data"})
			return
		}

		if player.Modifiers == nil {
			log.Printf("[MODIFIER-ERROR] No modifiers available for user %s", username)
			client.Emit("error", gin.H{"error": "No modifiers available"})
			return
		}

		var player_modifiers []poker.Modifier
		err = json.Unmarshal(player.Modifiers, &player_modifiers)
		if err != nil {
			log.Printf("[MODIFIER-ERROR] Error parsing modifiers: %v", err)
			client.Emit("error", gin.H{"error": "Error parsing modifiers"})
			return
		}

		// Check if the modifiers are available
		found := false
		var mod int
		for _, m := range player_modifiers {
			for _, modifier := range modifiers {
				if m == modifier {
					found = true
					mod = int(m.Value)
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			log.Printf("[MODIFIER-ERROR] Modifier %d not available for user %s", mod, username)
			client.Emit("error", gin.H{"error": "Modifier not available"})
			return
		}

		// Add the activated modifiers to the player
		var activated_modifiers []poker.Modifier
		activated_modifiers = append(activated_modifiers, modifiers...)
		activated_modifiersJSON, err := json.Marshal(activated_modifiers)
		if err != nil {
			log.Printf("[MODIFIER-ERROR] Error marshaling activated modifiers: %v", err)
			client.Emit("error", gin.H{"error": "Error processing modifiers"})
			return
		}
		player.ActivatedModifiers = activated_modifiersJSON
		log.Printf("[MODIFIER-INFO] Activated modifiers for user %s: %v", username, activated_modifiers)

		// Remove the activated modifier from the available modifiers
		for i, v := range player_modifiers {
			for _, value := range modifiers {
				if v == value {
					player_modifiers = append(player_modifiers[:i], player_modifiers[i+1:]...)
				}
			}
		}

		modifiersJSON, err := json.Marshal(player_modifiers)
		if err != nil {
			log.Printf("[MODIFIER-ERROR] Error marshaling modifiers: %v", err)
			client.Emit("error", gin.H{"error": "Error processing modifiers"})
			return
		}
		player.Modifiers = modifiersJSON

		err = redisClient.UpdateDeckPlayer(*player)
		if err != nil {
			log.Printf("[MODIFIER-ERROR] Error updating player data: %v", err)
			client.Emit("error", gin.H{"error": "Error updating player data"})
			return
		}

		// Emit success response
		client.Emit("modifiers_activated", gin.H{
			"modifiers": player.Modifiers,
			"activated": player.ActivatedModifiers,
		})
		log.Printf("[MODIFIER-SUCCESS] Modifiers activated for user %s", username)
	}
}

func HandleSendModifiers(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("ActivateModifier request - User: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())

		// 1. Check if the player is in the game
		lobbyID := args[0].(string)

		// Check player is in lobby
		isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
		if err != nil {
			log.Printf("[MODIFIER-ERROR] Database error: %v", err)
			client.Emit("error", gin.H{"error": "Database error"})
			return
		}

		if !isInLobby {
			log.Printf("[MODIFIER-ERROR] User is NOT in lobby: %s, Lobby: %s", username, lobbyID)
			client.Emit("error", gin.H{"error": "You must join the lobby before sending messages"})
			return
		}

		// Validate vouchers phase
		valid, err := socketio_utils.ValidateVouchersPhase(redisClient, client, lobbyID)
		if err != nil || !valid {
			return
		}

		if len(args) < 1 {
			log.Printf("[MODIFIER-ERROR] Missing arguments for user %s", username)
			client.Emit("error", gin.H{"error": "Missing modifier or lobby"})
			return
		}

		modifiers := args[1].([]poker.Modifier)

		player, err := redisClient.GetInGamePlayer(username)
		if err != nil {
			log.Printf("[MODIFIER-ERROR] Error getting player data: %v", err)
			client.Emit("error", gin.H{"error": "Error getting player data"})
			return
		}

		if player.Modifiers == nil {
			log.Printf("[MODIFIER-ERROR] No modifiers available for user %s", username)
			client.Emit("error", gin.H{"error": "No modifiers available"})
			return
		}

		var player_modifiers []poker.Modifier
		err = json.Unmarshal(player.Modifiers, &player_modifiers)
		if err != nil {
			log.Printf("[MODIFIER-ERROR] Error parsing modifiers: %v", err)
			client.Emit("error", gin.H{"error": "Error parsing modifiers"})
			return
		}

		// Check if the modifiers are available
		found := false
		var mod int
		for _, m := range player_modifiers {
			for _, modifier := range modifiers {
				if m == modifier {
					found = true
					mod = int(m.Value)
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			log.Printf("[MODIFIER-ERROR] Modifier %d not available for user %s", mod, username)
			client.Emit("error", gin.H{"error": "Modifier not available"})
			return
		}

		// Remove the activated modifier from the available modifiers
		for i, v := range player_modifiers {
			for _, value := range modifiers {
				if v == value {
					player_modifiers = append(player_modifiers[:i], player_modifiers[i+1:]...)
				}
			}
		}

		modifiersJSON, err := json.Marshal(player_modifiers)
		if err != nil {
			log.Printf("[MODIFIER-ERROR] Error marshaling modifiers: %v", err)
			client.Emit("error", gin.H{"error": "Error processing modifiers"})
			return
		}
		player.Modifiers = modifiersJSON

		err = redisClient.UpdateDeckPlayer(*player)
		if err != nil {
			log.Printf("[MODIFIER-ERROR] Error updating player data: %v", err)
			client.Emit("error", gin.H{"error": "Error updating player data"})
			return
		}

		request_player := args[2].(string)

		// Update the receiving player

		receiver, err := redisClient.GetInGamePlayer(request_player)
		if err != nil {
			log.Printf("[MODIFIER-ERROR] Error getting player data: %v", err)
			client.Emit("error", gin.H{"error": "Error getting player data"})
			return
		}

		// Add the activated modifiers to the player
		var activated_modifiers []poker.ReceivedModifier
		for _, modifier := range modifiers {
			activated_modifiers = append(activated_modifiers, poker.ReceivedModifier{
				Modifier: modifier,
				Sender:   username,
			})
		}
		activated_modifiersJSON, err := json.Marshal(activated_modifiers)
		if err != nil {
			log.Printf("[MODIFIER-ERROR] Error marshaling activated modifiers: %v", err)
			client.Emit("error", gin.H{"error": "Error processing modifiers"})
			return
		}

		// TODO: con esto no estaríamos mezclando los modificadores recibidos con los activados
		// (que igual el usuario ha activado por sí mismo)?
		receiver.ReceivedModifiers = activated_modifiersJSON
		log.Printf("[MODIFIER-INFO] Activated modifiers for user %s: %v", receiver.Username, activated_modifiers)

		// Notify the receiving player
		sio.UserConnections[receiver.Username].Emit("modifiers_received", gin.H{
			"modifiers": activated_modifiers,
			"sender":    username,
		})

		// Notify the sender
		client.Emit("modifiers_sended", gin.H{
			"modifiers": player.Modifiers,
		})

		log.Printf("[MODIFIER-SUCCESS] Modifiers sent to user %s from %s", request_player, username)

	}
}

/*
func HandlePlayVoucher(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
			if len(args) < 3 {
				log.Printf("[VOUCHER-ERROR] Faltan argumentos para usuario %s", username)
				client.Emit("error", gin.H{"error": "Faltan parametros"})
				return
			}

			// Assert as float64 first
			voucher_id_float, ok := args[0].(float64)
			if !ok {
				log.Printf("[VOUCHER-ERROR] Invalid type for proposed blind: expected number, got %T", args[0])
				client.Emit("error", gin.H{"error": "Summoned voucher ID is not valid"})
				return
			}
			// Convert float64 to int
			voucher_id := int(voucher_id_float)

			vector_de_usuarios := args[1].(string) // Asumiré que es un string, entiendo q es un JSOBN

			lobbyID := args[2].(string)

			// Get the lobby
			lobby, err := redisClient.GetGameLobby(lobbyID)
			if err != nil {
				return nil, fmt.Errorf("error getting lobby: %v", err)
			}

			// Check if user_vector is empty, in which case it will affect himself

			if len(vector_de_usuarios) == 0 { // Se lo aplica a si mismo
				voucherMap[voucher_id]()
			} else {
				// Aplica al resto
				//Parsear
				//
			}

			// Broadcast the eliminated players
			sio.Sio_server.To(socket.Room(lobbyID)).Emit("recive_voucher", gin.H{
				"voucher_id": voucher_id,
				"username":   username,
			})
	}
}
*/
