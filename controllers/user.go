package controllers

import (
	"Nogler/constants/auth"
	"Nogler/middleware"
	models "Nogler/models/postgres"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// @title Nogler API
// @version 1.0
// @description Gin-Gonic server for the "Nogler" game API
// @host https://nogler.ddns.net:443
// @BasePath /
// @paths

// @Summary Login user
// @Description Authenticates a user and creates a session
// @Tags auth
// @Accept x-www-form-urlencoded
// @Produce json
// @Param email formData string true "User email"
// @Param password formData string true "User password"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Router /login [post]
func Login(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		// Generate JWT
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			auth.Email: user.Email,
		})

		secret := os.Getenv("KEY")
		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating JWT"})
		}

		c.JSON(http.StatusOK, gin.H{"message": "Successfully logged in.", "token": tokenString})
	}
}

// @Summary Log out a user
// @Description Ends the user's session
// @Tags auth
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /auth/logout [delete]
func Logout(c *gin.Context) {
	//This serves no purpose with JWT so TODO rething
	_, err := middleware.JWT_decoder(c)
	if err != nil {
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

// @Summary Sign up a new user
// @Description Creates a new user account
// @Tags auth
// @Accept x-www-form-urlencoded
// @Produce json
// @Param username formData string true "Username"
// @Param email formData string true "Email"
// @Param password formData string true "Password"
// @Param icono formData string true "Icon number"
// @Success 201 {object} object{message=string,user=object{username=string,email=string}}
// @Failure 400 {object} object{error=string}
// @Failure 409 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /signup [post]
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
			// TODO: do it with a transaction?
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

// @Summary Get all users
// @Description Returns a list of all users with their usernames and icons
// @Tags users
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @Success 200 {array} object{username=string,icon=integer}
// @Failure 500 {object} object{error=string}
// @Router /allusers [get]
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

// @Summary Get user public info
// @Description Returns public information about a specific user (username and icon)
// @Tags users
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @Param username path string true "Username"
// @Success 200 {object} object{username=string,icon=integer}
// @Failure 400 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /users/{username} [get]
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

// @Summary Get user private info
// @Description Returns private information about the authenticated user
// @Tags users
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @Success 200 {object} object{username=string,email=string,icon=integer}
// @Failure 401 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /auth/me [get]
func GetUserPrivateInfo(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get email from session
		email, err := middleware.JWT_decoder(c)
		if err != nil {
			c.Abort()
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

// @Summary Update user information
// @Description Updates the authenticated user's information
// @Tags users
// @Accept x-www-form-urlencoded
// @Produce json
// @Param Authorization header string true "Bearer JWT token"
// @Param username formData string false "New username"
// @Param email formData string false "New email"
// @Param password formData string false "New password"
// @Param icon formData string false "New icon number"
// @Success 200 {object} object{message=string,user=object{username=string,email=string,icon=integer}}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 409 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /auth/update [patch]
func UpdateUserInfo(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get email from session
		currentEmail, err := middleware.JWT_decoder(c)
		if err != nil {
			c.Abort()
			return
		}

		// Get update data from request
		username := c.PostForm("username")
		email := c.PostForm("email")
		password := c.PostForm("password")
		icon := c.PostForm("icon")

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
			// Check if new username is already taken
			var existingUser models.User
			if err := tx.Where("profile_username = ? AND email != ?", username, currentEmail).First(&existingUser).Error; err == nil {
				tx.Rollback()
				c.JSON(http.StatusConflict, gin.H{"error": "Username already taken"})
				return
			}
			
			// With ON UPDATE CASCADE constraints properly set up, we can simply update the username
			// in the game_profiles table, and all related records will be updated automatically
			
			// First, update the game profile's username (primary key)
			// NOTE: with raw GORM, it can be problematic
			if err := tx.Exec("UPDATE game_profiles SET username = ? WHERE username = ?", 
							 username, user.ProfileUsername).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update game profile username"})
				return
			}
			
			// Then update the user's profile_username field
			user.ProfileUsername = username
			
			// Save user changes
			if err := tx.Save(&user).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
				return
			}
			
			// Update our local gameProfile variable to reflect the change
			gameProfile.Username = username
		}

		// Check if new email is already taken (if changing email)
		if email != "" && email != currentEmail {
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

		var tokenString string
		// Update email if provided
		if email != "" && email != currentEmail {
			user.Email = email
			// Generate JWT
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				auth.Email: user.Email,
			})

			secret := os.Getenv("KEY")
			tokenString, err = token.SignedString([]byte(secret))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating JWT"})
			}
		} else {
			// NEW: better not return an empty string, even if the user didn't change his email
			authHeader := c.GetHeader("Authorization")
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
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

		// Save user changes if we didn't already save them above
		if err := tx.Save(&user).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
			return
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
			"token": tokenString,
		})
	}
}
