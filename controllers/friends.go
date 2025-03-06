package controllers

import (
	"Nogler/models/postgres"
	// "errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"gorm.io/gorm"
)

// @Summary Get a list of a user friends
// @Description Returns a list of the user's friends
// @Tags friends
// @Produce json
// @Success 200 {array} object{username=string,icon=integer}
// @Failure 500 {object} object{error=string}
// @Router /auth/friends [get]
func ListFriends(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		// TODO: we should get the username from the session instead of the query
		username := c.Param("username")

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
// @Accept json
// @Produce json
// @Param username path string true "Username of the user sending the friend request"
// @Param friendUsername query string true "Username of the friend to be added"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /auth/addFriend [post]
func AddFriend(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Get the username of the requester
		// TODO: we should get the username from the session instead of the query
		// With JWT username is NOT in the session, only email
		username := c.PostForm("username")
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
			"(sender = ? AND recipient = ?) OR (sender = ? AND recipient = ?)",
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

// @Summary Remove a friend
// @Description Removes a friend from the user's friend list
// @Tags friends
// @Accept json
// @Produce json
// @Param username path string true "Username of the user removing the friend"
// @Param friendUsername query string true "Username of the friend to be removed"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /auth/deleteFriend [delete]
func DeleteFriend(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract usernames
		username := c.Param("username")
		friendUsername := c.PostForm("friendUsername")

		if username == "" || friendUsername == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Both usernames are required"})
			return
		}

		// Check if the friendship exists
		var friendship postgres.Friendship
		result := db.Where(
			"(sender = ? AND recipient = ?) OR (sender = ? AND recipient = ?)",
			username, friendUsername, friendUsername, username,
		).First(&friendship)

		if result.RowsAffected == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Friendship does not exist"})
			return
		}

		// Delete the friendship
		result = db.Delete(&friendship)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting friend"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Friend removed successfully"})
	}
}

// @Summary Send a friend request
// @Description Sends a friend request from the sender to another user
// @Tags friends
// @Accept json
// @Produce json
// @Param username path string true "Username of the sender"
// @Param friendUsername query string true "Username of the recipient"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /auth/sendFriendRequest [post]
func SendFriendRequest(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract sender's username
		senderUsername := c.Param("username")
		// Extract recipient's username
		receiverUsername := c.PostForm("friendUsername")

		if senderUsername == "" || receiverUsername == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Both usernames are required"})
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
			"(sender = ? AND recipient = ?) OR (sender = ? AND recipient = ?)",
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
