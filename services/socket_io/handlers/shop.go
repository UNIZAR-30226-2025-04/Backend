package handlers

import (
	game_constants "Nogler/constants/game"
	"fmt"

	"Nogler/services/poker"
	redis_services "Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	socketio_utils "Nogler/services/socket_io/utils"
	"Nogler/services/socket_io/utils/stages/shop"
	"encoding/json"
	"log"

	"golang.org/x/exp/rand"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

// Handler that will be called.
// TODO, we should call shop.ValidatePurchase and decrement the player's balance afterwards
// Also, the user will send a selection event, with the selected jokers and cards. We should
// validate that he has actually bought the pack and that the selected items were in that pack,
// and then add those items to the player's inventory.
func HandlePurchasePack(redisClient *redis_services.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("OpenPack iniciado - Usuario: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())

		if len(args) < 2 { // Changed from 1 to 2 to require pack ID and price
			log.Printf("[HAND-ERROR] Faltan argumentos para usuario %s", username)
			client.Emit("error", gin.H{"error": "Falta el pack a abrir o el precio"})
			return
		}

		// Handle pack ID (JavaScript numbers come as float64)
		itemIDFloat, ok := args[0].(float64)
		if !ok {
			client.Emit("error", gin.H{"error": "El pack ID debe ser un número"})
			return
		}
		itemID := int(itemIDFloat) // Convert to int

		// Get client-provided price
		priceFloat, ok := args[1].(float64)
		if !ok {
			client.Emit("error", gin.H{"error": "El precio debe ser un número"})
			return
		}
		clientPrice := int(priceFloat) // Convert to int

		// Get player state first to extract lobby ID
		playerState, err := redisClient.GetInGamePlayer(username)
		if err != nil {
			log.Printf("[SHOP-ERROR] Error getting player state: %v", err)
			client.Emit("error", gin.H{"error": "Error retrieving player state"})
			return
		}

		// Extract lobby ID from player state
		lobbyID := playerState.LobbyId
		if lobbyID == "" {
			log.Printf("[SHOP-ERROR] Player %s not associated with any lobby", username)
			client.Emit("error", gin.H{"error": "Player not in a lobby"})
			return
		}

		log.Printf("[INFO] Obteniendo información del lobby ID: %s para usuario: %s", lobbyID, username)

		// Validate that we are in the shop phase
		valid, err := socketio_utils.ValidateShopPhase(redisClient, client, lobbyID)
		if err != nil || !valid {
			// Error already emitted in ValidateShopPhase
			return
		}

		// Get the game lobby from Redis
		lobbyState, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			log.Printf("[SHOP-ERROR] Error getting lobby state: %v", err)
			client.Emit("error", gin.H{"error": "Error getting lobby state"})
			return
		}

		if lobbyState.ShopState == nil {
			client.Emit("error", gin.H{"error": "Lobby shop state not found"})
			return
		}

		item, exists := shop.FindShopItem(*lobbyState, itemID)
		if !exists || item.Type != game_constants.PACK_TYPE {
			client.Emit("invalid_pack")
			return
		}

		// Check if this is a joker pack and player already has max jokers
		if item.PackType == game_constants.PACK_TYPE_JOKERS {
			// Parse current jokers
			var currentJokers poker.Jokers
			if playerState.CurrentJokers != nil && len(playerState.CurrentJokers) > 0 {
				if err := json.Unmarshal(playerState.CurrentJokers, &currentJokers); err != nil {
					log.Printf("[SHOP-ERROR] Error parsing player's jokers: %v", err)
					client.Emit("error", gin.H{"error": "Error processing jokers"})
					return
				}
			} else {
				currentJokers = poker.Jokers{
					Juglares: []int{},
				}
			}

			// Check if player already has max jokers
			if len(currentJokers.Juglares) >= game_constants.MaxJokersPerPlayer {
				client.Emit("purchase_failed", gin.H{
					"error": fmt.Sprintf("You cannot have more than %d jokers", game_constants.MaxJokersPerPlayer),
				})
				return
			}
		}

		// Validate the purchase
		if err := shop.ValidatePurchase(item, game_constants.PACK_TYPE, clientPrice, playerState); err != nil {
			log.Printf("[SHOP-ERROR] Purchase validation failed: %v", err)
			client.Emit("purchase_failed", gin.H{"error": err.Error()})
			return
		}

		// Get pack contents and process jokers to include sell prices
		contents, err := shop.GetOrGeneratePackContents(redisClient, lobbyState, item)
		if err != nil {
			client.Emit("pack_generation_failed")
			return
		}

		// Process jokers to include sell prices if this is a joker pack or any pack containing jokers
		jokersWithPrices := make([]gin.H, len(contents.Jokers))
		for i, jokerGroup := range contents.Jokers {
			// For each joker in the group, add its sell price
			jokersWithIDs := make([]gin.H, len(jokerGroup.Juglares))
			for j, jokerID := range jokerGroup.Juglares {
				jokersWithIDs[j] = gin.H{
					"id":         jokerID,
					"sell_price": poker.CalculateJokerSellPrice(jokerID),
				}
			}

			// Create a new structure that mirrors the original but with prices
			jokersWithPrices[i] = gin.H{
				"Juglares": jokersWithIDs,
			}
		}

		// Update player's LastPurchasedPackItemId and deduct money
		// NOTE: potential exploit by not sending a pack selection event and
		// then reusing this same id during the next round. Already fixed by resetting
		// LastPurchasedPackItemId to -1 when starting the shop phase
		playerState.LastPurchasedPackItemId = itemID
		playerState.PlayersMoney -= item.Price // Deduct the money

		// Save the updated player state
		if err := redisClient.SaveInGamePlayer(playerState); err != nil {
			log.Printf("[SHOP-ERROR] Error saving player state: %v", err)
			client.Emit("error", gin.H{"error": "Failed to save purchase"})
			return
		}

		// Emit the pack_purchased event with joker sell prices included
		client.Emit("pack_purchased", gin.H{
			"item_id":         item.ID,
			"cards":           contents.Cards,
			"jokers":          jokersWithPrices, // Use the processed jokers with sell prices
			"vouchers":        contents.Vouchers,
			"max_selectable":  item.MaxSelectable,
			"pack_type":       item.PackType,
			"remaining_money": playerState.PlayersMoney,
		})
	}
}

// TODO: set playerState.LastPurchasedPackItemId to -1 upon ending the pack selection event

func HandleBuyJoker(redisClient *redis_services.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("BuyJoker initiated - User: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())

		if len(args) < 2 {
			log.Printf("[SHOP-ERROR] Missing arguments for user %s", username)
			client.Emit("error", gin.H{"error": "Missing joker ID or price"})
			return
		}

		// Parse joker ID (JavaScript numbers come as float64)
		itemIDFloat, ok := args[0].(float64)
		if !ok {
			client.Emit("error", gin.H{"error": "Item ID must be a number"})
			return
		}
		itemID := int(itemIDFloat)

		// Get the client-provided price
		priceFloat, ok := args[1].(float64)
		if !ok {
			client.Emit("error", gin.H{"error": "Price must be a number"})
			return
		}
		clientPrice := int(priceFloat)

		// Get player state first to extract lobby ID
		playerState, err := redisClient.GetInGamePlayer(username)
		if err != nil {
			log.Printf("[SHOP-ERROR] Error getting player state: %v", err)
			client.Emit("error", gin.H{"error": "Error retrieving player state"})
			return
		}

		// Extract lobby ID from player state
		lobbyID := playerState.LobbyId
		if lobbyID == "" {
			log.Printf("[SHOP-ERROR] Player %s not associated with any lobby", username)
			client.Emit("error", gin.H{"error": "Player not in a lobby"})
			return
		}

		log.Printf("[INFO] Processing joker purchase for user: %s in lobby: %s, joker ID: %d, price: %d",
			username, lobbyID, itemID, clientPrice)

		// Validate that we are in the shop phase
		valid, err := socketio_utils.ValidateShopPhase(redisClient, client, lobbyID)
		if err != nil || !valid {
			// Error already emitted in ValidateShopPhase
			return
		}

		// Get the lobby state from Redis
		lobbyState, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			log.Printf("[SHOP-ERROR] Error getting lobby state: %v", err)
			client.Emit("error", gin.H{"error": "Error getting lobby state"})
			return
		}

		if lobbyState.ShopState == nil {
			client.Emit("error", gin.H{"error": "Lobby shop state not found"})
			return
		}

		// Find the joker in the shop
		item, exists := shop.FindShopItem(*lobbyState, itemID)
		if !exists {
			client.Emit("invalid_item_id", gin.H{"error": "Shop item not found"})
			return
		}

		// Process the joker purchase with price validation
		success, updatedPlayer, err := shop.PurchaseJoker(redisClient, playerState, item, clientPrice)
		if err != nil || !success {
			log.Printf("[SHOP-ERROR] Purchase failed: %v", err)
			client.Emit("purchase_failed", gin.H{"error": err.Error()})
			return
		}

		// Save the updated player state
		if err := redisClient.SaveInGamePlayer(updatedPlayer); err != nil {
			log.Printf("[SHOP-ERROR] Error saving player state: %v", err)
			client.Emit("error", gin.H{"error": "Failed to save purchase"})
			return
		}

		// Notify client of successful purchase
		client.Emit("joker_purchased", gin.H{
			"item_id":  item.ID,
			"joker_id": item.JokerId,
			// NOTE: the sell price is calculated based on the joker ID, not the corresponding shop item ID
			"sell_price":      poker.CalculateJokerSellPrice(item.JokerId),
			"remaining_money": updatedPlayer.PlayersMoney,
		})
	}
}

func HandleBuyVoucher(redisClient *redis_services.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("BuyVoucher initiated - User: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())

		if len(args) < 2 {
			log.Printf("[SHOP-ERROR] Missing arguments for user %s", username)
			client.Emit("error", gin.H{"error": "Missing voucher ID or price"})
			return
		}

		// Parse voucher ID (JavaScript numbers come as float64)
		itemIDFloat, ok := args[0].(float64)
		if !ok {
			client.Emit("error", gin.H{"error": "Voucher ID must be a number"})
			return
		}
		itemID := int(itemIDFloat)

		// Get the client-provided price
		priceFloat, ok := args[1].(float64)
		if !ok {
			client.Emit("error", gin.H{"error": "Price must be a number"})
			return
		}
		clientPrice := int(priceFloat)

		// Get player state first to extract lobby ID
		playerState, err := redisClient.GetInGamePlayer(username)
		if err != nil {
			log.Printf("[SHOP-ERROR] Error getting player state: %v", err)
			client.Emit("error", gin.H{"error": "Error retrieving player state"})
			return
		}

		// Extract lobby ID from player state
		lobbyID := playerState.LobbyId
		if lobbyID == "" {
			log.Printf("[SHOP-ERROR] Player %s not associated with any lobby", username)
			client.Emit("error", gin.H{"error": "Player not in a lobby"})
			return
		}

		log.Printf("[INFO] Processing voucher purchase for user: %s in lobby: %s, voucher ID: %d, price: %d",
			username, lobbyID, itemID, clientPrice)

		// Validate that we are in the shop phase
		valid, err := socketio_utils.ValidateShopPhase(redisClient, client, lobbyID)
		if err != nil || !valid {
			// Error already emitted in ValidateShopPhase
			return
		}

		// Get the lobby state from Redis
		lobbyState, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			log.Printf("[SHOP-ERROR] Error getting lobby state: %v", err)
			client.Emit("error", gin.H{"error": "Error getting lobby state"})
			return
		}

		if lobbyState.ShopState == nil {
			client.Emit("error", gin.H{"error": "Lobby shop state not found"})
			return
		}

		// Find the voucher in the shop
		item, exists := shop.FindShopItem(*lobbyState, itemID)
		if !exists {
			client.Emit("invalid_item_id", gin.H{"error": "Shop item not found"})
			return
		}

		// Process the voucher purchase with price validation
		success, updatedPlayer, err := shop.PurchaseVoucher(redisClient, playerState, item, clientPrice)
		if err != nil || !success {
			log.Printf("[SHOP-ERROR] Purchase failed: %v", err)
			client.Emit("purchase_failed", gin.H{"error": err.Error()})
			return
		}

		// Save the updated player state
		if err := redisClient.SaveInGamePlayer(updatedPlayer); err != nil {
			log.Printf("[SHOP-ERROR] Error saving player state: %v", err)
			client.Emit("error", gin.H{"error": "Failed to save purchase"})
			return
		}

		// Notify client of successful purchase
		client.Emit("voucher_purchased", gin.H{
			"item_id":         item.ID,
			"voucher_id":      item.ModifierId,
			"remaining_money": updatedPlayer.PlayersMoney,
		})
	}
}

func HandleSellJoker(redisClient *redis_services.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("SellJoker initiated - User: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())

		if len(args) < 1 {
			log.Printf("[SHOP-ERROR] Missing joker ID for user %s", username)
			client.Emit("error", gin.H{"error": "Missing joker ID to sell"})
			return
		}

		// Parse joker ID (JavaScript numbers come as float64)
		jokerIDFloat, ok := args[0].(float64)
		if !ok {
			client.Emit("error", gin.H{"error": "Joker ID must be a number"})
			return
		}
		jokerID := int(jokerIDFloat)

		// Get player state
		playerState, err := redisClient.GetInGamePlayer(username)
		if err != nil {
			log.Printf("[SHOP-ERROR] Error getting player state: %v", err)
			client.Emit("error", gin.H{"error": "Error retrieving player state"})
			return
		}

		// Extract lobby ID from player state
		lobbyID := playerState.LobbyId
		if lobbyID == "" {
			log.Printf("[SHOP-ERROR] Player %s not associated with any lobby", username)
			client.Emit("error", gin.H{"error": "Player not in a lobby"})
			return
		}

		// Validate that we are in the shop phase
		valid, err := socketio_utils.ValidateShopPhase(redisClient, client, lobbyID)
		if err != nil || !valid {
			// Error already emitted in ValidateShopPhase
			return
		}

		// Process the joker sale
		updatedPlayer, sellPrice, err := shop.SellJoker(playerState, jokerID)
		if err != nil {
			log.Printf("[SHOP-ERROR] Sale failed: %v", err)
			client.Emit("error", gin.H{"error": err.Error()})
			return
		}

		// Save the updated player state
		if err := redisClient.SaveInGamePlayer(updatedPlayer); err != nil {
			log.Printf("[SHOP-ERROR] Error saving player state: %v", err)
			client.Emit("error", gin.H{"error": "Failed to save joker sale"})
			return
		}

		// Notify client of successful sale
		client.Emit("joker_sold", gin.H{
			"joker_id":        jokerID,
			"sell_price":      sellPrice,
			"remaining_money": updatedPlayer.PlayersMoney,
		})
	}
}

func HandlePackSelection(redisClient *redis_services.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("PackSelection initiated - User: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())

		// Check if we have all required arguments
		if len(args) < 2 {
			log.Printf("[SHOP-ERROR] Missing arguments for user %s", username)
			client.Emit("error", gin.H{"error": "Missing pack ID or selections"})
			return
		}

		// Parse shop item ID
		itemIDFloat, ok := args[0].(float64)
		if !ok {
			client.Emit("error", gin.H{"error": "Shop item ID must be a number"})
			return
		}
		itemID := int(itemIDFloat)

		// Parse selected items map
		selectionsMap, ok := args[1].(map[string]interface{})
		if !ok {
			client.Emit("error", gin.H{"error": "Selections must be an object"})
			return
		}

		// Get player state
		playerState, err := redisClient.GetInGamePlayer(username)
		if err != nil {
			log.Printf("[SHOP-ERROR] Error getting player state: %v", err)
			client.Emit("error", gin.H{"error": "Error retrieving player state"})
			return
		}

		// Extract lobby ID from player state
		lobbyID := playerState.LobbyId
		if lobbyID == "" {
			log.Printf("[SHOP-ERROR] Player %s not associated with any lobby", username)
			client.Emit("error", gin.H{"error": "Player not in a lobby"})
			return
		}

		// Validate we are in shop phase
		valid, err := socketio_utils.ValidateShopPhase(redisClient, client, lobbyID)
		if err != nil || !valid {
			// Error already emitted in ValidateShopPhase
			return
		}

		// Verify that the player actually bought this pack
		if playerState.LastPurchasedPackItemId != itemID {
			client.Emit("error", gin.H{"error": "You have not purchased this pack or already selected items from it"})
			return
		}

		// Get the lobby state
		lobbyState, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			log.Printf("[SHOP-ERROR] Error getting lobby state: %v", err)
			client.Emit("error", gin.H{"error": "Error getting lobby state"})
			return
		}

		// Process the selection
		updatedPlayer, err := shop.ProcessPackSelection(redisClient, lobbyState, playerState, itemID, selectionsMap)
		if err != nil {
			log.Printf("[SHOP-ERROR] Pack selection failed: %v", err)
			client.Emit("error", gin.H{"error": err.Error()})
			return
		}

		// Save the updated player state
		if err := redisClient.SaveInGamePlayer(updatedPlayer); err != nil {
			log.Printf("[SHOP-ERROR] Error saving player state: %v", err)
			client.Emit("error", gin.H{"error": "Failed to save pack selection"})
			return
		}

		// Notify client of successful selection
		client.Emit("pack_selection_complete", gin.H{
			"message":         "Successfully added selected items to your inventory",
			"selections":      selectionsMap,
			"remaining_money": updatedPlayer.PlayersMoney,
		})
	}
}

func HandleRerollShop(redisClient *redis_services.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("RerollShop initiated - User: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())
		// Get player state
		playerState, err := redisClient.GetInGamePlayer(username)
		if err != nil {
			log.Printf("[SHOP-ERROR] Error getting player state: %v", err)
			client.Emit("error", gin.H{"error": "Error retrieving player state"})
			return
		}
		// Extract lobby ID from player state
		lobbyID := playerState.LobbyId
		if lobbyID == "" {
			log.Printf("[SHOP-ERROR] Player %s not associated with any lobby", username)
			client.Emit("error", gin.H{"error": "Player not in a lobby"})
			return
		}
		// Validate we are in shop phase
		valid, err := socketio_utils.ValidateShopPhase(redisClient, client, lobbyID)
		if err != nil || !valid {
			// Error already emitted in ValidateShopPhase
			return
		}
		// Get the lobby state
		lobby, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			log.Printf("[SHOP-ERROR] Error getting lobby state: %v", err)
			client.Emit("error", gin.H{"error": "Error getting lobby state"})
			return
		}
		// Check if the player has enough money to reroll
		if playerState.PlayersMoney < lobby.ShopState.Rerolls+2 {
			client.Emit("error", gin.H{"error": "Not enough money to reroll"})
			return
		}

		playerState.PlayersMoney -= lobby.ShopState.Rerolls + 2

		// Check if it is the highest reroll
		if playerState.Rerolls == lobby.ShopState.Rerolls {
			// Hay que generar el nuevo reroll
			lobby.ShopState.Rerolls++
			playerState.Rerolls++
			rng := rand.New(rand.NewSource(uint64(lobby.ShopState.RerollSeed) + uint64(lobby.CurrentRound) + uint64(lobby.ShopState.Rerolls)))
			rerolledJokers := shop.GenerateRerollableItems(rng, &lobby.ShopState.NextUniqueId)

			lobby.ShopState.Rerolled = append(lobby.ShopState.Rerolled, rerolledJokers)
			// Notify client of successful selection
			client.Emit("rerolled_jokers", gin.H{
				"message":    "Successfully rerolled jokers",
				"new_jokers": rerolledJokers,
			})

			// Save the updated player state
			if err := redisClient.SaveInGamePlayer(playerState); err != nil {
				log.Printf("[SHOP-ERROR] Error saving player state: %v", err)
				client.Emit("error", gin.H{"error": "Failed to save pack selection"})
				return
			}

			if err := redisClient.SaveGameLobby(lobby); err != nil {
				log.Printf("[SHOP-ERROR] Error saving lobby state: %v", err)
				client.Emit("error", gin.H{"error": "Failed to save lobby state"})
				return
			}

		} else {
			playerState.Rerolls++
			newJokers := lobby.ShopState.Rerolled[playerState.Rerolls]

			// Save the updated player state
			if err := redisClient.SaveInGamePlayer(playerState); err != nil {
				log.Printf("[SHOP-ERROR] Error saving player state: %v", err)
				client.Emit("error", gin.H{"error": "Failed to save pack selection"})
				return
			}

			client.Emit("rerolled_jokers", gin.H{
				"message":    "Successfully rerolled jokers",
				"new_jokers": newJokers,
			})
		}
	}
}
