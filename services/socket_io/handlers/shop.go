package handlers

import (
	"Nogler/models/postgres"

	redis_services "Nogler/services/redis"
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

		contents, err := shop.GetOrGeneratePackContents(redisClient, lobbyState, item, username)
		if err != nil {
			client.Emit("pack_generation_failed")
			return
		}

		// What does the pack contain (type) + id of that item in that type
		client.Emit("pack_opened", gin.H{
			"cards":  contents.Cards,
			"jokers": contents.Jokers,
		})
	}
}

// TODO: add more functionalities, buy vouchers, jokers, etc?
