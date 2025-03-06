package controllers

import (
	"Nogler/middleware"
	models "Nogler/models/postgres"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetAllReceivedFriendshipRequests godoc
// @Summary Get all friendship requests for the authenticated user
// @Description Retrieve all friendship requests where the authenticated user is the recipient. Each request includes the sender's public information: username and icon.
// @Tags Friendship
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "friendship_requests"
// @Failure 401 {object} map[string]string "error: User not authenticated"
// @Failure 404 {object} map[string]string "error: User not found"
// @Failure 500 {object} map[string]string "error: Error retrieving friendship requests"
// @Router /auth/friendship-requests [get]
// @Security ApiKeyAuth
func GetAllReceivedFriendshipRequests(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtaining user mail from JWT
		email, err := middleware.JWT_decoder(c)
		if err != nil {
			c.Abort()
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving received friendship requests"})
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
		c.JSON(http.StatusOK, gin.H{"received_friendship_requests": requestsInfo})
	}
}

// GetAllSentFriendshipRequests godoc
// @Summary Get all friendship requests sent by the authenticated user
// @Description Retrieve all friendship requests where the authenticated user is the sender. Each request includes the recipient's public information: username and icon.
// @Tags Friendship
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "sent_friendship_requests"
// @Failure 401 {object} map[string]string "error: User not authenticated"
// @Failure 404 {object} map[string]string "error: User not found"
// @Failure 500 {object} map[string]string "error: Error retrieving friendship requests"
// @Router /auth/sent-friendship-requests [get]
// @Security ApiKeyAuth
func GetAllSentFriendshipRequests(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtaining user mail from JWT
		email, err := middleware.JWT_decoder(c)
		if err != nil {
			c.Abort()
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
		if err := db.Where("sender = ?", user.ProfileUsername).Find(&friendRequests).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving sent friendship requests"})
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
		c.JSON(http.StatusOK, gin.H{"sent_friendship_requests": requestsInfo})
	}
}

// GetAllReceivedGameLobbyInvitations godoc
// @Summary Get all game lobby invitations for the authenticated user
// @Description Retrieve all game lobby invitations where the authenticated user is the recipient. Each invitation includes the sender's public information: username, icon, and the lobby ID.
// @Tags GameLobby
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "game_lobby_invitations"
// @Failure 401 {object} map[string]string "error: User not authenticated"
// @Failure 404 {object} map[string]string "error: User not found"
// @Failure 500 {object} map[string]string "error: Error retrieving game lobby invitations"
// @Router /auth/game-lobby-invitations [get]
// @Security ApiKeyAuth
func GetAllReceivedGameLobbyInvitations(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtaining user mail from JWT
		email, err := middleware.JWT_decoder(c)
		if err != nil {
			c.Abort()
			return
		}

		// Find the user's game profile
		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Retrieve all game lobby invitations where the user is the recipient
		// Preload the GameLobby relationship
		var gameInvitations []models.GameInvitation
		if err := db.Preload("GameLobby").Where("invited_username = ?", user.ProfileUsername).Find(&gameInvitations).Error; err != nil {
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
		c.JSON(http.StatusOK, gin.H{"received_game_lobby_invitations": invitationsInfo})
	}
}

// GetAllSentGameLobbyInvitations godoc
// @Summary Get all game lobby invitations sent by the authenticated user
// @Description Retrieve all game lobby invitations where the authenticated user is the sender. Each invitation includes the recipient's public information: username, icon, and the lobby ID.
// @Tags GameLobby
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "sent_game_lobby_invitations"
// @Failure 401 {object} map[string]string "error: User not authenticated"
// @Failure 404 {object} map[string]string "error: User not found"
// @Failure 500 {object} map[string]string "error: Error retrieving game lobby invitations"
// @Router /auth/sent-game-lobby-invitations [get]
// @Security ApiKeyAuth
func GetAllSentGameLobbyInvitations(db *gorm.DB) gin.HandlerFunc {
	// TODO: add more checks, user in a game, etc? Ask Victor?
	return func(c *gin.Context) {
		// Obtaining user mail from JWT
		email, err := middleware.JWT_decoder(c)
		if err != nil {
			c.Abort()
			return
		}

		// Find the user's game profile
		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Retrieve all game lobby invitations where the user is the recipient
		// Preload the GameLobby relationship
		var gameInvitations []models.GameInvitation
		if err := db.Preload("GameLobby").Where("sender_username = ?", user.ProfileUsername).Find(&gameInvitations).Error; err != nil {
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
		c.JSON(http.StatusOK, gin.H{"sent_game_lobby_invitations": invitationsInfo})
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
// @Router /auth/friendship-requests/{username} [delete]
// @Security ApiKeyAuth
func DeleteSentFriendshipRequest(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtaining user mail from JWT
		email, err := middleware.JWT_decoder(c)
		if err != nil {
			c.Abort()
			return
		}

		// Buscar el perfil de juego del usuario
		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Obtener el nombre de usuario del parámetro de la URL
		username := c.Param("username")

		// Eliminar la solicitud de amistad
		result := db.Where("sender = ? AND recipient = ?", user.ProfileUsername, username).Delete(&models.FriendshipRequest{})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting friendship request"})
			return
		}

		// Verificar si la solicitud se eliminó realmente
		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Friendship request not found"})
			return
		}

		// Devolver mensaje de éxito
		c.JSON(http.StatusOK, gin.H{"message": "Friendship request deleted successfully"})
	}
}

// DeleteReceivedFriendshipRequest godoc
// @Summary Delete a friendship request received by the authenticated user
// @Description Delete a friendship request where the authenticated user is the recipient and the specified username is the sender.
// @Tags Friendship
// @Accept json
// @Produce json
// @Param username path string true "Sender's username"
// @Success 200 {object} map[string]string "message: Friendship request deleted successfully"
// @Failure 401 {object} map[string]string "error: User not authenticated"
// @Failure 404 {object} map[string]string "error: Friendship request not found"
// @Failure 500 {object} map[string]string "error: Error deleting friendship request"
// @Router /auth/received-friendship-requests/{username} [delete]
// @Security ApiKeyAuth
func DeleteReceivedFriendshipRequest(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtaining user mail from JWT
		email, err := middleware.JWT_decoder(c)
		if err != nil {
			c.Abort()
			return
		}

		// Buscar el perfil de juego del usuario
		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Obtener el nombre de usuario del parámetro de la URL
		senderUsername := c.Param("username")

		// Eliminar la solicitud de amistad donde el usuario es el receptor y el nombre de usuario especificado es el emisor
		result := db.Where("sender = ? AND recipient = ?", senderUsername, user.ProfileUsername).Delete(&models.FriendshipRequest{})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting friendship request"})
			return
		}

		// Verificar si la solicitud se eliminó realmente
		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Friendship request not found"})
			return
		}

		// Devolver mensaje de éxito
		c.JSON(http.StatusOK, gin.H{"message": "Friendship request deleted successfully"})
	}
}

// DeleteReceivedGameLobbyInvitation godoc
// @Summary Delete a game lobby invitation received by the authenticated user
// @Description Delete a game lobby invitation where the authenticated user is the recipient and the specified lobby ID and sender username are the targets.
// @Tags GameLobby
// @Accept json
// @Produce json
// @Param lobby_id path string true "Lobby ID"
// @Param username path string true "Sender's username"
// @Success 200 {object} map[string]string "message: Game lobby invitation deleted successfully"
// @Failure 401 {object} map[string]string "error: User not authenticated"
// @Failure 404 {object} map[string]string "error: Game lobby invitation not found"
// @Failure 500 {object} map[string]string "error: Error deleting game lobby invitation"
// @Router /auth/received_lobby_invitation/{lobby_id}/{username} [delete]
// @Security ApiKeyAuth
func DeleteReceivedGameLobbyInvitation(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtaining user mail from JWT
		email, err := middleware.JWT_decoder(c)
		if err != nil {
			c.Abort()
			return
		}

		// Find the user's game profile
		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Get the lobby ID and sender username from the URL parameters
		lobbyID := c.Param("lobby_id")
		senderUsername := c.Param("username")

		// Delete the game invitation where the user is the recipient, and the specified lobby ID and sender are the targets
		result := db.Where("lobby_id = ? AND sender_username = ? AND invited_username = ?",
			lobbyID, senderUsername, user.ProfileUsername).Delete(&models.GameInvitation{})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting game lobby invitation"})
			return
		}

		// Check if the invitation was actually deleted
		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Game lobby invitation not found"})
			return
		}

		// Return success message
		c.JSON(http.StatusOK, gin.H{"message": "Game lobby invitation deleted successfully"})
	}
}

// DeleteSentGameLobbyInvitation godoc
// @Summary Delete a game lobby invitation sent by the authenticated user
// @Description Delete a game lobby invitation where the authenticated user is the sender and the specified lobby ID and recipient username are the targets.
// @Tags GameLobby
// @Accept json
// @Produce json
// @Param lobby_id path string true "Lobby ID"
// @Param username path string true "Recipient's username"
// @Success 200 {object} map[string]string "message: Game lobby invitation deleted successfully"
// @Failure 401 {object} map[string]string "error: User not authenticated"
// @Failure 404 {object} map[string]string "error: Game lobby invitation not found"
// @Failure 500 {object} map[string]string "error: Error deleting game lobby invitation"
// @Router /auth/sent_lobby_invitation/{lobby_id}/{username} [delete]
// @Security ApiKeyAuth
func DeleteSentGameLobbyInvitation(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtaining user mail from JWT
		email, err := middleware.JWT_decoder(c)
		if err != nil {
			c.Abort()
			return
		}

		// Find the user's game profile
		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Get the lobby ID and recipient username from the URL parameters
		lobbyID := c.Param("lobby_id")
		recipientUsername := c.Param("username")

		// Delete the game invitation where the user is the sender and the specified lobby ID and recipient are the targets
		result := db.Where("lobby_id = ? AND sender_username = ? AND invited_username = ?",
			lobbyID, user.ProfileUsername, recipientUsername).Delete(&models.GameInvitation{})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting game lobby invitation"})
			return
		}

		// Check if the invitation was actually deleted
		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Game lobby invitation not found"})
			return
		}

		// Return success message
		c.JSON(http.StatusOK, gin.H{"message": "Game lobby invitation deleted successfully"})
	}
}
