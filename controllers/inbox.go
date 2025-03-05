package controllers

import (
	"Nogler/constants/auth"
	models "Nogler/models/postgres"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetAllFriendshipRequests godoc
// @Summary Get all friendship requests for the authenticated user
// @Description Retrieve all friendship requests where the authenticated user is the recipient. Each request includes the sender's public information: username and icon.
// @Tags Friendship
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "friendship_requests"
// @Failure 401 {object} map[string]string "error: User not authenticated"
// @Failure 404 {object} map[string]string "error: User not found"
// @Failure 500 {object} map[string]string "error: Error retrieving friendship requests"
// @Router /friendship-requests [get]
// @Security ApiKeyAuth
func GetAllFriendshipRequests(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtener el email del usuario desde el token
		session := sessions.Default(c)
		email := session.Get(auth.Email)
		if email == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		// Buscar el perfil de juego del usuario
		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Obtener todas las solicitudes de amistad donde el usuario es el receptor
		var friendRequests []models.FriendshipRequest
		if err := db.Where("recipient = ?", user.ProfileUsername).Find(&friendRequests).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving friendship requests"})
			return
		}

		// Recopilar la información pública del emisor
		var requestsInfo []gin.H
		for _, request := range friendRequests {
			var gameProfile models.GameProfile
			if err := db.Where("username = ?", request.Sender).First(&gameProfile).Error; err != nil {
				continue
			}
			requestsInfo = append(requestsInfo, gin.H{
				"username": gameProfile.Username,
				"icon":     gameProfile.UserIcon,
			})
		}

		// Devolver la información de las solicitudes de amistad
		c.JSON(http.StatusOK, gin.H{"friendship_requests": requestsInfo})
	}
}

// GetAllGameLobbyInvitations godoc
// @Summary Get all game lobby invitations for the authenticated user
// @Description Retrieve all game lobby invitations where the authenticated user is the recipient. Each invitation includes the sender's public information: username, icon, and the lobby ID.
// @Tags GameLobby
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "game_lobby_invitations"
// @Failure 401 {object} map[string]string "error: User not authenticated"
// @Failure 404 {object} map[string]string "error: User not found"
// @Failure 500 {object} map[string]string "error: Error retrieving game lobby invitations"
// @Router /game-lobby-invitations [get]
// @Security ApiKeyAuth
func GetAllGameLobbyInvitations(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtain the user's email from the token
		session := sessions.Default(c)
		email := session.Get(auth.Email)
		if email == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		// Find the user's game profile
		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Retrieve all game lobby invitations where the user is the recipient
		var gameInvitations []models.GameInvitation
		if err := db.Where("invited_username = ?", user.ProfileUsername).Find(&gameInvitations).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving game lobby invitations"})
			return
		}

		// Collect the public information of the sender and the lobby ID
		var invitationsInfo []gin.H
		for _, invitation := range gameInvitations {
			var gameProfile models.GameProfile
			if err := db.Where("username = ?", invitation.GameLobby.CreatorUsername).First(&gameProfile).Error; err != nil {
				continue
			}
			invitationsInfo = append(invitationsInfo, gin.H{
				"username": gameProfile.Username,
				"icon":     gameProfile.UserIcon,
				"lobby_id": invitation.LobbyID,
			})
		}

		// Return the information of the game lobby invitations
		c.JSON(http.StatusOK, gin.H{"game_lobby_invitations": invitationsInfo})
	}
}

// DeleteFriendshipRequest godoc
// @Summary Delete a friendship request from a user
// @Description Delete a friendship request where the authenticated user is the sender and the specified username is the recipient.
// @Tags Friendship
// @Accept json
// @Produce json
// @Param username path string true "Recipient's username"
// @Success 200 {object} map[string]string "message: Friendship request deleted successfully"
// @Failure 401 {object} map[string]string "error: User not authenticated"
// @Failure 404 {object} map[string]string "error: Friendship request not found"
// @Failure 500 {object} map[string]string "error: Error deleting friendship request"
// @Router /friendship-requests/{username} [delete]
// @Security ApiKeyAuth
func DeleteFriendshipRequest(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtain the user's email from the token
		session := sessions.Default(c)
		email := session.Get(auth.Email)
		if email == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		// Find the user's game profile
		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Get the recipient's username from the URL parameter
		username := c.Param("username")

		// Delete the friendship request
		if err := db.Where("sender = ? AND recipient = ?", user.ProfileUsername, username).Delete(&models.FriendshipRequest{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting friendship request"})
			return
		}

		// Check if the request was actually deleted
		if db.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Friendship request not found"})
			return
		}

		// Return success message
		c.JSON(http.StatusOK, gin.H{"message": "Friendship request deleted successfully"})
	}
}

// DeleteGameLobbyInvitation godoc
// @Summary Delete a game lobby invitation for the authenticated user
// @Description Delete a game lobby invitation where the authenticated user is the recipient and the specified username is the sender for a specific lobby code.
// @Tags GameLobby
// @Accept json
// @Produce json
// @Param username path string true "Sender's username"
// @Param code path string true "Lobby code"
// @Success 200 {object} map[string]string "message: Game lobby invitation deleted successfully"
// @Failure 401 {object} map[string]string "error: User not authenticated"
// @Failure 404 {object} map[string]string "error: Game lobby invitation not found"
// @Failure 500 {object} map[string]string "error: Error deleting game lobby invitation"
// @Router /game-lobby-invitations/{username}/{code} [delete]
// @Security ApiKeyAuth
func DeleteGameLobbyInvitation(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtain the user's email from the token
		session := sessions.Default(c)
		email := session.Get(auth.Email)
		if email == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		// Find the user's game profile
		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Get the sender's username and lobby code from the URL parameters
		username := c.Param("username")
		code := c.Param("code")

		// Delete the game lobby invitation
		if err := db.Where("invited_username = ? AND lobby_id = ? AND game_lobby.creator_username = ?", user.ProfileUsername, code, username).Delete(&models.GameInvitation{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting game lobby invitation"})
			return
		}

		// Check if the invitation was actually deleted
		if db.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Game lobby invitation not found"})
			return
		}

		// Return success message
		c.JSON(http.StatusOK, gin.H{"message": "Game lobby invitation deleted successfully"})
	}
}
