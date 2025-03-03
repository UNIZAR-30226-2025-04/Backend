package controllers

import (
	"Nogler/redis"
	"Nogler/services/sync"
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

type LobbyController struct {
	DB          *sql.DB
	RedisClient *redis.RedisClient
	SyncManager *sync.SyncManager
}

// GetLobbyInfo gets information about a lobby with the provided code
func (lc *LobbyController) GetLobbyInfo(c *gin.Context) {
	codigo := c.Param("codigo")

	// Query basic lobby information in the database
	var lobby_psql struct {
		Code            string `json:"code"`
		CreatorUsername string `json:"host_name"`
		IsStarted       bool   `json:"is_started"`
	}

	err := lc.DB.QueryRow(`
		SELECT id, creator_username, is_started
		FROM game_lobbies
		WHERE id = $1 AND is_started = false
	`, codigo).Scan(
		&lobby_psql.Code, &lobby_psql.CreatorUsername, &lobby_psql.IsStarted,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Lobby not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error querying database: " + err.Error()})
		}
		return
	}

	// Query how many players are in the database
	var playerCount int
	err = lc.DB.QueryRow(`
		SELECT COUNT(*) 
		FROM in_game_players 
		WHERE lobby_id = $1
	`, codigo).Scan(&playerCount)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error counting players: " + err.Error()})
		return
	}

	// Get the host icon - Modified to use the game_profiles table directly
	var hostIcon string
	err = lc.DB.QueryRow(`
		SELECT user_icon
		FROM game_profiles
		WHERE username = $1
	`, lobby_psql.CreatorUsername).Scan(&hostIcon)

	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting host icon: " + err.Error()})
		return
	}

	// If no icon is found, use a default value
	if err == sql.ErrNoRows || hostIcon == "" {
		hostIcon = "5" // Default value
	}

	// Response structure
	response := gin.H{
		"code":         lobby_psql.Code,
		"host_name":    lobby_psql.CreatorUsername,
		"host_icon":    hostIcon,
		"player_count": playerCount,
	}

	c.JSON(http.StatusOK, response)
}
