package middleware

import (
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func SetUpMiddleware(r *gin.Engine) {
	key := os.Getenv("KEY")
	store := cookie.NewStore([]byte(key))
	store.Options(sessions.Options{
		Path:     "/",
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	})
	r.Use(sessions.Sessions("mysession", store))

	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		AllowMethods:     []string{"GET", "PUT", "PATCH", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
	}))
}
