package controllers

import (
	models "Nogler/models/postgres"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Login(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		email := c.PostForm("email")
		password := c.PostForm("password")

		//Minimum input sanitizing
		if strings.Trim(email, " ") == "" || strings.Trim(password, " ") == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parameters can't be empty"})
			return
		}

		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password!"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password!"})
			return
		}

		session.Set("Email", user.Email)
		if err := session.Save(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No session!"})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	}
}

// Logout from server, deletes the session associated with the Email key
func Logout(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("Email")
	// There is no session for the user, won't delete nothing
	if user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session token"})
		return
	}

	// Deletes the session associated with that userkey
	session.Delete("Email")
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func SignUp(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) { // TODO: completar
		fmt.Print("test")
	}
}
