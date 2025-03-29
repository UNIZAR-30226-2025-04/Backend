package utils

import (
	"Nogler/models/postgres"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
)

// LoggerMiddleware logs information about each request
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start time
		startTime := time.Now()

		// Process request
		c.Next()

		// End time
		endTime := time.Now()

		// Calculate latency
		latency := endTime.Sub(startTime)

		// Get request details
		method := c.Request.Method
		path := c.Request.URL.Path
		statusCode := c.Writer.Status()

		// Register information
		c.JSON(http.StatusOK, gin.H{
			"status":  statusCode,
			"latency": latency,
			"method":  method,
			"path":    path,
		})
	}
}

// ErrorHandler handles global errors
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// Function to check if a lobby exists
func CheckLobbyExists(db *gorm.DB, lobbyID string) (*postgres.GameLobby, error) {
	var lobby postgres.GameLobby
	result := db.Where("id = ?", lobbyID).First(&lobby)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("lobby not found")
		}
		return nil, result.Error
	}

	return &lobby, nil
}

func IsPlayerInLobby(db *gorm.DB, lobbyID string, username string) (bool, error) {
	var count int64
	err := db.Model(&postgres.InGamePlayer{}).
		Where("lobby_id = ? AND username = ?", lobbyID, username).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// Check if user is in lobby
func UserExistsInLobby(db *gorm.DB, lobbyID string, username string, client *socket.Socket) (inLobby bool, e error) {
	isInLobby, err := IsPlayerInLobby(db, lobbyID, username)
	if err != nil {
		fmt.Println("Database error:", err)
		client.Emit("error", gin.H{"error": "Database error"})
	}
	return isInLobby, err
}

// Returns the icon of the user
func UserIcon(db *gorm.DB, username string) int {
	var icon int
	err := db.Model(&postgres.GameProfile{}).
		Select("user_icon").
		Where("username = ?", username).
		Find(&icon).Error
	if err != nil {
		return 1
	}

	return icon
}
