package controllers

import (
	"Nogler/middleware"
	"Nogler/models/postgres"
	models "Nogler/models/postgres"
	"log"

	// "errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"gorm.io/gorm"
)

// @Summary Creates a new lobby
// @Description Returns the id of a new created lobby?
// @Tags lobby
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @Success 200 {array} object{message=string,lobby_id=integer}
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


// @Summary Gives info of a lobby
// @Description Given a lobby id, it will return its information
// @Tags lobby
// @Accept x-www-form-urlencoded
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @in header
// @Param lobby_id path string true "Id of the lobby wanted"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /auth/lobbyInfo/{lobby_id} [get]
// @Security ApiKeyAuth
func GetLobbyInfo(db *gorm.DB) gin.HandlerFunc {
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
// @Success 200 {object} object{message=string}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Security ApiKeyAuth
// @Router /auth/getAllLobbies [get]
func GetAllLobies(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		var game_lobbies []models.GameLobby

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
