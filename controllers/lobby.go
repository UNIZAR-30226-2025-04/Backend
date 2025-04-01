package controllers

import (
	"Nogler/middleware"
	models "Nogler/models/postgres"
	redis_models "Nogler/models/redis"
	"Nogler/services/redis"
	"Nogler/utils"
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
// @Param public formData boolean false "Set to true for public lobby, false or omitted for private lobby"
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
		isPublic := false
		publicParam := c.PostForm("public")
		if publicParam == "true" {
			isPublic = true
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
			Id:              NewLobby.ID,
			CreatorUsername: username,
			NumberOfRounds:  0,
			TotalPoints:     0,
			CreatedAt:       NewLobby.CreatedAt,
			GameHasBegun:    false,
			IsPublic:        isPublic, // Set from parameter
			ChatHistory:     []redis_models.ChatMessage{},
		}

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
// @Success 200 {object} object{lobby_id=string,creator_username=string,number_rounds=integer,total_points=integer,created_at=string,number_players=integer,players=[]string}
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
		if err := db.Where("is_public = ?", true).Find(&gameLobbies).Error; err != nil {
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
				"is_public":        true,                   // Always true since we're filtering for public lobbies
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
// @Success 200 {object} object{message=string}
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

		// Start transaction
		tx := db.Begin()
		if err := tx.Create(&gamePlayer).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding user to the lobby"})
			return
		}

		// Create Redis InGamePlayer entry
		redisPlayer := &redis_models.InGamePlayer{
			Username:       username,
			LobbyId:        lobbyID,
			PlayersMoney:   0,   // Initial money --> TODO: ver cuánto es la cifra inicial
			CurrentDeck:    nil, // Will be initialized when game starts
			Modifiers:      nil, // Will be initialized when game starts
			CurrentJokers:  nil, // Will be initialized when game starts
			MostPlayedHand: nil, // Will be initialized during game
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
				"number_rounds":  redisLobby.NumberOfRounds,
				"total_points":   redisLobby.TotalPoints,
				"game_has_begun": redisLobby.GameHasBegun,
			},
		})
	}
}

// @Summary Removes the user from the lobby
// @Description Removes the user to the relation user-lobby
// @Tags lobby
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @Param lobby_id path string true "lobby_id"
// @in header
// @Success 200 {object} object{message=string}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Security ApiKeyAuth
// @Router /auth/exitLobby/{lobby_id} [post]
func ExitLobby(db *gorm.DB) gin.HandlerFunc {
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

		if result.RowsAffected == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User does not exist in that lobby"})
			return
		}

		// Delete the friendship
		result = db.Delete(&userInLobby)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting user from lobby"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Exited lobby successfully"})
	}

	// TODO: usar la redis, hacer que esto sea un método de socketio?
	// Si no, igual es un poco lío tener que pillar la conexión del socket
	// aquí y llamar a socket.Leave(roomId)
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

// TODO: ver cómo integramos la Redis aquí
// @Summary Kicks a user from a lobby
// @Description Host can remove another user from their lobby
// @Tags lobby
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @Param lobby_id path string true "lobby_id"
// @Param username path string true "username to kick"
// @in header
// @Success 200 {object} object{message=string}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 403 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Security ApiKeyAuth
// @Router /auth/kickFromLobby/{lobby_id}/{username} [post]
func KickFromLobby(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get lobby ID and username to kick from parameters
		lobbyID := c.Param("lobby_id")
		usernameToKick := c.Param("username")

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

		// Get the requesting user's info from JWT
		email, err := middleware.JWT_decoder(c)
		if err != nil {
			log.Print("Error in jwt...")
			return
		}

		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found: invalid email"})
			return
		}

		// Check if the requesting user is the host
		if user.ProfileUsername != lobby.CreatorUsername {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only the host can kick players"})
			return
		}

		// Check if the user to kick exists in the lobby
		var userInLobby models.InGamePlayer
		result := db.Where(
			"lobby_id = ? AND username = ?",
			lobbyID, usernameToKick,
		).First(&userInLobby)

		if result.RowsAffected == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User is not in the lobby"})
			return
		}

		// Cannot kick yourself (the host)
		if usernameToKick == user.ProfileUsername {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Host cannot kick themselves"})
			return
		}

		// Delete the player from lobby
		if err := db.Delete(&userInLobby).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error kicking user from lobby"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Player kicked successfully"})
	}

	// TODO: quitar este método y hacer que sea una funcionalidad directamente
	// de socketio, más fácil porque si no habría que pillar aquí la conexión del
	// socket y llamar a socket.Leave(roomId)
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
// @Router /auth/matchMaking [post]
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

		userScore := userProfile.UserScore
		difference := 100

		log.Printf("User score: %d", userScore)

		for i := 0; i < 10; i++ {
			var averageScore int

			var gameLobbies []models.GameLobby
			// Get public and not started lobbies players from database
			if err := db.Preload("InGamePlayers").Where("game_has_begun = ? AND is_public = ?", false, true).Find(&gameLobbies).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve lobbies"})
				return
			}

			log.Printf("Found %d game lobbies", len(gameLobbies))

			// Search for a lobby with players with similar score
			for _, lobby := range gameLobbies {
				// Only lobbies with less than 8 players and more than 0
				if len(lobby.InGamePlayers) >= 8 || len(lobby.InGamePlayers) == 0 {
					continue
				}

				// Get usernames of players in the lobby
				var usernames []string
				for _, player := range lobby.InGamePlayers {
					usernames = append(usernames, player.Username)
				}

				// Get score of the player from PostgreSQL
				var players []models.GameProfile
				if err := db.Where("username IN ?", usernames).Find(&players).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve players in lobby"})
					return
				}

				// Calculate total score of the lobby
				var totalScore int
				for _, player := range players {
					totalScore += player.UserScore
				}
				// Calculate average score
				averageScore = totalScore / len(players)

				log.Printf("Lobby ID: %s, Average score: %d", lobby.ID, averageScore)

				// Check if the lobby is suitable for the user
				if userScore >= averageScore-difference && userScore <= averageScore+difference {
					c.JSON(http.StatusOK, gin.H{
						"lobby_id": lobby.ID,
						"message":  "Lobby found",
					})
					return
				}
			}
			difference += 100
			time.Sleep(2 * time.Second) // Sleep for 2 seconds before searching again
		}

		// If no lobby is found, return an empty lobby ID
		c.JSON(http.StatusOK, gin.H{
			"lobby_id": "",
			"message":  "No lobbies found",
		})
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
		isPublic := isPublicStr == "true"

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
