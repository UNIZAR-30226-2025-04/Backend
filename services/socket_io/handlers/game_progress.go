package handlers

import (
	redis_models "Nogler/models/redis"
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	"Nogler/utils"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

func HandleProposeBlind(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("[BLIND] %s is proposing a blind", username)

		if len(args) < 2 {
			log.Printf("[BLIND-ERROR] Missing arguments for user %s", username)
			client.Emit("error", gin.H{"error": "Missing proposed blind or lobby"})
			return
		}

		proposedBlind := args[0].(int)
		lobbyID := args[1].(string)

		isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
		if err != nil {
			log.Printf("[BLIND-ERROR] Database error: %v", err)
			client.Emit("error", gin.H{"error": "Database error"})
			return
		}

		if !isInLobby {
			log.Printf("[BLIND-ERROR] User is NOT in lobby: %s, Lobby: %s", username, lobbyID)
			client.Emit("error", gin.H{"error": "You must join the lobby before proposing blinds"})
			return
		}

		// Get the lobby from Redis
		lobby, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			log.Printf("[BLIND-ERROR] Error getting game lobby: %v", err)
			client.Emit("error", gin.H{"error": "Error getting game lobby"})
			return
		}

		// Check if blind timeout has already started
		if lobby.BlindTimeout.IsZero() {
			log.Printf("[BLIND-WARNING] Trying to propose blind without active blind phase for lobby %s", lobbyID)
			client.Emit("error", gin.H{"error": "Blind voting phase is not active"})
			return
		}

		currentBlind, err := redisClient.GetCurrentBlind(lobbyID)
		if err != nil {
			log.Printf("[BLIND-ERROR] Error getting current blind: %v", err)
			client.Emit("error", gin.H{"error": "Error getting current blind"})
			return
		}

		// Update current blind if this proposal is higher
		if proposedBlind > currentBlind {
			err := redisClient.SetCurrentBlind(lobbyID, proposedBlind, username)
			if err != nil {
				log.Printf("[BLIND-ERROR] Could not update current blind: %v", err)
				client.Emit("error", gin.H{"error": "Error updating blind"})
				return
			}

			// Broadcast the new blind value to everyone in the lobby
			sio.Sio_server.To(socket.Room(lobbyID)).Emit("blind_updated", gin.H{
				"new_blind":   proposedBlind,
				"proposed_by": username,
			})
		}

		// Increment the counter of proposed blinds
		lobby.TotalProposedBlinds++
		log.Printf("[BLIND] Player %s proposed blind. Total proposals: %d/%d",
			username, lobby.TotalProposedBlinds, lobby.PlayerCount)

		// Save the updated lobby
		err = redisClient.SaveGameLobby(lobby)
		if err != nil {
			log.Printf("[BLIND-ERROR] Error saving game lobby: %v", err)
			client.Emit("error", gin.H{"error": "Error saving game state"})
			return
		}

		// If all players have proposed, start the round
		if lobby.TotalProposedBlinds >= lobby.PlayerCount {
			log.Printf("[BLIND-COMPLETE] All players have proposed blinds (%d/%d). Starting round.",
				lobby.TotalProposedBlinds, lobby.PlayerCount)

			// Start the round immediately instead of waiting for timeout
			go send_round_start_event(redisClient, db, lobbyID, sio)
		}
	}
}

// Function we should call when
func Send_chosen_blind(lobbyID string, rc *redis.RedisClient, sio *socketio_types.SocketServer) {
	lobby, err := rc.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("Error obtaining lobby to broadcast blind: %v", err)
		return
	}
	blind := lobby.CurrentBlind
	sio.Sio_server.To(socket.Room(lobbyID)).Emit("send_chosen_blind", blind)
}

/*func HandleStarttimeout(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, sio *socketio_types.SocketServer) func(args ...interface{}) {
	return func(args ...interface{}) {

		lobbyID := args[0].(string)
		timeout := args[1].(int)

		log.Printf("[TIMEOUT] Starting timeout of %d minutes for lobby %s", timeout, lobbyID)

		// Check if lobby exists
		var lobbyPG *models.GameLobby
		lobbyPG, err := utils.CheckLobbyExists(db, lobbyID)
		if err != nil {
			fmt.Println("Lobby does not exist:", lobbyID)
			client.Emit("error", gin.H{"error": "Lobby does not exist"})
			return
		}

		// Check if the user is in the lobby
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

		// Check if user is the host
		if username != lobbyPG.CreatorUsername {
			client.Emit("error", gin.H{"error": "Only the host can start the game"})
			return
		}

		// Start the timeout for the lobby
		lobby, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			log.Printf("[TIMEOUT-ERROR] Error obtaining lobby to start timeout: %v", err)
			return
		}

		// Check if the game is already in timeout
		if !lobby.Timeout.IsZero() {
			log.Printf("[TIMEOUT-ERROR] Game is already in timeout: %v", lobby.Timeout)
			return
		}

		// Set the timeout to the current time
		lobby.Timeout = time.Now()
		err = redisClient.SaveGameLobby(lobby)
		if err != nil {
			log.Printf("[TIMEOUT-ERROR] Error setting lobby timeout: %v", err)
			return
		}

		// Sleep for timeout duration
		time.Sleep(time.Minute * time.Duration(timeout))

		// Broadcast the timeout to all players in the lobby
		sio.Sio_server.To(socket.Room(lobbyID)).Emit("blind_timeout_expired", gin.H{"message": "Timeout reached"})
		log.Printf("[TIMEOUT] 2 minutes timeout reached %s", lobbyID)

		// Reset the timeout
		lobby.Timeout = time.Time{}
		err = redisClient.SaveGameLobby(lobby)
		if err != nil {
			log.Printf("[TIMEOUT-ERROR] Error resetting lobby timeout: %v", err)
			return
		}
		log.Printf("[TIMEOUT] Timeout reset for lobby %s", lobbyID)
	}
}*/

// Helper function to validate lobby and user, returning the lobby if valid
func validateLobbyAndUser(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string, lobbyID string) (*redis_models.GameLobby, error) {

	log.Printf("[TIMEOUT-REQUEST] Validating lobby %s and user %s", lobbyID, username)

	// Check if the user is in the lobby
	isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
	if err != nil {
		log.Printf("[TIMEOUT-ERROR] Database error: %v", err)
		client.Emit("error", gin.H{"error": "Database error"})
		return nil, err
	}

	if !isInLobby {
		log.Printf("[TIMEOUT-ERROR] User is NOT in lobby: %s, Lobby: %s", username, lobbyID)
		client.Emit("error", gin.H{"error": "You must join the lobby before requesting timeout info"})
		return nil, fmt.Errorf("user not in lobby")
	}

	// Get lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[TIMEOUT-ERROR] Error obtaining lobby: %v", err)
		client.Emit("error", gin.H{"error": "Error obtaining lobby information"})
		return nil, err
	}

	return lobby, nil
}

func HandleRequestBlindTimeout(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {
		if len(args) < 1 {
			client.Emit("error", gin.H{"error": "Lobby ID is required"})
			return
		}

		lobbyID := args[0].(string)
		log.Printf("[BLIND-TIMEOUT-REQUEST] Requesting blind timeout for lobby %s by user %s", lobbyID, username)

		lobby, err := validateLobbyAndUser(redisClient, client, db, username, lobbyID)
		if err != nil {
			return
		}

		client.Emit("blind_timeout_info", gin.H{
			"timeout": lobby.BlindTimeout,
		})
	}
}

func HandleRequestGameRoundTimeout(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {
		if len(args) < 1 {
			client.Emit("error", gin.H{"error": "Lobby ID is required"})
			return
		}

		lobbyID := args[0].(string)
		log.Printf("[GAME-ROUND-TIMEOUT-REQUEST] Requesting game round timeout for lobby %s by user %s", lobbyID, username)

		lobby, err := validateLobbyAndUser(redisClient, client, db, username, lobbyID)
		if err != nil {
			return
		}

		client.Emit("game_round_timeout_info", gin.H{
			"timeout": lobby.GameRoundTimeout,
		})
	}
}

func HandleRequestShopTimeout(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {
		if len(args) < 1 {
			client.Emit("error", gin.H{"error": "Lobby ID is required"})
			return
		}

		lobbyID := args[0].(string)
		log.Printf("[SHOP-TIMEOUT-REQUEST] Requesting shop timeout for lobby %s by user %s", lobbyID, username)

		lobby, err := validateLobbyAndUser(redisClient, client, db, username, lobbyID)
		if err != nil {
			return
		}

		client.Emit("shop_timeout_info", gin.H{
			"timeout": lobby.ShopTimeout,
		})
	}
}

func startBlindTimeout(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {

	log.Printf("[BLIND-TIMEOUT] Starting blind timeout for lobby %s", lobbyID)

	// Check if lobby exists
	_, err := utils.CheckLobbyExists(db, lobbyID)
	if err != nil {
		log.Printf("[BLIND-TIMEOUT-ERROR] Lobby does not exist: %s", lobbyID)
		return
	}

	// Start the timeout for the lobby
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
		send_round_start_event(redisClient, db, lobbyID, sio)
	}()

	log.Printf("[BLIND-TIMEOUT] Blind timeout started for lobby %s", lobbyID)
}

func send_round_start_event(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {
	log.Printf("[ROUND-START] Attempting to send round start event for lobby %s", lobbyID)

	// Get the game lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[ROUND-START-ERROR] Error getting lobby info: %v", err)
		return
	}

	// If the blind timeout is already reset, another process already started the round
	if lobby.BlindTimeout.IsZero() {
		log.Printf("[ROUND-START-INFO] Round already started for lobby %s, skipping", lobbyID)
		return
	}

	// Increment the round number
	// NOTE: the initial value on creation is 0 (see controllers/lobby.go/CreateLobby)
	lobby.CurrentRound++

	// Reset players finished round counter in redis
	lobby.TotalPlayersFinishedRound = 0

	// Get the blind value
	blind := lobby.CurrentBlind

	// Broadcast round start event to all players in the lobby with the current round number
	sio.Sio_server.To(socket.Room(lobbyID)).Emit("starting_round", gin.H{
		"round_number": lobby.CurrentRound,
		"blind":        blind,
	})

	log.Printf("[ROUND-START] Broadcast round start event to lobby %s with round %d and blind %d",
		lobbyID, lobby.CurrentRound, blind)

	// Reset the blind timeout to indicate round has started
	lobby.BlindTimeout = time.Time{}

	// Save the updated lobby
	err = redisClient.SaveGameLobby(lobby)
	if err != nil {
		log.Printf("[ROUND-START-ERROR] Error updating lobby state: %v", err)
	}

	// Start a timeout for the round
	startRoundPlayTimeout(redisClient, db, lobbyID, sio)
}

func startRoundPlayTimeout(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {
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
		handleRoundEnd(redisClient, db, lobbyID, sio)
	}()

	log.Printf("[ROUND-PLAY-TIMEOUT] Round play timeout started for lobby %s", lobbyID)
}

// Function to handle the end of a round
func handleRoundEnd(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {
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

	// Process eliminations based on blind achievement
	_, err = handlePlayerEliminations(redisClient, lobbyID, sio)
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
	shop, err := InitializeShop(lobbyID, lobby.CurrentRound)
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
		startShopTimeout(redisClient, db, lobbyID, sio)
	}

	log.Printf("[ROUND-END] Round ended for lobby %s", lobbyID)
}

// Separate function to handle player eliminations based on blind achievement
func handlePlayerEliminations(redisClient *redis.RedisClient, lobbyID string, sio *socketio_types.SocketServer) ([]string, error) {
	// List to track eliminated players
	var eliminatedPlayers []string

	// Get the lobby
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		return nil, fmt.Errorf("error getting lobby: %v", err)
	}

	highestBlindProposer := lobby.HighestBlindProposer
	blind := lobby.CurrentBlind

	if highestBlindProposer == "" {
		log.Printf("[ELIMINATION-INFO] No blind proposer found for lobby %s, skipping eliminations", lobbyID)
		return nil, nil
	}

	// Get all players in the lobby
	players, err := redisClient.GetAllPlayersInLobby(lobbyID)
	if err != nil {
		return nil, fmt.Errorf("error getting players: %v", err)
	}

	// Find the highest blind proposer player
	var proposerPlayer *redis_models.InGamePlayer
	for i := range players {
		if players[i].Username == highestBlindProposer {
			proposerPlayer = &players[i]
			break
		}
	}

	// Apply elimination rules
	if proposerPlayer != nil {
		proposerReachedBlind := proposerPlayer.CurrentPoints >= blind

		if !proposerReachedBlind {
			// Only eliminate the highest blind proposer
			eliminatedPlayers = append(eliminatedPlayers, highestBlindProposer)
			log.Printf("[ELIMINATION] Player %s eliminated for not reaching their proposed blind of %d (scored %d)",
				highestBlindProposer, blind, proposerPlayer.CurrentPoints)
		} else {
			// Eliminate all players who didn't reach the blind
			for _, player := range players {
				if player.CurrentPoints < blind {
					eliminatedPlayers = append(eliminatedPlayers, player.Username)
					log.Printf("[ELIMINATION] Player %s eliminated for not reaching the blind of %d (scored %d)",
						player.Username, blind, player.CurrentPoints)
				}
			}
		}

		// Remove eliminated players from Redis
		for _, username := range eliminatedPlayers {
			if err := redisClient.DeleteInGamePlayer(username, lobbyID); err != nil {
				log.Printf("[ELIMINATION-ERROR] Error removing player %s: %v", username, err)
			}
		}

		// Update player count
		if len(eliminatedPlayers) > 0 {
			lobby.PlayerCount -= len(eliminatedPlayers)
			if lobby.PlayerCount < 0 {
				lobby.PlayerCount = 0
			}

			if err := redisClient.SaveGameLobby(lobby); err != nil {
				log.Printf("[ELIMINATION-ERROR] Error updating player count: %v", err)
			}

			// Broadcast the eliminated players
			sio.Sio_server.To(socket.Room(lobbyID)).Emit("players_eliminated", gin.H{
				"eliminated_players": eliminatedPlayers,
				"reason":             "blind_check",
				"blind_value":        blind,
			})
		}
	}

	return eliminatedPlayers, nil
}

// Function to start the shop timeout
func startShopTimeout(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {
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
		// Sleep for shop duration (e.g., 1 minute)
		time.Sleep(1 * time.Minute)

		// Handle shop phase end
		handleShopEnd(redisClient, db, lobbyID, sio)
	}()

	log.Printf("[SHOP-TIMEOUT] Shop timeout started for lobby %s", lobbyID)
}

// Empty function for now - will handle shop phase end
func handleShopEnd(redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, sio *socketio_types.SocketServer) {
	log.Printf("[SHOP-END] Shop phase ended for lobby %s", lobbyID)

	// Get the lobby
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[SHOP-END-ERROR] Error getting lobby: %v", err)
		return
	}

	// Check if shop timeout is already reset
	if lobby.ShopTimeout.IsZero() {
		log.Printf("[SHOP-END-INFO] Shop already ended for lobby %s, skipping", lobbyID)
		return
	}

	// Reset the shop timeout
	lobby.ShopTimeout = time.Time{}

	// Save the updated lobby
	err = redisClient.SaveGameLobby(lobby)
	if err != nil {
		log.Printf("[SHOP-END-ERROR] Error saving lobby: %v", err)
	}

	// Broadcast shop end
	sio.Sio_server.To(socket.Room(lobbyID)).Emit("shop_phase_end", gin.H{
		"message": "Shop phase has ended",
	})

	// Start the next round's blind voting phase
	startBlindTimeout(redisClient, nil, db, lobbyID, sio)
}
