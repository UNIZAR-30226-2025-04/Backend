package middleware

import (
	"net/http"
	"os"
	"strings"

	"Nogler/constants/auth"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthRequired is a simple middleware to check the session.
func AuthRequired(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		c.Abort()
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
		c.Abort()
		return
	}

	secret := os.Getenv("KEY")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		c.Abort()
		return
	}
	c.Next()
}

func JWT_decoder(c *gin.Context) (string, error) {
	// Obtener el token del encabezado Authorization
	authHeader := c.GetHeader("Authorization")
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Parsear el JWT
	secret := os.Getenv("KEY")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	// Obtener los datos del token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		return "", err
	}

	email := claims[auth.Email].(string)

	return email, nil

}

// me is the handler that will return the user information stored in the
// session.
func me(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(auth.Email)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

// status is the handler that will tell the user whether it is logged in or not.
func status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "You are logged in"})
}
