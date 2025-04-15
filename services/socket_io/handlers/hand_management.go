package handlers

import (
	"Nogler/services/poker"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	socketio_utils "Nogler/services/socket_io/utils"
	"Nogler/services/socket_io/utils/game_flow"
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

		// 1. Check if the player is in the game
		lobbyID := args[1].(string)

		// Check player is in lobby
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
		player, err := redisClient.GetInGamePlayer(username)
		if err != nil {
			log.Printf("[HAND-ERROR] Error getting player data: %v", err)
			client.Emit("error", gin.H{"error": "Error al obtener los datos del jugador"})
			return
		}
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

		// 3. Calculate base points
		fichas, mult, _ := poker.BestHand(hand)

		// 4. Apply jokers (passing the hand which contains the jokers)
		finalFichas, finalMult, finalGold, jokersTriggered := poker.ApplyJokers(hand, hand.Jokers, fichas, mult, hand.Gold)
		valorFinal := finalFichas * finalMult

		// 5. Update player data in Redis
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
		// 6. Emit success response
		client.Emit("played_hand", gin.H{
			"points":          valorFinal,
			"gold":            finalGold,
			"jokersTriggered": jokersTriggered,
			"message":         "¡Mano jugada con éxito!",
		})

		// 7. If the player has no plays left, emit a message
		if player.HandPlaysLeft <= 0 {
			client.Emit("no_plays_left", gin.H{"message": "No hand plays left"})
			log.Printf("[HAND-NO-PLAYS] User %s has no plays left", username)

			checkPlayerFinishedRound(redisClient, db, username, lobbyID, sio)
		}

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

	// Check if player has no plays and discards left
	if player.HandPlaysLeft <= 0 && player.DiscardsLeft <= 0 {
		log.Printf("[ROUND-CHECK] Player %s has finished their round (no plays or discards left)", username)

		// Get the lobby
		lobby, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			log.Printf("[ROUND-CHECK-ERROR] Error getting lobby: %v", err)
			return
		}

		// Increment the counter of players who finished the round (NEW, using a map to avoid same user incrementing the counter several times)
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

// Do this function plis TODO
func HandleDrawCards(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("DrawCards request - User: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())

		// 1. Check if the player is in the game
		lobbyID := args[1].(string)

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

		// 2. Get player's deck from Redis
		player, err := redisClient.GetInGamePlayer(username)
		if err != nil {
			log.Printf("[DECK-ERROR] Error getting player data: %v", err)
			client.Emit("error", gin.H{"error": "Error al obtener el mazo"})
			return
		}

		// Check if the user has enough draws left
		if player.DiscardsLeft <= 0 {
			log.Printf("[HAND-ERROR] No draws left for user %s", username)
			client.Emit("error", gin.H{"error": "No draws left"})
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

		handData, ok := args[0].(map[string]interface{})
		if !ok {
			log.Printf("[DRAW-ERROR] Invalid argument type for user %s. Expected map[string]interface{}, got %T", username, args[0])
			client.Emit("error", gin.H{"error": "Formato de datos inválido"})
			return
		}

		// Convert handData to JSON
		handJson, err := json.Marshal(handData)
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

		// 3. Determine how many cards the player needs
		cardsNeeded := 8 - len(hand.Cards)
		if cardsNeeded <= 0 {
			client.Emit("error", gin.H{"error": "El jugador ya tiene suficientes cartas"})
			return
		}

		// 4. Get the necessary cards
		newCards := deck.Draw(cardsNeeded)
		if newCards == nil {
			client.Emit("error", gin.H{"error": "No hay suficientes cartas disponibles en el mazo"})
			return
		}

		// Serialize new cards and the full deck
		newCardsJson, _ := json.Marshal(newCards)
		totalCardsJson, _ := json.Marshal(deck.TotalCards)

		// 5. Update the player's deck in Redis
		deck.RemoveCards(newCards)
		player.CurrentDeck = deck.ToJSON()
		player.DiscardsLeft--
		err = redisClient.UpdateDeckPlayer(*player)
		if err != nil {
			log.Printf("[DECK-ERROR] Error updating player data: %v", err)
			client.Emit("error", gin.H{"error": "Error updating player data"})
			return
		}

		// 6. Prepare the response with the full deck state
		response := gin.H{
			"new_cards":   string(newCardsJson),
			"total_cards": string(totalCardsJson),
			"deck_size":   len(deck.TotalCards),
		}

		// 7. Send the response to the client
		client.Emit("drawed_cards", response)
		log.Printf("[DRAW-SUCCESS] Sent updated deck to user %s (%d total cards)", username, response["deck_size"])

		// 8. If the player has no draws left, emit a message
		if player.DiscardsLeft <= 0 {
			client.Emit("no_draws_left", gin.H{"message": "No draws left"})
			log.Printf("[DRAW-NO-DRAWS] User %s has no draws left", username)
			checkPlayerFinishedRound(redisClient, db, username, lobbyID, sio)
		}
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
		valid, err := socketio_utils.ValidateModifiersPhase(redisClient, client, lobbyID)
		if err != nil || !valid {
			return
		}

		if len(args) < 1 {
			log.Printf("[MODIFIER-ERROR] Missing arguments for user %s", username)
			client.Emit("error", gin.H{"error": "Missing modifier or lobby"})
			return
		}

		modifiers := args[1].([]redis.Modifier)

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

		var player_modifiers []redis.Modifier
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
		var activated_modifiers []redis.Modifier
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

		// Validate modifiers phase
		valid, err := socketio_utils.ValidateModifiersPhase(redisClient, client, lobbyID)
		if err != nil || !valid {
			return
		}

		if len(args) < 1 {
			log.Printf("[MODIFIER-ERROR] Missing arguments for user %s", username)
			client.Emit("error", gin.H{"error": "Missing modifier or lobby"})
			return
		}

		modifiers := args[1].([]redis.Modifier)

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

		var player_modifiers []redis.Modifier
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
		var activated_modifiers []redis.Modifier
		activated_modifiers = append(activated_modifiers, modifiers...)
		activated_modifiersJSON, err := json.Marshal(activated_modifiers)
		if err != nil {
			log.Printf("[MODIFIER-ERROR] Error marshaling activated modifiers: %v", err)
			client.Emit("error", gin.H{"error": "Error processing modifiers"})
			return
		}
		receiver.ReceivedModifiers = activated_modifiersJSON
		log.Printf("[MODIFIER-INFO] Activated modifiers for user %s: %v", receiver.Username, activated_modifiers)

		// Notify the receiving player
		sio.UserConnections[receiver.Username].Emit("modifiers_received", gin.H{
			"modifiers": receiver.ReceivedModifiers,
			"sender":    username,
		})

		// Notify the sender
		client.Emit("modifiers_sended", gin.H{
			"modifiers": player.Modifiers,
		})

		log.Printf("[MODIFIER-SUCCESS] Modifiers sent to user %s from %s", request_player, username)

	}
}
