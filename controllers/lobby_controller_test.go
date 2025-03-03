package controllers

import (
	"Nogler/redis"
	"Nogler/services/sync"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetLobbyInfo(t *testing.T) {
	// Setup test environment
	gin.SetMode(gin.TestMode)
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// Create controller with mocked dependencies
	lobbyController := &LobbyController{
		DB:          db,
		RedisClient: &redis.RedisClient{},
		SyncManager: &sync.SyncManager{},
	}

	// Setup router
	router := gin.New()
	router.GET("/lobby/:codigo", lobbyController.GetLobbyInfo)

	// Setup mock expectations with minimal verbosity
	fmt.Println("Request: GET /lobby/test123")

	mock.ExpectQuery(`SELECT id, creator_username, is_started FROM game_lobbies WHERE id = \$1 AND is_started = false`).
		WithArgs("test123").
		WillReturnRows(sqlmock.NewRows([]string{"id", "creator_username", "is_started"}).
			AddRow("test123", "testuser", false))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM in_game_players WHERE lobby_id = \$1`).
		WithArgs("test123").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	mock.ExpectQuery(`SELECT user_icon FROM game_profiles WHERE username = \$1`).
		WithArgs("testuser").
		WillReturnRows(sqlmock.NewRows([]string{"user_icon"}).AddRow("5"))

	// Create HTTP request
	req, _ := http.NewRequest("GET", "/lobby/test123", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	fmt.Println("Response:", w.Body.String())

	// Verify response fields
	assert.Equal(t, "test123", response["code"])
	assert.Equal(t, "testuser", response["host_name"])
	assert.Equal(t, "5", response["host_icon"])
	assert.Equal(t, float64(3), response["player_count"])

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetLobbyInfoNotFound(t *testing.T) {
	// Setup test environment
	gin.SetMode(gin.TestMode)
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// Create controller with mocked dependencies
	lobbyController := &LobbyController{
		DB:          db,
		RedisClient: &redis.RedisClient{},
		SyncManager: &sync.SyncManager{},
	}

	// Setup router
	router := gin.New()
	router.GET("/lobby/:codigo", lobbyController.GetLobbyInfo)

	// Setup mock expectations for not found case
	fmt.Println("Request: GET /lobby/nonexistent")

	mock.ExpectQuery(`SELECT id, creator_username, is_started FROM game_lobbies WHERE id = \$1 AND is_started = false`).
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	// Create HTTP request
	req, _ := http.NewRequest("GET", "/lobby/nonexistent", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Verify response
	fmt.Println("Response:", w.Body.String())
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
