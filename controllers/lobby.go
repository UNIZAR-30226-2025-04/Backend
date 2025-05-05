package controllers

import (
	game_constants "Nogler/constants/game"
	"Nogler/middleware"
	models "Nogler/models/postgres"
	redis_models "Nogler/models/redis"
	"Nogler/services/poker"
	"Nogler/services/redis"
	"Nogler/services/socket_io/utils/game_flow"
	"Nogler/utils"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TODO: we need to add an option to make the created lobby public or private

// @Summary Creates a new lobby
// @Description Returns the id of a new created lobby
// @Tags lobby
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @Param public formData int true "Set to 1 for public lobby, 2 for AI lobby and 0 for private lobby"
// @Success 200 {object} object{message=string,lobby_id=string}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /auth/CreateLobby [post]
// @Security ApiKeyAuth
func CreateLobby(db *gorm.DB, redisClient *redis.RedisClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		email, err := middleware.JWT_decoder(c)
		if err != nil {
			log.Print("Error en jwt...")
			return
		}

		// Parse public parameter with default to false (private)
		isPublic := 0
		publicParam := c.PostForm("public")
		if publicParam == "1" {
			isPublic = 1
		} else if publicParam == "2" {
			isPublic = 2
		} else if publicParam == "0" {
			isPublic = 0
		}

		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found: invalid email"})
			return
		}

		username := user.ProfileUsername
		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username is required"})
			return
		}

		// Create lobby with public/private setting
		NewLobby := models.GameLobby{
			CreatorUsername: username,
			NumberOfRounds:  0,
			TotalPoints:     0,
			IsPublic:        isPublic, // Set from parameter
		}

		if err := db.Create(&NewLobby).Error; err != nil {
			log.Printf("Failed to create lobby: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating lobby"})
			return
		}

		// Create corresponding Redis lobby with matching public/private setting
		redisLobby := &redis_models.GameLobby{
			Id:                      NewLobby.ID,
			CreatorUsername:         username,
			MaxRounds:               game_constants.MaxGameRounds,
			TotalPoints:             0,
			CreatedAt:               NewLobby.CreatedAt,
			GameHasBegun:            false,
			IsPublic:                isPublic,
			CurrentHighBlind:        0,
			NumberOfVotes:           0,
			CurrentRound:            0,
			ProposedBlinds:          make(map[string]bool),
			PlayersFinishedRound:    make(map[string]bool),
			PlayersFinishedShop:     make(map[string]bool),
			PlayersFinishedVouchers: make(map[string]bool),
			PlayerCount:             0,
			BlindTimeout:            time.Time{},
			GameRoundTimeout:        time.Time{},
			ShopTimeout:             time.Time{},
			VouchersTimeout:         time.Time{},
			BlindsCompleted:         make(map[int]bool),
			GameRoundsCompleted:     make(map[int]bool),
			ShopsCompleted:          make(map[int]bool),
			VouchersCompleted:       make(map[int]bool),
			CurrentPhase:            redis_models.PhaseNone, // Initialize with "none" phase
			CurrentBaseBlind:        game_constants.BASE_BLIND,
		}

		if isPublic == 2 {
			// Add bot to Redis lobby
			purchasedPackCards := make([]poker.Card, 0)
			purchasedPackCardsJSON, _ := json.Marshal(purchasedPackCards)
			redisAIPlayer := &redis_models.InGamePlayer{
				Username:                game_flow.FormatAIPlayerName(NewLobby.ID),
				LobbyId:                 NewLobby.ID,
				PlayersMoney:            10, // Initial money --> TODO: ver cuánto es la cifra inicial
				Rerolls:                 0,
				CurrentDeck:             poker.InitializePlayerDeck(), // Will be initialized when game starts
				Modifiers:               nil,                          // Will be initialized when game starts
				CurrentJokers:           nil,                          // Will be initialized when game starts
				MostPlayedHand:          nil,                          // Will be initialized during game
				HandPlaysLeft:           game_constants.TOTAL_HAND_PLAYS,
				DiscardsLeft:            game_constants.TOTAL_DISCARDS,
				Winner:                  false,
				CurrentRoundPoints:      0,
				TotalGamePoints:         0,
				BetMinimumBlind:         true,
				IsBot:                   true, // Mark as AI player
				LastPurchasedPackItemId: -1,
				PurchasedPackCards:      purchasedPackCardsJSON,
			}

			// Save the AI player in Redis
			if err := redisClient.SaveInGamePlayer(redisAIPlayer); err != nil {
				log.Printf("[AI-ERROR] Error saving AI player in Redis: %v", err)
				// Rollback lobby creation in Redis and PostgreSQL
				if err := redisClient.DeleteGameLobby(NewLobby.ID); err != nil {
					log.Printf("Failed to rollback Redis lobby creation: %v", err)
				}
				if err := db.Delete(&NewLobby).Error; err != nil {
					log.Printf("Failed to rollback PostgreSQL lobby creation: %v", err)
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving AI player in Redis"})
				return
			}

			// Update player count in Redis lobby
			redisLobby.PlayerCount = 1
		}

		// Save the lobby in Redis
		if err := redisClient.SaveGameLobby(redisLobby); err != nil {
			log.Printf("Failed to create lobby in Redis: %v", err)
			if err := db.Delete(&NewLobby).Error; err != nil {
				log.Printf("Failed to rollback PostgreSQL lobby creation: %v", err)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating lobby in Redis"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"lobby_id": NewLobby.ID,
			"message":  "Lobby created successfully",
			"public":   isPublic,
		})

		// TODO!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! INITIAL USER DOESNT HAVE DECK SINCE HE DOESNT EXPLICITLY JOIN??
	}

	// NOTE: after this endpoint returns the response to the client, the client should initiate the
	// socket.io connection with the server. For example:
	/*
		const socket = io('http://nogler.ddns.net:8080');
		socket.emit('joinLobby', { lobbyId: response.lobby_id });
	*/
}

// @Summary Gives info of a lobby
// @Description Given a lobby id, it will return its information
// @Tags lobby
// @Accept x-www-form-urlencoded
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @in header
// @Param lobby_id path string true "Id of the lobby wanted"
// @Success 200 {object} object{lobby_id=string,creator_username=string,number_rounds=integer,total_points=integer,created_at=string,is_public=boolean,number_players=integer,players=[]string}
// @Failure 400 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /auth/lobbyInfo/{lobby_id} [get]
// @Security ApiKeyAuth
func GetLobbyInfo(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		lobbyID := c.Param("lobby_id")

		lobby, err := utils.CheckLobbyExists(db, lobbyID)

		if err != nil {
			if err.Error() == "lobby not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		var usersInLobby []string
		if err := db.Model(&models.InGamePlayer{}).Where("lobby_id = ?", lobbyID).Pluck("username", &usersInLobby).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users in lobby"})
			return
		}

		number := len(usersInLobby)

		c.JSON(http.StatusOK, gin.H{
			"lobby_id":         lobby.ID,
			"creator_username": lobby.CreatorUsername,
			"number_rounds":    lobby.NumberOfRounds,
			"total_points":     lobby.TotalPoints,
			"created_at":       lobby.CreatedAt,
			"is_public":        lobby.IsPublic,
			"number_players":   number,
			"players":          usersInLobby,
		})
	}
}

// @Summary Lists all existing public lobbies
// @Description Returns a list of all public lobbies with player count
// @Tags lobby
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @in header
// @Success 200 {array} object{lobby_id=string,creator_username=string,number_rounds=integer,total_points=integer,created_at=string,host_icon=integer,player_count=integer,is_public=boolean}
// @Failure 401 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /auth/getAllLobbies [get]
// @Security ApiKeyAuth
func GetAllLobbies(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate JWT token
		email, err := middleware.JWT_decoder(c)
		if err != nil {
			log.Print("Error en jwt...")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Verify user exists
		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found: invalid email"})
			return
		}

		var gameLobbies []models.GameLobby

		// Get only public lobbies from database
		if err := db.Where("is_public = ?", 1).Find(&gameLobbies).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve lobbies"})
			return
		}

		// Extract all creator usernames
		var usernames []string
		for _, lobby := range gameLobbies {
			usernames = append(usernames, lobby.CreatorUsername)
		}

		// Get host icons (same logic as before)
		hostIcons := make(map[string]int)
		var profiles []struct {
			Username string
			UserIcon int
		}

		if err := db.Model(&models.GameProfile{}).Where("username IN ?", usernames).Select("username, user_icon").Find(&profiles).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve host icons"})
			return
		}

		for _, profile := range profiles {
			hostIcons[profile.Username] = profile.UserIcon
		}

		// Initialize a map to store player counts by lobby_id
		playerCounts := make(map[string]int)

		// Count players for each lobby
		var playerCountResult []struct {
			LobbyID     string
			PlayerCount int64
		}

		if err := db.Model(&models.InGamePlayer{}).
			Select("lobby_id, COUNT(*) AS player_count").
			Group("lobby_id").
			Find(&playerCountResult).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count players"})
			return
		}

		// Populate the playerCounts map
		for _, result := range playerCountResult {
			playerCounts[result.LobbyID] = int(result.PlayerCount)
		}

		// Prepare the lobbies slice with player count and other information
		lobbies := make([]gin.H, len(gameLobbies))
		for i, lobby := range gameLobbies {
			lobbies[i] = gin.H{
				"lobby_id":         lobby.ID,
				"creator_username": lobby.CreatorUsername,
				"number_rounds":    lobby.NumberOfRounds,
				"total_points":     lobby.TotalPoints,
				"created_at":       lobby.CreatedAt,
				"host_icon":        hostIcons[lobby.CreatorUsername],
				"player_count":     playerCounts[lobby.ID], // Add player count from the map
				"is_public":        1,                      // Always true since we're filtering for public lobbies
			}
		}

		// Return the lobbies with player count
		c.JSON(http.StatusOK, lobbies)
	}
}

// @Summary Inserts a user into a lobby
// @Description Adds the user to the relation user-lobby
// @Tags lobby
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @Param lobby_id path string true "lobby_id"
// @in header
// @Success 200 {object} object{message=string,lobby_info=object{id=string,creator=string,number_rounds=integer,total_points=integer,game_has_begun=boolean,public=boolean}}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Security ApiKeyAuth
// @Router /auth/joinLobby/{lobby_id} [post]
func JoinLobby(db *gorm.DB, redisClient *redis.RedisClient) gin.HandlerFunc {
	return func(c *gin.Context) {

		lobbyID := c.Param("lobby_id")

		var lobby models.GameLobby
		result := db.Where("id = ?", lobbyID).First(&lobby)

		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Lobby not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			}
			return
		}

		email, err := middleware.JWT_decoder(c)
		if err != nil {
			log.Print("Error en jwt...")
			return
		}

		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found: invalid email"})
			return
		}

		username := user.ProfileUsername

		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username is required"})
			return
		}

		// Check if the user in lobby exists
		var userInLobby models.InGamePlayer
		result = db.Where(
			"lobby_id = ? AND username = ?",
			lobbyID, username,
		).First(&userInLobby)

		if result.RowsAffected > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User already exists in a lobby"})
			return
		}

		gamePlayer := models.InGamePlayer{
			LobbyID:  lobbyID,
			Username: username,
		}

		// Check if the lobby is full
		var playersInLobby []models.InGamePlayer
		if err := db.Where("lobby_id = ?", lobbyID).Find(&playersInLobby).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users in lobby"})
			return
		}

		if len(playersInLobby) >= 8 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Lobby is full"})
			return
		}

		// Check if the game has already started
		if lobby.GameHasBegun {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Game has already started"})
			return
		}

		// Start transaction
		tx := db.Begin()
		if err := tx.Create(&gamePlayer).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding user to the lobby"})
			return
		}

		// Create Redis InGamePlayer entry
		purchasedPackCards := make([]poker.Card, 0)
		purchasedPackCardsJSON, _ := json.Marshal(purchasedPackCards)
		redisPlayer := &redis_models.InGamePlayer{
			Username:     username,
			LobbyId:      lobbyID,
			Rerolls:      0,
			PlayersMoney: 10,                           // Initial money --> TODO: ver cuánto es la cifra inicial
			CurrentDeck:  poker.InitializePlayerDeck(), // Will be initialized when game starts
			// TODO: see in_game_player.go
			// PlayersRemainingCards: 52,
			Modifiers:          nil, // Will be initialized when game starts
			CurrentJokers:      nil, // Will be initialized when game starts
			MostPlayedHand:     nil, // Will be initialized during game
			HandPlaysLeft:      game_constants.TOTAL_HAND_PLAYS,
			DiscardsLeft:       game_constants.TOTAL_DISCARDS,
			Winner:             false,
			CurrentRoundPoints: 0,
			TotalGamePoints:    0,
			BetMinimumBlind:    true,
			// NEW: initially, the player hasn't bought any packs yet
			LastPurchasedPackItemId:     -1,
			PurchasedPackCards:          purchasedPackCardsJSON,
			CurrentShopPurchasedItemIDs: make(map[int]bool),
		}

		if err := redisClient.SaveInGamePlayer(redisPlayer); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding user to Redis lobby"})
			return
		}

		// Get Redis lobby to update
		redisLobby, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving Redis lobby"})
			return
		}

		// Commit PostgreSQL transaction
		if err := tx.Commit().Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error committing transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "joined lobby successfully",
			"lobby_info": gin.H{
				"id":             redisLobby.Id,
				"creator":        redisLobby.CreatorUsername,
				"number_rounds":  redisLobby.MaxRounds,
				"total_points":   redisLobby.TotalPoints,
				"game_has_begun": redisLobby.GameHasBegun,
				"public":         redisLobby.IsPublic,
			},
		})
	}
}

// @Summary Sends a lobby invitation
// @Description Sends a lobby invitation from the sender to another user
// @Tags lobby
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @Param lobby_id formData string true "Lobby ID"
// @Param friendUsername formData string true "Username of the recipient"
// @Success 200 {object} object{message=string} "Lobby invitation sent successfully"
// @Failure 400 {object} object{error=string} "Friendship does not exist"
// @Failure 401 {object} object{error=string} "User not authenticated"
// @Failure 500 {object} object{error=string} "Error sending invitation"
// @Router /auth/sendLobbyInvitation [post]
// @Security ApiKeyAuth
func SendLobbyInvitation(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		email, err := middleware.JWT_decoder(c)
		if err != nil {
			log.Print("Error in jwt...")
			return
		}

		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}

		username := user.ProfileUsername
		friendUsername := c.PostForm("friendUsername")

		if username == "" || friendUsername == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Both usernames are required"})
			return
		}

		// Check if the friendship exists
		var friendship models.Friendship
		if err := db.Where("(username1 = ? AND username2 = ?) OR (username1 = ? AND username2 = ?)", username, friendUsername, friendUsername, username).First(&friendship).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Friendship does not exist"})
			return
		}

		// Get the lobby ID from the URL parameters
		lobbyID := c.PostForm("lobby_id")

		// Check if lobby exists
		var lobby models.GameLobby
		if err := db.Where("id = ?", lobbyID).First(&lobby).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Lobby does not exist"})
			return
		}

		// Check if the invitation already exists
		var existingInvitation models.GameInvitation
		if err := db.Where("lobby_id = ? AND sender_username = ? AND invited_username = ?", lobbyID, username, friendUsername).First(&existingInvitation).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invitation already sent to this user"})
			return
		}

		// Create and save new friendship
		newLobbyInvitation := models.GameInvitation{
			LobbyID:         lobbyID,
			SenderUsername:  username,
			InvitedUsername: friendUsername,
		}

		if err := db.Create(&newLobbyInvitation).Error; err != nil {
			log.Fatal("Failed to send lobby invitation:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error sending invitation"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Lobby invitation sent successfully"})
	}
}

// @Summary Returns a lobby code
// @Description Returns the code of a lobby with a similiar score to the user
// @Tags lobby
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @Success 200 {object} object{lobby_id=string,message=string}
// @Failure 400 {object} object{error=string} "User not found"
// @Failure 401 {object} object{error=string} "User not authenticated"
// @Failure 500 {object} object{error=string} "Error retrieving lobby"
// @Router /auth/matchMaking [get]
// @Security ApiKeyAuth
func MatchMaking(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Validate JWT token
		email, err := middleware.JWT_decoder(c)
		if err != nil {
			log.Print("Error en jwt...")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Verify user exists
		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found: invalid email"})
			return
		}

		// Get user's game profile
		var userProfile models.GameProfile
		if err := db.Where("username = ?", user.ProfileUsername).First(&userProfile).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user's game profile"})
			return
		}

		var lobby models.GameLobby
		// Get public and not started lobbies players from database
		if err := db.Preload("InGamePlayers").Where("game_has_begun = ? AND is_public = ?", false, 1).Take(&lobby).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve lobby"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"lobby_id": lobby.ID})
	}
}

// @Summary Updates lobby visibility (public/private)
// @Description Toggles a lobby between public and private. Only the creator can change this setting.
// @Tags lobby
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @Param lobby_id path string true "Lobby ID"
// @Param is_public formData boolean true "Set to true for public lobby, false for private lobby"
// @Success 200 {object} object{message=string,is_public=boolean}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 403 {object} object{error=string} "User is not the lobby creator"
// @Failure 404 {object} object{error=string} "Lobby not found"
// @Failure 500 {object} object{error=string}
// @Router /auth/setLobbyVisibility/{lobby_id} [post]
// @Security ApiKeyAuth
func SetLobbyVisibility(db *gorm.DB, redisClient *redis.RedisClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get lobby ID from parameters
		lobbyID := c.Param("lobby_id")

		// Get visibility parameter
		isPublicStr := c.PostForm("is_public")

		// NOTE: should be named something like "game_type", the values it takes are:
		// 0: private
		// 1: public
		// 2: vs AI
		var isPublic int
		if isPublicStr == "true" {
			isPublic = 1
		} else {
			isPublic = 0
		}

		// Get user from JWT token
		email, err := middleware.JWT_decoder(c)
		if err != nil {
			log.Print("Error in JWT...")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found: invalid email"})
			return
		}

		username := user.ProfileUsername

		// Find the lobby
		var lobby models.GameLobby
		if err := db.Where("id = ?", lobbyID).First(&lobby).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Lobby not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		// Verify the user is the lobby creator
		if lobby.CreatorUsername != username {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only the lobby creator can change visibility settings"})
			return
		}

		// Start transaction
		tx := db.Begin()

		// Update visibility in PostgreSQL
		if err := tx.Model(&lobby).Update("is_public", isPublic).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update lobby visibility"})
			return
		}

		// Update visibility in Redis
		redisLobby, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve Redis lobby"})
			return
		}

		redisLobby.IsPublic = isPublic
		if err := redisClient.SaveGameLobby(redisLobby); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Redis lobby"})
			return
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error committing transaction"})
			return
		}

		// Return success
		c.JSON(http.StatusOK, gin.H{
			"message":   "Lobby visibility updated successfully",
			"is_public": isPublic,
		})
	}
}

// CheckUserLobby Verfies if a user is in a lobby and returns the id of the lobby if true
// @Summary Checks if user is in a lobby
// @Description Returns true or false, and if true, return the id of the user the lobby is in, and if false, empty string
// @Tags lobby
// @Param Authorization header string true "Bearer JWT token"
// @Produce json
// @Success 200 {object} object{in_lobby=boolean,lobby_id=string}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /auth/isUserInLobby [get]
func IsUserInLobby(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		email, err := middleware.JWT_decoder(c)
		if err != nil {
			log.Print("JWT error...")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found: invalid email"})
			return
		}

		username := user.ProfileUsername

		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username is required"})
			return
		}

		var inGamePlayer models.InGamePlayer
		if err := db.Where("username = ?", username).First(&inGamePlayer).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusOK, gin.H{"in_lobby": false, "lobby_id": "", "public": ""})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting lobby"})
			return
		}

		var lobby models.GameLobby
		if err := db.Where("id = ?", inGamePlayer.LobbyID).First(&lobby).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Lobby not found"})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"in_lobby": true,
			"lobby_id": inGamePlayer.LobbyID,
			"public":   lobby.IsPublic,
		})
	}
}
