package controllers

import (
	"Nogler/constants/auth"
	models "Nogler/models/postgres"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found: invalid email"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
			return
		}

		session.Set(auth.Email, user.Email)
		if err := session.Save(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No session!"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "bien hecho niga"})
	}
}

// Logout from server, deletes the session associated with the Email key
func Logout(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(auth.Email)
	// There is no session for the user, won't delete nothing
	if user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session token"})
		return
	}

	// Deletes the session associated with that userkey
	session.Delete(auth.Email)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func SignUp(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.PostForm("username")
		email := c.PostForm("email")
		password := c.PostForm("password")
		icon := c.PostForm("icono")

		// Minimum input sanitizing
		if strings.TrimSpace(username) == "" || strings.TrimSpace(email) == "" || strings.TrimSpace(password) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username, email and password are required"})
			return
		}

		// Check if user already exists
		var existingUser models.User
		if err := db.Where("email = ? OR profile_username = ?", email, username).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
			return
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
			return
		}

		// Convert icon string to int
		iconInt, err := strconv.Atoi(icon)
		if err != nil {
			iconInt = 0 // default icon if conversion fails
		}

		// Create GameProfile first
		gameProfile := models.GameProfile{
			Username:  username,
			UserStats: datatypes.JSON([]byte(`{"stats": "TBD"}`)),
			UserIcon:  iconInt,
			IsInAGame: false,
		}

		if err := db.Create(&gameProfile).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating game profile"})
			return
		}

		// Create User
		user := models.User{
			Email:           email,
			ProfileUsername: username,
			PasswordHash:    string(hashedPassword),
			MemberSince:     time.Now(),
		}

		if err := db.Create(&user).Error; err != nil {
			// Rollback game profile creation if user creation fails
			db.Delete(&gameProfile)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating user"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "User created successfully",
			"user": gin.H{
				"username": username,
				"email":    email,
			},
		})
	}
}

func GetAllUsers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var users []models.User

		// Preload GameProfile to get the icon
		result := db.Preload("GameProfile").Find(&users)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching users"})
			return
		}

		// Create a slice of simplified user data
		simplifiedUsers := make([]gin.H, len(users))
		for i, user := range users {
			simplifiedUsers[i] = gin.H{
				"username": user.ProfileUsername,
				"icon":     user.GameProfile.UserIcon,
			}
		}

		c.JSON(http.StatusOK, simplifiedUsers)
	}
}
