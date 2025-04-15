package game_flow

import (
	game_constants "Nogler/constants/game"
	redis_models "Nogler/models/redis"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	socketio_utils "Nogler/services/socket_io/utils"
	"Nogler/services/socket_io/utils/stages/blind"
	"Nogler/services/socket_io/utils/stages/end_game"
	"Nogler/services/socket_io/utils/stages/play_round"
	"Nogler/services/socket_io/utils/stages/shop"
	"Nogler/utils"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// ---------------------------------------------------------------
// Functions that are executed to start the next blind
// ---------------------------------------------------------------

func AdvanceToNextBlindIfUndone(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer, isFirstBlind bool, expectedRound int) error {
	log.Printf("[ROUND-ADVANCE] Advancing to next round for lobby %s (expected round: %d)", lobbyID, expectedRound)

	// Get the lobby for early check
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-ADVANCE-ERROR] Error getting lobby: %v", err)
		return fmt.Errorf("error getting lobby: %v", err)
	}

	// Validate the round number - abort if this is an old timeout trying to advance a newer round
	if !isFirstBlind && lobby.CurrentRound != expectedRound {
		log.Printf("[ROUND-ADVANCE-WARN] Round mismatch - current: %d, expected: %d. Ignoring stale timeout.",
			lobby.CurrentRound, expectedRound)
		return nil
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

	// Calculate new base blind for the round: BASE_BLIND * 2^(round_number - 1)
	newBaseBlind := game_constants.BASE_BLIND
	for i := 1; i < newRound; i++ {
		newBaseBlind *= 2
		if newBaseBlind >= game_constants.MAX_BLIND {
			newBaseBlind = game_constants.MAX_BLIND
			break
		}
	}

	// Update the CurrentBaseBlind in the lobby
	lobby.CurrentBaseBlind = newBaseBlind
	if err := redisClient.SaveGameLobby(lobby); err != nil {
		log.Printf("[ROUND-ADVANCE-ERROR] Failed to update base blind: %v", err)
		return fmt.Errorf("failed to update base blind: %v", err)
	}

	log.Printf("[ROUND-ADVANCE] Updated base blind for lobby %s to %d for round %d",
		lobbyID, newBaseBlind, newRound)

	// Set all players' BetMinimumBlind to true as default
	players, err := redisClient.GetAllPlayersInLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-ADVANCE-ERROR] Error getting players: %v", err)
		return fmt.Errorf("error getting players: %v", err)
	}

	for i := range players {
		players[i].BetMinimumBlind = true
		if err := redisClient.SaveInGamePlayer(&players[i]); err != nil {
			log.Printf("[ROUND-ADVANCE-ERROR] Error setting default blind bet for player %s: %v",
				players[i].Username, err)
			// Continue with other players despite error
		}
	}

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

	// Reset the blind-related map
	lobby.ProposedBlinds = make(map[string]bool)

	// Set the blind timeout to the current time
	lobby.BlindTimeout = time.Now()
	err = redisClient.SaveGameLobby(lobby)
	if err != nil {
		log.Printf("[BLIND-TIMEOUT-ERROR] Error setting lobby blind timeout: %v", err)
		return
	}

	// Start a goroutine to handle the timeout
	go func() {
		// Capture the current round when creating the goroutine
		currentRound := lobby.CurrentRound

		// TODO, change the timeout value
		time.Sleep(20 * time.Second)

		// Pass the expected round to AdvanceToNextRoundPlayIfUndone
		AdvanceToNextRoundPlayIfUndone(redisClient, db, lobbyID, sio, currentRound)
	}()

	log.Printf("[BLIND-TIMEOUT] Blind timeout started for lobby %s", lobbyID)
}

// ---------------------------------------------------------------
// Functions that are executed to start the next game round
// ---------------------------------------------------------------

// Update AdvanceToNextRoundPlayIfUndone
func AdvanceToNextRoundPlayIfUndone(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer, expectedRound int) {
	log.Printf("[ROUND-PLAY-ADVANCE] Advancing to round play phase for lobby %s (expected round: %d)",
		lobbyID, expectedRound)

	// Get the lobby to check if round already started
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-PLAY-ADVANCE-ERROR] Error getting lobby info: %v", err)
		return
	}

	// Validate the round number
	if lobby.CurrentRound != expectedRound {
		log.Printf("[ROUND-PLAY-ADVANCE-WARN] Round mismatch - current: %d, expected: %d. Ignoring stale timeout.",
			lobby.CurrentRound, expectedRound)
		return
	} else if lobby.BlindTimeout.IsZero() {
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

	// NOTE: Already done in PrepareRoundStart
	// lobby.PlayersFinishedRound = make(map[string]bool)

	// Set the game round timeout to the current time
	lobby.GameRoundTimeout = time.Now()
	err = redisClient.SaveGameLobby(lobby)
	if err != nil {
		log.Printf("[ROUND-PLAY-ERROR] Error setting lobby round timeout: %v", err)
		return
	}

	// Start a goroutine to handle the timeout
	go func() {
		// Capture the current round
		currentRound := lobby.CurrentRound

		// TODO, change the timeout value
		time.Sleep(2 * time.Minute)

		// Call the function to handle round end
		HandleRoundPlayEnd(redisClient, db, lobbyID, sio, currentRound)
	}()

	log.Printf("[ROUND-PLAY-TIMEOUT] Round play timeout started for lobby %s", lobbyID)
}

// ---------------------------------------------------------------
// Functions that are executed when the current game round
// finishes and to start the next shop phase / finish game
// ---------------------------------------------------------------

// Now modify the HandleRoundEnd function
// Update HandleRoundPlayEnd
func HandleRoundPlayEnd(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer, expectedRound int) {
	log.Printf("[ROUND-END] Handling end of round for lobby %s (expected round: %d)",
		lobbyID, expectedRound)

	// Get the lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-END-ERROR] Error getting lobby: %v", err)
		return
	}

	// Validate the round number
	if lobby.CurrentRound != expectedRound {
		log.Printf("[ROUND-END-WARN] Round mismatch - current: %d, expected: %d. Ignoring stale timeout.",
			lobby.CurrentRound, expectedRound)
		return
	} else if lobby.GameRoundTimeout.IsZero() {
		log.Printf("[ROUND-END-INFO] Round already ended for lobby %s, skipping", lobbyID)
		return
	}

	// Reset the game round timeout to indicate round has ended
	lobby.GameRoundTimeout = time.Time{}

	// CRITICAL: save game lobby to indicate round has ended
	err = redisClient.SaveGameLobby(lobby)
	if err != nil {
		log.Printf("[ROUND-END-ERROR] Error saving lobby with updated GameRoundTimeout: %v", err)
		return
	}

	// Process eliminations based on blind achievement
	_, err = play_round.HandlePlayerEliminations(redisClient, lobbyID, sio)
	if err != nil {
		log.Printf("[ELIMINATION-ERROR] Error handling player eliminations: %v", err)
	}

	// Get updated lobby (player count might have changed after eliminations)
	lobby, err = redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-END-ERROR] Error getting updated lobby: %v", err)
		return
	}

	// Check if the game should end (player count <= 1 or max rounds reached)
	if lobby.PlayerCount <= 1 || lobby.CurrentRound >= game_constants.MaxGameRounds {
		log.Printf("[ROUND-END] Game ending conditions met: players=%d, current_round=%d",
			lobby.PlayerCount, lobby.CurrentRound)

		// Go to game end phase
		end_game.AnnounceWinner(redisClient, db, lobbyID, sio)

		// Short delay to ensure all clients receive the game_end event before cleanup
		time.Sleep(2 * time.Second)

		// Clean up all game resources (ALL the players and the lobby from both redis and Postgres)
		// TODO: might have to remove it
		end_game.CleanupGame(redisClient, db, lobbyID)
	} else {
		// Continue with shop phase
		AdvanceToShop(redisClient, db, lobbyID, sio)
	}

	log.Printf("[ROUND-END] Round ended for lobby %s", lobbyID)
}

// Create the AdvanceToShop function that handles shop initialization
func AdvanceToShop(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {
	log.Printf("[SHOP-ADVANCE] Advancing to shop phase for lobby %s", lobbyID)

	// Get updated lobby
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[SHOP-ADVANCE-ERROR] Error getting updated lobby: %v", err)
		return
	}

	// Initialize the shop
	shopItems, err := shop.InitializeShop(lobbyID, lobby.CurrentRound)
	if err != nil {
		log.Printf("[SHOP-INIT-ERROR] Error initializing shop: %v", err)
		return
	}

	// Update the current phase (set it to redis_models.PhaseShop)
	if err := socketio_utils.SetGamePhase(redisClient, lobbyID, redis_models.PhaseShop); err != nil {
		log.Printf("[SHOP-ADVANCE-ERROR] Error setting shop phase: %v", err)
		return
	}

	// Get the fresh lobby after phase update
	lobby, err = redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[SHOP-ADVANCE-ERROR] Error getting updated lobby: %v", err)
		return
	}

	// Store shop state in lobby
	lobby.ShopState = shopItems

	// Reset shop-related counters (NEW, using map)
	lobby.PlayersFinishedShop = make(map[string]bool)

	// Save the updated lobby
	if err := redisClient.SaveGameLobby(lobby); err != nil {
		log.Printf("[SHOP-ADVANCE-ERROR] Error saving lobby: %v", err)
		return
	}

	// Broadcast shop start to all players
	shop.BroadcastStartingShop(sio, lobbyID, shopItems)

	// Start the shop timeout
	StartShopTimeout(redisClient, db, lobbyID, sio)

	log.Printf("[SHOP-ADVANCE] Successfully advanced lobby %s to shop phase", lobbyID)
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
		// Capture the expected round for the next blind (current round)
		currentRound := lobby.CurrentRound

		// TODO, change the timeout value
		time.Sleep(1 * time.Minute)

		// Advance to the next blind
		AdvanceToNextBlindIfUndone(redisClient, db, lobbyID, sio, false, currentRound)
	}()

	log.Printf("[SHOP-TIMEOUT] Shop timeout started for lobby %s", lobbyID)
}
