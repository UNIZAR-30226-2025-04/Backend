package game_flow

import (
	game_constants "Nogler/constants/game"
	redis_models "Nogler/models/redis"
	"Nogler/services/poker"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	socketio_utils "Nogler/services/socket_io/utils"
	"Nogler/services/socket_io/utils/stages/shop"
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

const username = "Noglerinho" // AI username

// BLIND

func ProposeBlindAI(redisClient *redis.RedisClient, lobbyID string, sio *socketio_types.SocketServer) {

	log.Printf("[AI-BLIND] %s is proposing a blind", username)

	// Get the lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[AI-BLIND-ERROR] Error getting game lobby: %v", err)
		return
	}

	// Validate blind phase
	valid, err := socketio_utils.ValidateBlindPhase(redisClient, nil, lobbyID)
	if err != nil || !valid {
		// Error already emitted in ValidateBlindPhase
		return
	}

	AI, err := redisClient.GetAIfromLobby(lobbyID)
	if err != nil {
		log.Printf("[AI-BLIND-ERROR] Error getting player data: %v", err)
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
		return
	}

	currentBlind, err := redisClient.GetCurrentBlind(lobbyID)
	if err != nil {
		log.Printf("[AI-BLIND-ERROR] Error getting current blind: %v", err)
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
		return
	}

	// Update current blind if this proposal is higher
	if proposedBlind > currentBlind {
		err := redisClient.SetCurrentHighBlind(lobbyID, proposedBlind, username)
		if err != nil {
			log.Printf("[AI-BLIND-ERROR] Could not update current blind: %v", err)
			return
		}

		// Simulate a delay for the AI to propose the blind
		time.Sleep(2 * time.Second)

		// Broadcast the new blind value to everyone in the lobby
		sio.Sio_server.To(socket.Room(lobbyID)).Emit("AI_blind_updated", gin.H{
			"old_max_blind": currentBlind,
			"new_blind":     proposedBlind,
			"proposed_by":   username,
		})
	}
}

// PLAY

func PlayHandIA(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {
	log.Printf("PlayHandsAI started - Usuario: %s", username)

	// 1. Get player data from Redis to extract lobby ID
	player, err := redisClient.GetAIfromLobby(lobbyID)
	if err != nil {
		log.Printf("[HAND-ERROR] Error getting player data: %v", err)
		return
	}

	// Validate play round phase
	valid, err := socketio_utils.ValidatePlayRoundPhase(redisClient, nil, lobbyID)
	if err != nil || !valid {
		// Error already emitted in ValidatePlayRoundPhase
		return
	}

	for i := 0; i < 6; i++ {

		// 2. Check if the player has enough plays left
		if player.HandPlaysLeft <= 0 {
			log.Printf("[HAND-ERROR] No hand plays left %s", username)
			return
		}

		// Get the current hand from Redis
		var currentHand []poker.Card
		err = json.Unmarshal(player.CurrentHand, &currentHand)
		if err != nil {
			log.Printf("[HAND-ERROR] Error parsing the hand %v", err)
			return
		}

		// 3. Calculate base points

		// Generate all combinations of 4 cards from the current hand
		combinations := poker.GenerateHands(currentHand, 4)

		// Iterate through all combinations to find the best hand
		var bestTokens, bestMult int = 0, 0
		var bestHandType int
		var bestScoredCards []poker.Card
		var bestHand poker.Hand

		var jokers poker.Jokers
		err = json.Unmarshal(player.CurrentJokers, &jokers)
		if err != nil {
			log.Printf("[HAND-ERROR] Error parsing jokers: %v", err)
			return
		}

		for _, combination := range combinations {
			hand := poker.Hand{
				Cards:  combination,
				Jokers: jokers,
				Gold:   player.PlayersMoney,
			}
			tokens, mult, handType, scoredCards := poker.BestHand(hand)
			if tokens*mult > bestTokens*bestMult {
				bestTokens = tokens
				bestMult = mult
				bestHandType = handType
				bestScoredCards = scoredCards
				bestHand = hand
			}
		}

		if bestHandType > 10 && player.DiscardsLeft > 0 {
			// Get 1 or 2 or 3 worst cards to discard
			size := rand.Intn(3) + 1
			poker.SortCards(currentHand)
			worstCards := currentHand[:size]
			discardCardsAI(redisClient, player, lobbyID, worstCards) // Discard the worst cards
			continue
		}

		enhancedFichas, enhancedMult := poker.ApplyEnhancements(bestTokens, bestMult, bestScoredCards)

		// 4. Apply jokers (passing the hand which contains the jokers)
		finalFichas, finalMult, finalGold, _ := poker.ApplyJokers(bestHand, bestHand.Jokers, enhancedFichas, enhancedMult, bestHand.Gold)

		// 5. Apply modifiers

		// Get the most played hand from the player
		var mostPlayedHand poker.Hand
		if player.MostPlayedHand != nil {
			err = json.Unmarshal(player.MostPlayedHand, &mostPlayedHand)
			if err != nil {
				log.Printf("[AI-HAND-ERROR] Error parsing most played hand: %v", err)
				return
			}
		}

		// Apply activated modifiers
		var activatedModifiers poker.Modifiers
		if player.ActivatedModifiers != nil {
			err = json.Unmarshal(player.ActivatedModifiers, &activatedModifiers)
			if err != nil {
				log.Printf("[AI-HAND-ERROR] Error parsing activated modifiers: %v", err)
				return
			}
		}

		finalFichas, finalMult, finalGold = poker.ApplyModifiers(bestHand, mostPlayedHand, &activatedModifiers, finalFichas, finalMult, finalGold)
		if err != nil {
			log.Printf("[AI-HAND-ERROR] Error applying modifiers: %v", err)
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
			log.Printf("[AI-HAND-ERROR] Error serializing activated modifiers: %v", err)
			return
		}

		// Apply received modifiers
		var receivedModifiers poker.Modifiers
		if player.ActivatedModifiers != nil {
			err = json.Unmarshal(player.ReceivedModifiers, &receivedModifiers)
			if err != nil {
				log.Printf("[AI-HAND-ERROR] Error parsing received modifiers: %v", err)
				return
			}
		}

		finalFichas, finalMult, finalGold = poker.ApplyModifiers(bestHand, mostPlayedHand, &activatedModifiers, finalFichas, finalMult, finalGold)
		if err != nil {
			log.Printf("[AI-HAND-ERROR] Error applying modifiers: %v", err)
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
			log.Printf("[AI-HAND-ERROR] Error serializing received modifiers: %v", err)
			return
		}

		// 6. Update player data in Redis
		// Delete the played hand from the current hand
		for _, card := range bestHand.Cards {
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
				log.Printf("[AI-DECK-ERROR] Error parsing deck: %v", err)
				return
			}
		} else {
			deck = &poker.Deck{
				TotalCards:  make([]poker.Card, 0),
				PlayedCards: make([]poker.Card, 0),
			}
		}

		// Get new cards from the deck
		newCards := deck.Draw(len(bestHand.Cards))
		if newCards == nil {
			log.Printf("[AI-DECK-ERROR] Not enough cards in the deck")
			return
		}
		// Add the new cards to the hand
		currentHand = append(currentHand, newCards...)
		player.CurrentHand, err = json.Marshal(currentHand)
		if err != nil {
			log.Printf("[AI-HAND-ERROR] Error serializing current hand: %v", err)
			return
		}
		// Add the played hand to the played cards
		deck.PlayedCards = append(deck.PlayedCards, bestHand.Cards...)
		// Remove the played hand from the deck
		deck.RemoveCards(newCards)
		player.CurrentDeck = deck.ToJSON()

		player.CurrentPoints = valorFinal
		player.TotalPoints += valorFinal
		player.HandPlaysLeft--
		err = redisClient.UpdateDeckPlayer(*player)
		if err != nil {
			log.Printf("[AI-HAND-ERROR] Error updating player data: %v", err)
			return
		}

		// Log the result
		log.Println("Jugador ha puntuado la friolera de:", valorFinal)
		// 7. Emit success response (FRONTEND WILL USE IT??????? SOME OF THEM????)
		/*
			client.Emit("AI_played_hand", gin.H{
				"total_score":         valorFinal,
				"gold":                finalGold,
				"jokersTriggered":     jokersTriggered,
				"left_plays":          player.HandPlaysLeft,
				"activated_modifiers": activatedModifiers,
				"received_modifiers":  receivedModifiers,
				"played_cards":        len(deck.PlayedCards),
				"unplayed_cards":      len(deck.TotalCards) + len(currentHand),
				"new_cards":           newCards,
				"red_score":           finalMult,
				"blue_score":          finalFichas,
			})
		*/

		// NOTE: check it outside the `if` sentence, since the player might have reached the blind
		checkAIFinishedRound(redisClient, db, lobbyID, sio)
	}
}

func discardCardsAI(redisClient *redis.RedisClient, player *redis_models.InGamePlayer,
	lobbyID string, discard []poker.Card) {

	var deck *poker.Deck
	var err error
	if player.CurrentDeck != nil {
		deck, err = poker.DeckFromJSON(player.CurrentDeck)
		if err != nil {
			log.Printf("[AI-DISCARD-ERROR] Error parsing deck: %v", err)
			return
		}
	} else {
		deck = &poker.Deck{
			TotalCards:  make([]poker.Card, 0),
			PlayedCards: make([]poker.Card, 0),
		}
	}

	// Get the current hand
	var hand []poker.Card
	err = json.Unmarshal(player.CurrentHand, &hand)
	if err != nil {
		log.Printf("[AI-GET_CARDS-ERROR] Error unmarshaling current hand: %v", err)
		return
	}

	// 5. Get new cards from the deck
	newCards := deck.Draw(len(discard))
	if newCards == nil {
		return
	}

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
		log.Printf("[AI-DISCARD-ERROR] Error serializing current hand: %v", err)
		return
	}

	// Update discards left
	player.DiscardsLeft--

	err = redisClient.UpdateDeckPlayer(*player)
	if err != nil {
		log.Printf("[AI-DISCARD-ERROR] Error updating player data: %v", err)
		return
	}
}

func checkAIFinishedRound(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {

	log.Printf("[AI-ROUND-CHECK] Checking if player %s has finished round in lobby %s", username, lobbyID)

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

			HandleRoundPlayEnd(redisClient, db, lobbyID, sio, lobby.CurrentRound)
		}
	}
}

// SHOP

func ShopAI(redisClient *redis.RedisClient, lobbyID string, shopState *redis_models.LobbyShop) {
	log.Printf("ShopAI initiated - User: %s", username)

	// Get player state first to extract lobby ID
	playerState, err := redisClient.GetAIfromLobby(username)
	if err != nil {
		log.Printf("[AI-SHOP-ERROR] Error getting player state: %v", err)
		return
	}

	log.Printf("[AI-INFO] Getting lobby ID info: %s for AI: %s", lobbyID, username)

	valid, err := socketio_utils.ValidateShopPhase(redisClient, nil, lobbyID)
	if err != nil || !valid {
		return
	}

	lobbyState, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[AI-SHOP-ERROR] Error getting lobby state: %v", err)
		return
	}

	if lobbyState.ShopState == nil {
		return
	}

	// If AI has less than 40 money, sell a joker if exists
	if playerState.PlayersMoney < 40 {
		// 33% chance to sell a joker
		randomValue := rand.Intn(3)
		if randomValue == 0 {
			var jokers poker.Jokers
			err = json.Unmarshal(playerState.CurrentJokers, &jokers)
			if err != nil {
				log.Printf("[AI-SHOP-ERROR] Error parsing jokers: %v", err)
				return
			}
			var numJokers int
			for i := 0; i < len(jokers.Juglares); i++ {
				if jokers.Juglares[i] != 0 {
					numJokers++
				}
			}
			if numJokers == 0 {
				log.Printf("[AI-SHOP-ERROR] No jokers to sell for player %s", username)
				return
			} else {
				jokerToSell := rand.Intn(numJokers)
				sellJokerAI(redisClient, playerState, jokers.Juglares[jokerToSell])
				// If the AI has more than 3 jokers, sell another one
				if numJokers > 3 {
					// Sell other joker
					jokerToSell2 := rand.Intn(numJokers)
					for jokerToSell2 == jokerToSell {
						jokerToSell2 = rand.Intn(numJokers)
					}
					if jokerToSell2 != jokerToSell {
						sellJokerAI(redisClient, playerState, jokers.Juglares[jokerToSell2])
					}
				}
				return
			}
		}
	} else {
		// Order to buy pack (0), joker (1) or voucher (2)
		var order []int
		for i := 0; i < 3; i++ {
			randomValue := rand.Intn(3)
			for j := 0; j < len(order); j++ {
				if order[j] == randomValue {
					randomValue = rand.Intn(3)
					j = -1
				}
			}
			order = append(order, randomValue)
		}

		// Until which one do we buy?
		until := rand.Intn(3)
		for i := 0; i <= until; i++ {
			// How many do we buy? (1 or 2)
			howMany := rand.Intn(2) + 1
			for j := 0; j < howMany; j++ {
				switch order[i] {
				case 0:
					// Buy pack
					// Which pack?
					which := rand.Intn(len(shopState.FixedPacks))
					item := shopState.FixedPacks[which]
					itemID := item.ID
					price := shopState.FixedPacks[which].Price
					purchasePackAI(redisClient, playerState, lobbyState, item, itemID, price)
				case 1:
					// Buy joker
					// Which joker?
					which := rand.Intn(len(shopState.RerollableItems))
					item := shopState.RerollableItems[which]
					itemID := item.ID
					price := shopState.RerollableItems[which].Price
					purchaseJokerAI(redisClient, playerState, lobbyState, item, itemID, price)
				case 2:
					// Buy voucher
					// Which voucher?
					which := rand.Intn(len(shopState.FixedModifiers))
					item := shopState.FixedModifiers[which]
					itemID := item.ID
					price := shopState.FixedModifiers[which].Price
					purchaseVoucherAI(redisClient, playerState, lobbyState, item, itemID, price)
				}
			}
		}
	}
}

func purchasePackAI(redisClient *redis.RedisClient, playerState *redis_models.InGamePlayer,
	lobbyState *redis_models.GameLobby, item redis_models.ShopItem, itemID int, clientPrice int) {

	// Validate the purchase
	if err := shop.ValidatePurchase(item, game_constants.PACK_TYPE, clientPrice, playerState); err != nil {
		log.Printf("[SHOP-ERROR] Purchase validation failed: %v", err)
		return
	}

	_, err := shop.GetOrGeneratePackContents(redisClient, lobbyState, item)
	if err != nil {
		return
	}

	// Update player's LastPurchasedPackItemId and deduct money
	// NOTE: potential exploit by not sending a pack selection event and
	// then reusing this same id during the next round. Already fixed by resetting
	// LastPurchasedPackItemId to -1 when starting the shop phase
	playerState.LastPurchasedPackItemId = itemID
	playerState.PlayersMoney -= item.Price // Deduct the money

	// Save the updated player state
	if err := redisClient.SaveInGamePlayer(playerState); err != nil {
		log.Printf("[AI-SHOP-ERROR] Error saving player state: %v", err)
		return
	}
}

func purchaseJokerAI(redisClient *redis.RedisClient, playerState *redis_models.InGamePlayer,
	lobbyState *redis_models.GameLobby, item redis_models.ShopItem, itemID int, clientPrice int) {

	// Process the joker purchase with price validation
	success, updatedPlayer, err := shop.PurchaseJoker(redisClient, playerState, item, clientPrice)
	if err != nil || !success {
		log.Printf("[AI-SHOP-ERROR] Purchase failed: %v", err)
		return
	}

	// Save the updated player state
	if err := redisClient.SaveInGamePlayer(updatedPlayer); err != nil {
		log.Printf("[AI-SHOP-ERROR] Error saving player state: %v", err)
		return
	}
}

func purchaseVoucherAI(redisClient *redis.RedisClient, playerState *redis_models.InGamePlayer,
	lobbyState *redis_models.GameLobby, item redis_models.ShopItem, itemID int, clientPrice int) {

	// Process the voucher purchase with price validation
	success, updatedPlayer, err := shop.PurchaseVoucher(redisClient, playerState, item, clientPrice)
	if err != nil || !success {
		log.Printf("[AI-SHOP-ERROR] Purchase failed: %v", err)
		return
	}

	// Save the updated player state
	if err := redisClient.SaveInGamePlayer(updatedPlayer); err != nil {
		log.Printf("[AI-SHOP-ERROR] Error saving player state: %v", err)
		return
	}
}

func sellJokerAI(redisClient *redis.RedisClient, playerState *redis_models.InGamePlayer, jokerID int) {

	// Process the joker sale
	updatedPlayer, _, err := shop.SellJoker(playerState, jokerID)
	if err != nil {
		log.Printf("[AI-SHOP-ERROR] Sale failed: %v", err)
		return
	}

	// Save the updated player state
	if err := redisClient.SaveInGamePlayer(updatedPlayer); err != nil {
		log.Printf("[AI-SHOP-ERROR] Error saving player state: %v", err)
		return
	}
}

// VOUCHERS

func VouchersAI(redisClient *redis.RedisClient, lobbyID string, sio *socketio_types.SocketServer) {
	log.Printf("VouchersAI initiated - User: %s", username)

	// Get player data from Redis
	player, err := redisClient.GetAIfromLobby(lobbyID)
	if err != nil {
		log.Printf("[AI-VOUCHER-ERROR] Error getting player data: %v", err)
		return
	}

	// Validate vouchers phase
	valid, err := socketio_utils.ValidateVouchersPhase(redisClient, nil, lobbyID)
	if err != nil || !valid {
		return
	}

	var modifiers poker.Modifiers
	err = json.Unmarshal(player.Modifiers, &modifiers)
	if err != nil {
		log.Printf("[AI-VOUCHER-ERROR] Error parsing modifiers: %v", err)
		return
	}

	// Activate vouchers
	numModifiers := len(modifiers.Modificadores)
	if numModifiers > 0 {
		// Order vouchers
		var order []int
		for i := 0; i < numModifiers; i++ {
			randomValue := rand.Intn(numModifiers)
			for j := 0; j < len(order); j++ {
				if order[j] == randomValue {
					randomValue = rand.Intn(3)
					j = -1
				}
			}
			order = append(order, randomValue)
		}
		// How many vouchers to activate?
		numVouchers := rand.Intn(numModifiers + 1)
		for i := 0; i < numVouchers; i++ {
			if modifiers.Modificadores[i].Value == 0 {
				continue
			}
			// If the voucher is "EvilEye", send it to the opponent
			if modifiers.Modificadores[i].Value == 1 {
				sendVoucherAI(redisClient, player, lobbyID, modifiers.Modificadores[i], sio)
			} else {
				activateVoucherAI(redisClient, player, modifiers.Modificadores[i])
			}
		}
	}
}

func activateVoucherAI(redisClient *redis.RedisClient, player *redis_models.InGamePlayer,
	modifier poker.Modifier) {

	var player_modifiers []poker.Modifier
	err := json.Unmarshal(player.Modifiers, &player_modifiers)
	if err != nil {
		log.Printf("[AI-MODIFIER-ERROR] Error parsing modifiers: %v", err)
		return
	}

	// Check if the modifier is available
	found := false
	var mod int
	for _, m := range player_modifiers {
		if m == modifier {
			found = true
			mod = int(m.Value)
			break
		}
		if found {
			break
		}
	}
	if !found {
		log.Printf("[AI-MODIFIER-ERROR] Modifier %d not available for user %s", mod, username)
		return
	}

	// Add the activated modifiers to the player
	var activated_modifiers []poker.Modifier
	activated_modifiers = append(activated_modifiers, modifier)
	activated_modifiersJSON, err := json.Marshal(activated_modifiers)
	if err != nil {
		log.Printf("[AI-MODIFIER-ERROR] Error marshaling activated modifiers: %v", err)
		return
	}
	player.ActivatedModifiers = activated_modifiersJSON
	log.Printf("[AI-MODIFIER-INFO] Activated modifiers for user %s: %v", username, activated_modifiers)

	// Remove the activated modifier from the available modifiers
	for i, v := range player_modifiers {
		if v == modifier {
			found = true
			player_modifiers = append(player_modifiers[:i], player_modifiers[i+1:]...)
		}
		if found {
			break
		}
	}

	modifiersJSON, err := json.Marshal(player_modifiers)
	if err != nil {
		log.Printf("[AI-MODIFIER-ERROR] Error marshaling modifiers: %v", err)
		return
	}
	player.Modifiers = modifiersJSON

	err = redisClient.UpdateDeckPlayer(*player)
	if err != nil {
		log.Printf("[AI-MODIFIER-ERROR] Error updating player data: %v", err)
		return
	}

}

func sendVoucherAI(redisClient *redis.RedisClient, player *redis_models.InGamePlayer,
	lobbyID string, modifier poker.Modifier, sio *socketio_types.SocketServer) {

	var player_modifiers []poker.Modifier
	err := json.Unmarshal(player.Modifiers, &player_modifiers)
	if err != nil {
		log.Printf("[AI-MODIFIER-ERROR] Error parsing modifiers: %v", err)
		return
	}

	// Check if the modifiers are available
	found := false
	var mod int
	for _, m := range player_modifiers {
		if m == modifier {
			found = true
			mod = int(m.Value)
			break
		}
		if found {
			break
		}
	}
	if !found {
		log.Printf("[AI-MODIFIER-ERROR] Modifier %d not available for user %s", mod, username)
		return
	}

	// Remove the activated modifier from the available modifiers
	for i, v := range player_modifiers {
		if v == modifier {
			found = true
			player_modifiers = append(player_modifiers[:i], player_modifiers[i+1:]...)
		}
		if found {
			break
		}
	}

	modifiersJSON, err := json.Marshal(player_modifiers)
	if err != nil {
		log.Printf("[AI-MODIFIER-ERROR] Error marshaling modifiers: %v", err)
		return
	}
	player.Modifiers = modifiersJSON

	err = redisClient.UpdateDeckPlayer(*player)
	if err != nil {
		log.Printf("[AI-MODIFIER-ERROR] Error updating player data: %v", err)
		return
	}

	// Get the opponent's username
	players, err := redisClient.GetAllPlayersInLobby(lobbyID)
	if err != nil {
		log.Printf("[AI-MODIFIER-ERROR] Error getting players in lobby: %v", err)
		return
	}
	var request_player string
	if players[0].IsBot {
		request_player = players[1].Username
	} else {
		request_player = players[0].Username
	}

	// Update the receiving player

	receiver, err := redisClient.GetInGamePlayer(request_player)
	if err != nil {
		log.Printf("[AI-MODIFIER-ERROR] Error getting player data: %v", err)
		return
	}

	// Add the activated modifiers to the player
	var activated_modifiers []poker.ReceivedModifier
	activated_modifiers = append(activated_modifiers, poker.ReceivedModifier{
		Modifier: modifier,
		Sender:   username,
	})

	activated_modifiersJSON, err := json.Marshal(activated_modifiers)
	if err != nil {
		log.Printf("[AI-MODIFIER-ERROR] Error marshaling activated modifiers: %v", err)
		return
	}

	receiver.ReceivedModifiers = activated_modifiersJSON
	log.Printf("[AI-MODIFIER-INFO] Activated modifiers for user %s: %v", receiver.Username, activated_modifiers)

	// Notify the receiving player
	sio.Sio_server.To(socket.Room(lobbyID)).Emit("AI_modifiers_received", gin.H{
		"modifiers": receiver.ReceivedModifiers,
		"sender":    username,
	})

	log.Printf("[AI-MODIFIER-SUCCESS] Modifiers sent to user %s from %s: %v", request_player, username, modifier)
}
