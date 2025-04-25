package handlers

import (
	game_constants "Nogler/constants/game"
	"Nogler/models/postgres"

	"Nogler/services/poker"
	redis_services "Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	socketio_utils "Nogler/services/socket_io/utils"
	"Nogler/services/socket_io/utils/stages/shop"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

// Handler that will be called.
func HandleOpenPack(redisClient *redis_services.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {

		log.Printf("OpenPack iniciado - Usuario: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())

		if len(args) < 2 {
			log.Printf("[HAND-ERROR] Faltan argumentos para usuario %s", username)
			client.Emit("error", gin.H{"error": "Falta el pack a abrir o la lobby ID"})
			return
		}

		// Handle pack ID (JavaScript numbers come as float64 says depsik)
		packIDFloat, ok := args[0].(float64)
		if !ok {
			client.Emit("error", gin.H{"error": "El pack ID debe ser un número"})
			return
		}

		packID := int(packIDFloat) // Convert to int
		lobbyID := args[1].(string)

		log.Printf("[INFO] Obteniendo información del lobby ID: %s para usuario: %s", lobbyID, username)

		// Check if lobby exists
		var lobby postgres.GameLobby
		if err := db.Where("id = ?", lobbyID).First(&lobby).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				client.Emit("error", gin.H{"error": "Lobby not found"})
			} else {
				client.Emit("error", gin.H{"error": "Database error"})
			}
			return
		}

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

		item, exists := shop.FindShopItem(*lobbyState, packID)
		if !exists || item.Type != game_constants.PACK_TYPE {
			client.Emit("invalid_pack")
			return
		}

		contents, err := shop.GetOrGeneratePackContents(redisClient, lobbyState, item)
		if err != nil {
			client.Emit("pack_generation_failed")
			return
		}

		client.Emit("pack_opened", gin.H{
			"cards":  contents.Cards,
			"jokers": contents.Jokers,
		})
	}
}

// TODO, document
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
		jokerIDFloat, ok := args[0].(float64)
		if !ok {
			client.Emit("error", gin.H{"error": "Joker ID must be a number"})
			return
		}
		jokerID := int(jokerIDFloat)

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
			username, lobbyID, jokerID, clientPrice)

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
		item, exists := shop.FindShopItem(*lobbyState, jokerID)
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
			"item_id":         item.ID,
			"joker_id":        item.JokerId,
			"sell_price":      poker.CalculateJokerSellPrice(jokerID),
			"remaining_money": updatedPlayer.PlayersMoney,
		})

		// NOTE: innecessary
		// Broadcast purchase to all players in lobby
		/*sio.Sio_server.To(socket.Room(lobbyID)).Emit("player_bought_joker", gin.H{
			"username": username,
			"joker_id": jokerID,
		})*/
	}
}

// TODO, document
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
		voucherIDFloat, ok := args[0].(float64)
		if !ok {
			client.Emit("error", gin.H{"error": "Voucher ID must be a number"})
			return
		}
		voucherID := int(voucherIDFloat)

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
			username, lobbyID, voucherID, clientPrice)

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
		item, exists := shop.FindShopItem(*lobbyState, voucherID)
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
