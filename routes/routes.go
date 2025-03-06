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

		authentication.GET("/received_friendship_requests", controllers.GetAllReceivedFriendshipRequests(db))

		authentication.GET("/sent_friendship_requests", controllers.GetAllSentFriendshipRequests(db))

		authentication.GET("/received_lobby_invitations", controllers.GetAllReceivedGameLobbyInvitations(db))

		authentication.GET("/sent_lobby_invitations", controllers.GetAllSentGameLobbyInvitations(db))

		authentication.DELETE("/received_friendship_request/:username", controllers.DeleteReceivedFriendshipRequest(db))

		authentication.DELETE("/sent_friendship_request/:username", controllers.DeleteSentFriendshipRequest(db))

		authentication.DELETE("/received_lobby_invitation/:lobby_id/:username", controllers.DeleteReceivedGameLobbyInvitation(db))

		authentication.DELETE("/sent_lobby_invitation/:lobby_id/:username", controllers.DeleteSentGameLobbyInvitation(db))

		authentication.GET("/friends/:username", controllers.ListFriends(db))

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
