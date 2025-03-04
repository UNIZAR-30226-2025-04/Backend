package controllers

import (
	"Nogler/constants/auth"
	models "Nogler/models/postgres"
	"errors"
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

// GetUserPublicInfo returns public information about a specific user
func GetUserPublicInfo(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")

		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username is required"})
			return
		}

		var user models.User
		result := db.Where("profile_username = ?", username).Preload("GameProfile").First(&user)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user"})
			}
			return
		}

		// Return only public information
		publicInfo := gin.H{
			"username": user.ProfileUsername,
			"icon":     user.GameProfile.UserIcon,
		}

		c.JSON(http.StatusOK, publicInfo)
	}
}

// GetUserPrivateInfo returns private information about the authenticated user
func GetUserPrivateInfo(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get email from session
		session := sessions.Default(c)
		email := session.Get(auth.Email)

		if email == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		// Query the database for user information
		var user models.User
		result := db.Where("email = ?", email).Preload("GameProfile").First(&user)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user data"})
			}
			return
		}

		// Return private user information
		privateInfo := gin.H{
			"username": user.ProfileUsername,
			"email":    user.Email,
			"icon":     user.GameProfile.UserIcon,
		}

		c.JSON(http.StatusOK, privateInfo)
	}
}

// UpdateUserInfo updates the authenticated user's information
func UpdateUserInfo(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get email from session
		session := sessions.Default(c)
		currentEmail := session.Get(auth.Email)

		if currentEmail == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		// Get update data from request
		username := c.PostForm("username")
		email := c.PostForm("email")
		password := c.PostForm("password")
		icon := c.PostForm("icono")

		// Start a transaction
		tx := db.Begin()
		if tx.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}

		// Get current user
		var user models.User
		if err := tx.Where("email = ?", currentEmail).First(&user).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Get current game profile
		var gameProfile models.GameProfile
		if err := tx.Where("username = ?", user.ProfileUsername).First(&gameProfile).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{"error": "Game profile not found"})
			return
		}

		// Check if new username is already taken (if changing username)
		if username != "" && username != user.ProfileUsername {
			var existingUser models.User
			if err := tx.Where("profile_username = ? AND email != ?", username, currentEmail).First(&existingUser).Error; err == nil {
				tx.Rollback()
				c.JSON(http.StatusConflict, gin.H{"error": "Username already taken"})
				return
			}
		}

		// Check if new email is already taken (if changing email)
		if email != "" && email != currentEmail.(string) {
			var existingUser models.User
			if err := tx.Where("email = ?", email).First(&existingUser).Error; err == nil {
				tx.Rollback()
				c.JSON(http.StatusConflict, gin.H{"error": "Email already taken"})
				return
			}
		}

		// Update password if provided
		if password != "" {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
				return
			}
			user.PasswordHash = string(hashedPassword)
		}

		// Update email if provided
		if email != "" && email != currentEmail.(string) {
			user.Email = email
			// Update session with new email
			session.Set(auth.Email, email)
			if err := session.Save(); err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
				return
			}
		}

		// Update icon if provided
		if icon != "" {
			iconInt, err := strconv.Atoi(icon)
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid icon value"})
				return
			}
			gameProfile.UserIcon = iconInt

			// Save game profile changes for icon
			if err := tx.Save(&gameProfile).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update game profile"})
				return
			}
		}

		// Update username if provided - this needs special handling due to foreign key constraints
		if username != "" && username != user.ProfileUsername {
			// Create a new game profile with the new username
			newGameProfile := models.GameProfile{
				Username:  username,
				UserStats: gameProfile.UserStats,
				UserIcon:  gameProfile.UserIcon,
				IsInAGame: gameProfile.IsInAGame,
			}

			// Create the new profile first
			if err := tx.Create(&newGameProfile).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new game profile"})
				return
			}

			// Update user to point to the new profile
			user.ProfileUsername = username

			// Save user changes
			if err := tx.Save(&user).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
				return
			}

			// Now we can delete the old profile
			if err := tx.Delete(&gameProfile).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete old game profile"})
				return
			}

			// Update gameProfile reference for the response
			gameProfile = newGameProfile
		} else {
			// Save user changes if we didn't already save them above
			if err := tx.Save(&user).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
				return
			}
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit changes"})
			return
		}

		// Return updated user info
		c.JSON(http.StatusOK, gin.H{
			"message": "User updated successfully",
			"user": gin.H{
				"username": user.ProfileUsername,
				"email":    user.Email,
				"icon":     gameProfile.UserIcon,
			},
		})
	}
}
