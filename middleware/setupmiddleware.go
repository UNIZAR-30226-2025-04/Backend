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

	r.Use(func(c *gin.Context) {
		c.Next()

		cookies := c.Writer.Header().Values("Set-Cookie")
		for i, cookie := range cookies {
			if !containsPartitioned(cookie) {
				cookies[i] += "; Partitioned"
			}
		}

		c.Writer.Header().Set("Set-Cookie", "")
		for _, cookie := range cookies {
			c.Writer.Header().Add("Set-Cookie", cookie)
		}
	})

	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		AllowMethods:     []string{"GET", "PUT", "PATCH", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
	}))
}

func containsPartitioned(cookie string) bool {
	return len(cookie) > 0 && (len(cookie) >= 12 && cookie[len(cookie)-12:] == "Partitioned")
}
