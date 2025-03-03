package middleware

import (
	"Nogler/controllers"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

var secret = []byte("secret")

// AuthRequired is a simple middleware to check the session.
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(userkey)
	if user == nil {
		// Abort the request with the appropriate error code
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	// Continue down the chain to handler etc
	c.Next()
}

func engine() *gin.Engine {
	r := gin.New()

	// Setup the cookie store for session management
	r.Use(sessions.Sessions("mysession", cookie.NewStore(secret)))

	// Login and logout routes
	r.POST("/login", controllers.Login)
	r.GET("/logout", controllers.Logout)

	// Private group, require authentication to access
	private := r.Group("/private")
	private.Use(AuthRequired)
	{
		private.GET("/me", me)
		private.GET("/status", status)
	}
	return r
}

// me is the handler that will return the user information stored in the
// session.
func me(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(userkey)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

// status is the handler that will tell the user whether it is logged in or not.
func status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "You are logged in"})
}
