package routes

import (
	"Nogler/controllers"
	"Nogler/redis"
	"Nogler/services/sync"
	utils "Nogler/utils"
	"database/sql"

	"github.com/gin-gonic/gin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Nogler API
// @version 1.0
// @description Gin-Gonic server for the "Nogler" game API
// @host 74.234.191.199:8080
// @BasePath /
// @paths

// SetupRoutes configures all API routes
func SetupRoutes(router *gin.Engine, db *sql.DB, redisClient *redis.RedisClient) {
	// Create SyncManager instance
	syncManager := sync.NewSyncManager(redisClient, db)

	// Create controllers
	lobbyController := &controllers.LobbyController{DB: db, RedisClient: redisClient, SyncManager: syncManager}

	// utils global
	router.Use(utils.ErrorHandler())

	// Swagger route
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Testing a basic endpoint, and the auto-docs

	// @Summary Endpoint just pings the server
	// @Description Returns a basic message
	// @Tags test
	// @Produce json
	// @Success 200 {object} string
	// @Router /ping [get]
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong, hola"})
	})

	// API routes group
	api := router.Group("/")

	authentication := api.Group("/auth")
	{
		api.POST("/login", user.login)
		api.POST("/signup", user.signup)
		api.POST("/logout", user.logout)
	}

	// Routes that require authentication
	authenticated := api.Group("/")
	{
		// Lobby routes TODO: Documentar
		lobby := authenticated.Group("/lobby")
		{
			lobby.GET("/:codigo", lobbyController.GetLobbyInfo)
		}
	}
}
