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

	// API routes group
	api := router.Group("/")

	api.GET("/ping", controllers.Ping)

	api.GET("/allusers", controllers.GetAllUsers(db))

	api.GET("/users/:username", controllers.GetUserPublicInfo(db))

	api.POST("/login", controllers.Login(db))

	api.POST("/signup", controllers.SignUp(db))

	authentication := api.Group("/auth")
	authentication.Use(middleware.AuthRequired)
	{
		authentication.DELETE("/logout", controllers.Logout)

		authentication.GET("/me", controllers.GetUserPrivateInfo(db))

		authentication.PATCH("/update", controllers.UpdateUserInfo(db))

		authentication.GET("/friendship_requests", controllers.GetAllFriendshipRequests(db))

		authentication.GET("/lobby_invitations", controllers.GetAllGameLobbyInvitations(db))

		authentication.DELETE("/delete_friendship_request", controllers.DeleteFriendshipRequest(db))

		authentication.DELETE("/delete_game_lobby_invitation", controllers.DeleteGameLobbyInvitation(db))

		authentication.GET("/friends", controllers.ListFriends(db))

		authentication.POST("/addFriend", controllers.AddFriend(db))
	}

	// Routes that require authentication
	/*authenticated := api.Group("/")
	{
		lobby := authenticated.Group("/lobby")
		{
			lobby.GET("/:codigo", lobbyController.GetLobbyInfo)
		}

		friends := authenticated.Group("/friends")
		{
			api.POST("/list", controllers.ListFriends(db))
		}
	}*/
}
