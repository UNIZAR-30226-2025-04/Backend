package utils

import (
	"net/http"
	"time"
	"gorm.io/gorm"
	"fmt"
	"Nogler/models/postgres"

	"github.com/gin-gonic/gin"
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
