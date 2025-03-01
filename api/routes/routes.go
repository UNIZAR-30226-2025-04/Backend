package routes

import (
	"Nogler/api/controllers"
	utils "Nogler/api/utils"
	"Nogler/redis"
	"Nogler/sync"
	"database/sql"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all API routes
func SetupRoutes(router *gin.Engine, db *sql.DB, redisClient *redis.RedisClient) {
	// Create SyncManager instance
	syncManager := sync.NewSyncManager(redisClient, db)

	// Create controllers
	lobbyController := &controllers.LobbyController{DB: db, RedisClient: redisClient, SyncManager: syncManager}

	// utils global
	router.Use(utils.ErrorHandler())

	// API routes group
	api := router.Group("/api/v1")

	// Routes that require authentication
	authenticated := api.Group("/")
	{
		// Lobby routes
		lobby := authenticated.Group("/lobby")
		{
			lobby.GET("/:codigo", lobbyController.GetLobbyInfo)
		}
	}
} 