package controllers

import (
	"Nogler/middleware"
	"Nogler/models/postgres"
	models "Nogler/models/postgres"
	"log"

	// "errors"
	"net/http"
	"Nogler/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// @Summary Creates a new lobby
// @Description Returns the id of a new created lobby?
// @Tags lobby
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @Success 200 {array} object{message=string,lobby_id=integer}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /auth/CreateLobby [post]
// @Security ApiKeyAuth
func CreateLobby(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

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
		// Here we already have the username for the creator of the lobby

		// *There is a function on the models gamelobby "beforeCreate" for the id generation

		NewLobby := postgres.GameLobby{
			CreatorUsername: username,
			NumberOfRounds:  0,
			TotalPoints:     0,
		}

		if err := db.Create(&NewLobby).Error; err != nil {
			log.Fatal("Failed to create lobby:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating lobby"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"lobby_id": NewLobby.ID, "message": "Lobby created sucessfully"})
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
// @Success 200 {object} object{lobby_id=string,creator_username=string,number_rounds=integer,total_points=integer,created_at=string}
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

		c.JSON(http.StatusOK, gin.H{
			"lobby_id":         lobby.ID,
			"creator_username": lobby.CreatorUsername,
			"number_rounds":    lobby.NumberOfRounds,
			"total_points":     lobby.TotalPoints,
			"created_at":       lobby.CreatedAt,
		})
	}
}

// @Summary Lists all existing lobbies
// @Description Returns a list of all the lobbies
// @Tags lobby
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @in header
// @Success 200 {array} object{lobby_id=string,creator_username=string,number_rounds=integer,total_points=integer,created_at=string}
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

		var game_lobbies []models.GameLobby
		
		// Get all lobbies from database
		if err := db.Find(&game_lobbies).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve lobbies"})
			return
		}

		// Create a slice of lobbies
		lobbies := make([]gin.H, len(game_lobbies))
		for i, lobby := range game_lobbies {
			lobbies[i] = gin.H{
				"lobby_id":         lobby.ID,
				"creator_username": lobby.CreatorUsername,
				"number_rounds":    lobby.NumberOfRounds,
				"total_points":     lobby.TotalPoints,
				"created_at":       lobby.CreatedAt,
			}
		}
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
func JoinLobby(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		lobbyID := c.Param("lobby_id")

		var lobby postgres.GameLobby
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
		var userInLobby postgres.InGamePlayer
		result = db.Where(
			"lobby_id = ? AND username = ?",
			lobbyID, username,
		).First(&userInLobby)

		if result.RowsAffected > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User exists in a lobby"})
			return
		}

		gamePlayer := postgres.InGamePlayer{
			LobbyID:    lobbyID,
			Username: 	username,
		}

		result = db.Create(&gamePlayer)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding user to lobby"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "joined lobby successfully"})
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

		var lobby postgres.GameLobby
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
		var userInLobby postgres.InGamePlayer
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
}

// @Summary Sends a lobby invitation
// @Description Sends a lobby invitation from the sender to another user
// @Tags lobby
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @Param lobby_id formData int true "Lobby ID"
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
		var friendship postgres.Friendship
		if err := db.Where("(username1 = ? AND username2 = ?) OR (username1 = ? AND username2 = ?)",username, friendUsername, friendUsername, username,).First(&friendship).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Friendship does not exist"})
			return
		}

		// Get the lobby ID from the URL parameters
		lobbyID := c.PostForm("lobby_id")

		// Check if lobby exists
		var lobby postgres.GameLobby
		if err := db.Where("id = ?", lobbyID).First(&lobby).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Lobby does not exist"})
			return
		}

		// Check if the invitation already exists
		var existingInvitation postgres.GameInvitation
		if err := db.Where("lobby_id = ? AND sender_username = ? AND invited_username = ?", lobbyID, username, friendUsername).First(&existingInvitation).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invitation already sent to this user"})
			return
		}

		// Create and save new friendship
		newLobbyInvitation := postgres.GameInvitation{
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
