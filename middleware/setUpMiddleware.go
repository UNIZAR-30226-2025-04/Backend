package middleware

import (
	"log"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func SetUpMiddleware(r *gin.Engine) {
	key := os.Getenv("KEY")
	log.Println(key)
	store := cookie.NewStore([]byte(key))
	r.Use(sessions.Sessions("mysession", store))
}
