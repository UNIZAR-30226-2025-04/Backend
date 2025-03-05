package postgres_test

import (
	config "Nogler/config/postgres"
	"Nogler/models/postgres"
	"log"
	"testing"
	"time"

	_ "github.com/lib/pq" // Add this line - PostgreSQL driver
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// Helper function to clean up after tests
func cleanupDB(t *testing.T, db *gorm.DB) {
	// Delete records in reverse order of dependencies
	assert.NoError(t, db.Exec("DELETE FROM in_game_players").Error)
	assert.NoError(t, db.Exec("DELETE FROM game_lobbies").Error)
	assert.NoError(t, db.Exec("DELETE FROM friendships").Error)
	assert.NoError(t, db.Exec("DELETE FROM friendship_requests").Error)
	assert.NoError(t, db.Exec("DELETE FROM users").Error)
	assert.NoError(t, db.Exec("DELETE FROM game_profiles").Error)
}

func TestUserAndGameProfile(t *testing.T) {
	db, err := config.ConnectGORM()
	if err != nil {
		t.Fatalf("Error connecting to PostgreSQL: %v", err)
	}

	defer cleanupDB(t, db)

	err = config.MigrateDatabase(db)
	if err != nil {
		t.Fatalf("Error migrating database: %v", err)
	}

	// Create test data
	profile := postgres.GameProfile{
		Username:  "testuser",
		UserIcon:  1,
		IsInAGame: false,
	}

	user := postgres.User{
		Email:           "test@example.com",
		ProfileUsername: "testuser",
		PasswordHash:    "hashedpassword",
		FullName:        "Test User",
		MemberSince:     time.Now(),
	}

	// Insert data
	err = db.Create(&profile).Error
	assert.NoError(t, err)

	err = db.Create(&user).Error
	assert.NoError(t, err)

	// Test retrieval
	var foundUser postgres.User
	err = db.Preload("GameProfile").Where("email = ?", "test@example.com").First(&foundUser).Error
	assert.NoError(t, err)
	assert.Equal(t, "testuser", foundUser.ProfileUsername)
	assert.Equal(t, "testuser", foundUser.GameProfile.Username)

	log.Println("FoundUser: ", foundUser)

	// Test game lobby creation
	lobby := postgres.GameLobby{
		ID:              "lobby1",
		CreatorUsername: "testuser",
		NumberOfRounds:  10,
	}

	err = db.Create(&lobby).Error
	assert.NoError(t, err)

	// Add player to lobby
	player := postgres.InGamePlayer{
		LobbyID:      "lobby1",
		Username:     "testuser",
		PlayersMoney: 1000,
	}

	err = db.Create(&player).Error
	assert.NoError(t, err)

	// Test retrieval of lobby with players
	var foundLobby postgres.GameLobby
	err = db.Preload("InGamePlayers").Preload("Creator").Where("id = ?", "lobby1").First(&foundLobby).Error
	assert.NoError(t, err)
	assert.Equal(t, 1, len(foundLobby.InGamePlayers))
	assert.Equal(t, "testuser", foundLobby.Creator.Username)
}
