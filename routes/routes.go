package routes

import (
	"Nogler/controllers"
	"Nogler/middleware"
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
	// @Failure 500 {object} object{error=string}
	// @Router /allusers [get]
	api.GET("/allusers", controllers.GetAllUsers(db))

	// @Summary Get user public info
	// @Description Returns public information about a specific user (username and icon)
	// @Tags users
	// @Produce json
	// @Param username path string true "Username"
	// @Success 200 {object} object{username=string,icon=integer}
	// @Failure 400 {object} object{error=string}
	// @Failure 404 {object} object{error=string}
	// @Failure 500 {object} object{error=string}
	// @Router /users/{username} [get]
	api.GET("/users/:username", controllers.GetUserPublicInfo(db))

	// @Summary Login user
	// @Description Authenticates a user and creates a session
	// @Tags auth
	// @Accept x-www-form-urlencoded
	// @Produce json
	// @Param email formData string true "User email"
	// @Param password formData string true "User password"
	// @Success 200 {object} object{message=string}
	// @Failure 400 {object} object{error=string}
	// @Failure 401 {object} object{error=string}
	// @Router /login [post]
	api.POST("/login", controllers.Login(db))

	// @Summary Sign up a new user
	// @Description Creates a new user account
	// @Tags auth
	// @Accept x-www-form-urlencoded
	// @Produce json
	// @Param username formData string true "Username"
	// @Param email formData string true "Email"
	// @Param password formData string true "Password"
	// @Param icono formData string true "Icon number"
	// @Success 201 {object} object{message=string,user=object{username=string,email=string}}
	// @Failure 400 {object} object{error=string}
	// @Failure 409 {object} object{error=string}
	// @Failure 500 {object} object{error=string}
	// @Router /signup [post]
	api.POST("/signup", controllers.SignUp(db))

	authentication := api.Group("/auth")
	authentication.Use(middleware.AuthRequired)
	{
		// @Summary Log out a user
		// @Description Ends the user's session
		// @Tags auth
		// @Produce json
		// @Success 200 {object} object{message=string}
		// @Failure 400 {object} object{error=string}
		// @Failure 500 {object} object{error=string}
		// @Router /auth/logout [delete]
		authentication.DELETE("/logout", controllers.Logout)

		// @Summary Get user private info
		// @Description Returns private information about the authenticated user
		// @Tags users
		// @Produce json
		// @Security ApiKeyAuth
		// @Success 200 {object} object{username=string,email=string,icon=integer}
		// @Failure 401 {object} object{error=string}
		// @Failure 404 {object} object{error=string}
		// @Failure 500 {object} object{error=string}
		// @Router /auth/me [get]
		authentication.GET("/me", controllers.GetUserPrivateInfo(db))

		// @Summary Update user information
		// @Description Updates the authenticated user's information
		// @Tags users
		// @Accept x-www-form-urlencoded
		// @Produce json
		// @Param username formData string false "New username"
		// @Param email formData string false "New email"
		// @Param password formData string false "New password"
		// @Param icono formData string false "New icon number"
		// @Success 200 {object} object{message=string,user=object{username=string,email=string,icon=integer}}
		// @Failure 400 {object} object{error=string}
		// @Failure 401 {object} object{error=string}
		// @Failure 404 {object} object{error=string}
		// @Failure 409 {object} object{error=string}
		// @Failure 500 {object} object{error=string}
		// @Router /auth/update [patch]
		authentication.PATCH("/update", controllers.UpdateUserInfo(db))
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
