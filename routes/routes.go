package routes

import (
	"Nogler/controllers"
	"Nogler/services/redis"
	utils "Nogler/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

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
func SetupRoutes(router *gin.Engine, db *gorm.DB, redisClient *redis.RedisClient) {
	// Create SyncManager instance
	// syncManager := sync.NewSyncManager(redisClient, db)

	// Create controllers
	// lobbyController := &controllers.LobbyController{DB: db, RedisClient: redisClient, SyncManager: syncManager}

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

	// @Summary Get all users
	// @Description Returns a list of all users with their usernames and icons
	// @Tags users
	// @Produce json
	// @Success 200 {array} object{username=string,icon=integer}
	// @Router /allusers [get]
	api.GET("/allusers", controllers.GetAllUsers(db))

	authentication := api.Group("/auth")
	{
		// @Summary Manages user login
		// @Description Checks user inputs validity and returns confirmation or denial
		// @Tags auth
		// @Produce json
		// @Success 200 {object} string
		// @Router /auth/login [post]
		authentication.POST("/login", controllers.Login(db))

		// @Summary Sign up a new user
		// @Description TODO: COMPLETE
		// @Tags auth
		// @Produce json
		// @Success 200 {object} string
		// @Router /auth/signup [postdiff]
		authentication.POST("/signup", controllers.SignUp(db))

		// @Summary Log out a user from the session
		// @Description TODO: COMPLETE
		// @Tags auth
		// @Produce json
		// @Success 200 {object} string
		// @Router /auth/logout [delete]
		authentication.DELETE("/logout", controllers.Logout)
	}

	// Routes that require authentication
	//authenticated := api.Group("/")
	{
		// Lobby routes TODO: Documentar
		//lobby := authenticated.Group("/lobby")
		{
			//lobby.GET("/:codigo", lobbyController.GetLobbyInfo)
		}
	}
}
