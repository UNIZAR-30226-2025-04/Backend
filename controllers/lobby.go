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
// @Success 200 {array} object{username=string,icon=integer}
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

		var friendships []postgres.Friendship
		result := db.Where("username1 = ? OR username2 = ?", username, username).Find(&friendships)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching friendships"})
			return
		}

		friendsUsernames := []string{}
		for _, friendship := range friendships {
			if friendship.Username1 == username {
				friendsUsernames = append(friendsUsernames, friendship.Username2)
			} else {
				friendsUsernames = append(friendsUsernames, friendship.Username1)
			}
		}

		// Fetch friend profiles
		var friends []postgres.GameProfile
		if len(friendsUsernames) > 0 {
			result = db.Where("username IN (?)", friendsUsernames).Find(&friends)
			if result.Error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching friends data"})
				return
			}
		}

		simplifiedFriends := make([]gin.H, len(friends))
		for i, friend := range friends {
			simplifiedFriends[i] = gin.H{
				"username": friend.Username,
				"icon":     friend.UserIcon,
			}
		}

		c.JSON(http.StatusOK, simplifiedFriends)

	}
}

// @Summary Add a new friend
// @Description Adds a new friend to the user's friend list
// @Tags friends
// @Accept x-www-form-urlencoded
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @in header
// @Param friendUsername formData string true "Username of the friend to be added"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /auth/lobbyInfo/:lobby_id [get]
// @Security ApiKeyAuth
func GetLobbyInfo(db *gorm.DB) gin.HandlerFunc {
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
		friendUsername := c.PostForm("friendUsername")

		if username == "" || friendUsername == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Both usernames are required"})
			return
		}

		if username == friendUsername {
			c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot add yourself as a friend"})
			return
		}

		// Check if friendship already exists
		var existingFriendship postgres.Friendship
		result := db.Where(
			"(username1 = ? AND username2 = ?) OR (username1 = ? AND username2 = ?)",
			username, friendUsername, friendUsername, username,
		).First(&existingFriendship)

		if result.RowsAffected > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Friendship already exists"})
			return
		}

		// Create and save new friendship
		newFriendship := postgres.Friendship{
			Username1: username,
			Username2: friendUsername,
		}

		result = db.Create(&newFriendship)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding friend"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Friend added successfully"})
	}
}

// @Summary Lists all existing lobbies
// @Description Returns a list of all the lobbies
// @Tags lobby
// @Accept json
// @Produce json
// @in header
// @Success 200 {object} object{message=string}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
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
	}
}

/*
// @Summary Send a friend request
// @Description Sends a friend request from the sender to another user
// @Tags friends
// @Accept x-www-form-urlencoded
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @in header
// @Param friendUsername formData string true "Username of the recipient"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Security ApiKeyAuth
// @Router /auth/sendFriendshipRequest [post]
func SendFriendshipRequest(db *gorm.DB) gin.HandlerFunc {
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

		senderUsername := user.ProfileUsername

		receiverUsername := c.PostForm("friendUsername")

		if senderUsername == "" || receiverUsername == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Both usernames are required", "senderUsername": senderUsername, "recieverUsername": receiverUsername})
			return
		}

		if senderUsername == receiverUsername {
			c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot send a friend request to yourself"})
			return
		}

		// Check if recipient exists
		var receiver postgres.GameProfile
		result := db.Where("username = ?", receiverUsername).First(&receiver)
		if result.Error != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Receiver user not found"})
			return
		}

		// Check if they are already friends
		var existingFriendship postgres.Friendship
		result = db.Where(
			"(username1 = ? AND username2 = ?) OR (username1 = ? AND username2 = ?)",
			senderUsername, receiverUsername, receiverUsername, senderUsername,
		).First(&existingFriendship)

		if result.RowsAffected > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "You are already friends"})
			return
		}

		// Check if a friend request already exists
		var existingRequest postgres.FriendshipRequest
		result = db.Where(
			"(sender = ? AND recipient = ?)",
			senderUsername, receiverUsername,
		).First(&existingRequest)

		if result.RowsAffected > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Friend request already sent"})
			return
		}

		// Create and save the new friend request
		friendRequest := postgres.FriendshipRequest{
			Sender:    senderUsername,
			Recipient: receiverUsername,
		}

		result = db.Create(&friendRequest)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error sending friend request"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Friend request sent successfully"})
	}
}
*/
