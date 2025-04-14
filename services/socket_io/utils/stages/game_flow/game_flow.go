package game_flow

import (
	redis_models "Nogler/models/redis"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	socketio_utils "Nogler/services/socket_io/utils"
	"Nogler/services/socket_io/utils/stages/blind"
	"Nogler/services/socket_io/utils/stages/play_round"
	"Nogler/services/socket_io/utils/stages/shop"
	"Nogler/utils"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------
// Functions that are executed to start the next blind
// ---------------------------------------------------------------

func AdvanceToNextBlind(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer, isFirstBlind bool) error {
	log.Printf("[ROUND-ADVANCE] Advancing to next round for lobby %s", lobbyID)

	// Get the lobby for early check
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-ADVANCE-ERROR] Error getting lobby: %v", err)
		return fmt.Errorf("error getting lobby: %v", err)
	}

	// Early return if already advancing to next round (shop timeout is zero and not first blind)
	if lobby.ShopTimeout.IsZero() && !isFirstBlind {
		log.Printf("[ROUND-ADVANCE-INFO] Already advancing to next round for lobby %s, skipping", lobbyID)
		return nil
	}

	// Step 1: Increment the round number
	newRound, err := socketio_utils.IncrementGameRound(redisClient, lobbyID, 1)
	if err != nil {
		log.Printf("[ROUND-ADVANCE-ERROR] Failed to increment round: %v", err)
		return fmt.Errorf("failed to increment round: %v", err)
	}

	log.Printf("[ROUND-ADVANCE] Lobby %s advanced to round %d", lobbyID, newRound)

	// Update the current phase (to PhaseBlind)
	if err := socketio_utils.SetGamePhase(redisClient, lobbyID, redis_models.PhaseBlind); err != nil {
		log.Printf("[ROUND-ADVANCE-ERROR] %v", err)
		return err
	}

	// Step 2: Broadcast the next blind phase event
	blind.BroadcastStartingNextBlind(redisClient, db, lobbyID, sio)

	// Step 3: Start the blind timeout process
	StartBlindTimeout(redisClient, db, lobbyID, sio, isFirstBlind)

	return nil
}

func StartBlindTimeout(redisClient *redis.RedisClient,
	db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer, isFirstBlind bool) {

	log.Printf("[BLIND-TIMEOUT] Starting blind timeout for lobby %s", lobbyID)

	// Get the lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[BLIND-TIMEOUT-ERROR] Error obtaining lobby to start timeout: %v", err)
		return
	}

	// Check if the blind voting is already in timeout
	if !lobby.BlindTimeout.IsZero() {
		log.Printf("[BLIND-TIMEOUT-ERROR] Blind voting is already in timeout: %v", lobby.BlindTimeout)
		return
	}

	// Reset the shop timeout to indicate shop phase has ended
	lobby.ShopTimeout = time.Time{}

	// Check if lobby exists in PostgreSQL
	_, err = utils.CheckLobbyExists(db, lobbyID)
	if err != nil {
		log.Printf("[BLIND-TIMEOUT-ERROR] Lobby does not exist: %s", lobbyID)
		return
	}

	// Reset the blind-related counters
	lobby.TotalProposedBlinds = 0

	// Set the blind timeout to the current time
	lobby.BlindTimeout = time.Now()
	err = redisClient.SaveGameLobby(lobby)
	if err != nil {
		log.Printf("[BLIND-TIMEOUT-ERROR] Error setting lobby blind timeout: %v", err)
		return
	}

	// Start a goroutine to handle the timeout
	go func() {
		// TODO, change the timeout
		time.Sleep(10 * time.Second)

		// Check if the blind phase is still active
		currentLobby, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			log.Printf("[BLIND-TIMEOUT-ERROR] Error getting lobby after timeout: %v", err)
			return
		}

		// NOTE: If the blind timeout was reset, it means the round already started
		// because ALL the players proposed their blinds
		if currentLobby.BlindTimeout.IsZero() {
			log.Printf("[BLIND-TIMEOUT-INFO] Blind phase already completed for lobby %s", lobbyID)
			return
		}

		// When the timeout expires, send the round start event
		AdvanceToNextRoundPlay(redisClient, db, lobbyID, sio)
	}()

	log.Printf("[BLIND-TIMEOUT] Blind timeout started for lobby %s", lobbyID)
}

// ---------------------------------------------------------------
// Functions that are executed to start the next game round
// ---------------------------------------------------------------

func AdvanceToNextRoundPlay(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {
	log.Printf("[ROUND-PLAY-ADVANCE] Advancing to round play phase for lobby %s", lobbyID)

	// Get the lobby to check if round already started
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-PLAY-ADVANCE-ERROR] Error getting lobby info: %v", err)
		return
	}

	// Early return if blind timeout is already reset (round already started)
	if lobby.BlindTimeout.IsZero() {
		log.Printf("[ROUND-PLAY-ADVANCE-INFO] Round already started for lobby %s, skipping", lobbyID)
		return
	}

	// Step 1: Prepare the round state in Redis
	updatedLobby, blind, err := play_round.PrepareRoundStart(redisClient, lobbyID)
	if err != nil {
		log.Printf("[ROUND-PLAY-ADVANCE-ERROR] Failed to prepare round: %v", err)
		return
	}

	// Step 2: Broadcast round start event
	play_round.BroadcastRoundStart(sio, lobbyID, updatedLobby.CurrentRound, blind)

	// Step 3: Start the round play timeout
	StartRoundPlayTimeout(redisClient, db, lobbyID, sio)

	log.Printf("[ROUND-PLAY-ADVANCE-SUCCESS] Advanced lobby %s to round play phase", lobbyID)
}

func StartRoundPlayTimeout(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {
	log.Printf("[ROUND-PLAY-TIMEOUT] Starting round play timeout for lobby %s", lobbyID)

	// Get the game lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-PLAY-ERROR] Error getting lobby info: %v", err)
		return
	}

	// Check if the round is already in timeout
	// NOTE: SHOULDN'T HAPPEN
	if !lobby.GameRoundTimeout.IsZero() {
		log.Printf("[ROUND-PLAY-ERROR] Round is already in timeout: %v", lobby.GameRoundTimeout)
		return
	}

	// Reset the round-related counters
	lobby.TotalPlayersFinishedRound = 0

	// Set the game round timeout to the current time
	lobby.GameRoundTimeout = time.Now()
	err = redisClient.SaveGameLobby(lobby)
	if err != nil {
		log.Printf("[ROUND-PLAY-ERROR] Error setting lobby round timeout: %v", err)
		return
	}

	// Start a goroutine to handle the timeout
	go func() {
		// Sleep for the round duration (e.g., 2 minutes)
		time.Sleep(2 * time.Minute)

		// Call the function to handle round end
		HandleRoundEnd(redisClient, db, lobbyID, sio)
	}()

	log.Printf("[ROUND-PLAY-TIMEOUT] Round play timeout started for lobby %s", lobbyID)
}

// ---------------------------------------------------------------
// Functions that are executed when the current game round
// finishes and to start the next shop phase
// ---------------------------------------------------------------

// Function to handle the end of a round
func HandleRoundEnd(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {
	log.Printf("[ROUND-END] Handling end of round for lobby %s", lobbyID)

	// Get the lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-END-ERROR] Error getting lobby: %v", err)
		return
	}

	// If GameRoundTimeout is zero, it means the round has already ended
	if lobby.GameRoundTimeout.IsZero() {
		log.Printf("[ROUND-END-INFO] Round already ended for lobby %s, skipping", lobbyID)
		return
	}

	// Reset the game round timeout to indicate round has ended
	lobby.GameRoundTimeout = time.Time{}

	// Update the current phase
	lobby.CurrentPhase = redis_models.PhaseShop

	// CRITICAL: save game lobby, we'll save it again in handlePlayerEliminations,
	// and before broadcasting `starting_shop` event to the players (avoid concurrency problems)
	err = redisClient.SaveGameLobby(lobby)
	if err != nil {
		log.Printf("[ROUND-END-ERROR] Error saving lobby with updated GameRoundTimeout and CurrentPhase: %v", err)
		return
	}

	// Process eliminations based on blind achievement
	_, err = play_round.HandlePlayerEliminations(redisClient, lobbyID, sio)
	if err != nil {
		log.Printf("[ELIMINATION-ERROR] Error handling player eliminations: %v", err)
	}

	// Get updated lobby (player count might have changed)
	lobby, err = redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-END-ERROR] Error getting updated lobby: %v", err)
		return
	}

	// Initialize the shop phase
	shop, err := shop.InitializeShop(lobbyID, lobby.CurrentRound)
	if err != nil {
		log.Printf("[SHOP-INIT-ERROR] Error initializing shop: %v", err)
	} else {
		// Store shop state in lobby
		lobby.ShopState = shop

		// Reset shop-related counters
		lobby.TotalPlayersFinishedShop = 0

		// Save the updated lobby
		if err := redisClient.SaveGameLobby(lobby); err != nil {
			log.Printf("[ROUND-END-ERROR] Error saving lobby: %v", err)
		}

		// Broadcast shop start to all players
		sio.Sio_server.To(socket.Room(lobbyID)).Emit("starting_shop", gin.H{
			"shop": shop,
		})

		// Start the shop timeout
		StartShopTimeout(redisClient, db, lobbyID, sio)
	}

	log.Printf("[ROUND-END] Round ended for lobby %s", lobbyID)
}

// Function to start the shop timeout
func StartShopTimeout(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {
	log.Printf("[SHOP-TIMEOUT] Starting shop timeout for lobby %s", lobbyID)

	// Get the game lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[SHOP-TIMEOUT-ERROR] Error getting lobby: %v", err)
		return
	}

	// Check if shop timeout is already active
	if !lobby.ShopTimeout.IsZero() {
		log.Printf("[SHOP-TIMEOUT-ERROR] Shop timeout already active for lobby %s", lobbyID)
		return
	}

	// Set the shop timeout
	lobby.ShopTimeout = time.Now()
	err = redisClient.SaveGameLobby(lobby)
	if err != nil {
		log.Printf("[SHOP-TIMEOUT-ERROR] Error saving shop timeout: %v", err)
		return
	}

	// Start the timeout goroutine
	go func() {
		// TODO, change the timeout value
		time.Sleep(1 * time.Minute)

		// Advance to the next blind
		AdvanceToNextBlind(redisClient, db, lobbyID, sio, false)
	}()

	log.Printf("[SHOP-TIMEOUT] Shop timeout started for lobby %s", lobbyID)
}
