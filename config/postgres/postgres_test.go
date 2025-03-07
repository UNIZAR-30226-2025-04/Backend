package postgres

import (
	"Nogler/models/postgres"
	"log"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var db *gorm.DB

func setup() {
	var err error
	db, err = ConnectGORM()
	if err != nil {
		panic("failed to connect to database")
	}
}

func TestMain(m *testing.M) {
	// Load environment variables from .env file
	err := godotenv.Load("../../.env")
	if err != nil {
		panic("Error loading .env file")
	}

	// Setup database connection
	setup()

	// Run tests
	m.Run()
}

func recordExists(model interface{}, query string, args ...interface{}) bool {
	var count int64
	db.Model(model).Where(query, args...).Count(&count)
	return count > 0
}

func TestAddMultipleGameProfiles(t *testing.T) {
	profiles := []postgres.GameProfile{
		{Username: "testuser1"},
		{Username: "testuser2"},
		{Username: "testuser3"},
		{Username: "testuser4"},
	}

	for _, profile := range profiles {
		if recordExists(&postgres.GameProfile{}, "username = ?", profile.Username) {
			log.Printf("GameProfile with username %s already exists", profile.Username)
			continue
		}
		err := db.Create(&profile).Error
		if err != nil {
			log.Printf("Error creating GameProfile %s: %v", profile.Username, err)
			continue
		}
	}
}

func TestAddMultipleUsers(t *testing.T) {
	// Create new GameProfiles for users if they don't exist
	profiles := []postgres.GameProfile{
		{Username: "testuser3"},
		{Username: "testuser4"},
	}

	for _, profile := range profiles {
		if recordExists(&postgres.GameProfile{}, "username = ?", profile.Username) {
			log.Printf("GameProfile with username %s already exists", profile.Username)
			continue
		}
		err := db.Create(&profile).Error
		if err != nil {
			log.Printf("Error creating GameProfile %s: %v", profile.Username, err)
			continue
		}
	}

	// Add users that reference the game profiles
	users := []postgres.User{
		{
			Email:           "test3@example.com",
			ProfileUsername: "testuser3",
			PasswordHash:    "hashedpassword3",
			MemberSince:     time.Now(),
		},
		{
			Email:           "test4@example.com",
			ProfileUsername: "testuser4",
			PasswordHash:    "hashedpassword4",
			MemberSince:     time.Now(),
		},
	}

	for _, user := range users {
		if recordExists(&postgres.User{}, "email = ?", user.Email) {
			log.Printf("User with email %s already exists", user.Email)
			continue
		}
		err := db.Create(&user).Error
		if err != nil {
			log.Printf("Error creating User %s: %v", user.Email, err)
			continue
		}
	}
}

func TestAddMultipleFriendships(t *testing.T) {
	// Ensure GameProfiles exist
	profiles := []postgres.GameProfile{
		{Username: "testuser1"},
		{Username: "testuser2"},
	}

	for _, profile := range profiles {
		if recordExists(&postgres.GameProfile{}, "username = ?", profile.Username) {
			log.Printf("GameProfile with username %s already exists", profile.Username)
			continue
		}
		err := db.Create(&profile).Error
		if err != nil {
			log.Printf("Error creating GameProfile %s: %v", profile.Username, err)
			continue
		}
	}

	// Create friendships between users
	friendships := []postgres.Friendship{
		{Username1: "testuser1", Username2: "testuser2"},
		{Username1: "testuser2", Username2: "testuser1"},
	}

	for _, friendship := range friendships {
		if recordExists(&postgres.Friendship{}, "username1 = ? AND username2 = ?", friendship.Username1, friendship.Username2) {
			log.Printf("Friendship between %s and %s already exists", friendship.Username1, friendship.Username2)
			continue
		}
		err := db.Create(&friendship).Error
		if err != nil {
			log.Printf("Error creating Friendship between %s and %s: %v", friendship.Username1, friendship.Username2, err)
			continue
		}
	}
}

func TestAddMultipleFriendshipRequests(t *testing.T) {
	// Ensure GameProfiles exist
	profiles := []postgres.GameProfile{
		{Username: "testuser1"},
		{Username: "testuser2"},
	}

	for _, profile := range profiles {
		if recordExists(&postgres.GameProfile{}, "username = ?", profile.Username) {
			log.Printf("GameProfile with username %s already exists", profile.Username)
			continue
		}
		err := db.Create(&profile).Error
		if err != nil {
			log.Printf("Error creating GameProfile %s: %v", profile.Username, err)
			continue
		}
	}

	// Create friendship requests
	requests := []postgres.FriendshipRequest{
		{Sender: "testuser1", Recipient: "testuser2", CreatedAt: time.Now()},
		{Sender: "testuser2", Recipient: "testuser1", CreatedAt: time.Now()},
	}

	for _, request := range requests {
		if recordExists(&postgres.FriendshipRequest{}, "sender = ? AND recipient = ?", request.Sender, request.Recipient) {
			log.Printf("FriendshipRequest from %s to %s already exists", request.Sender, request.Recipient)
			continue
		}
		err := db.Create(&request).Error
		if err != nil {
			log.Printf("Error creating FriendshipRequest from %s to %s: %v", request.Sender, request.Recipient, err)
			continue
		}
	}
}

func TestAddMultipleGameLobbies(t *testing.T) {
	// Ensure GameProfiles exist
	profiles := []postgres.GameProfile{
		{Username: "testuser1"},
		{Username: "testuser2"},
	}

	for _, profile := range profiles {
		if recordExists(&postgres.GameProfile{}, "username = ?", profile.Username) {
			log.Printf("GameProfile with username %s already exists", profile.Username)
			continue
		}
		err := db.Create(&profile).Error
		if err != nil {
			log.Printf("Error creating GameProfile %s: %v", profile.Username, err)
			continue
		}
	}

	// Create game lobbies
	lobbies := []postgres.GameLobby{
		{ID: "lobby1", CreatorUsername: "testuser1", NumberOfRounds: 5, TotalPoints: 100, CreatedAt: time.Now()},
		{ID: "lobby2", CreatorUsername: "testuser2", NumberOfRounds: 3, TotalPoints: 50, CreatedAt: time.Now()},
	}

	for _, lobby := range lobbies {
		if recordExists(&postgres.GameLobby{}, "id = ?", lobby.ID) {
			log.Printf("GameLobby with ID %s already exists", lobby.ID)
			continue
		}
		err := db.Create(&lobby).Error
		if err != nil {
			log.Printf("Error creating GameLobby %s: %v", lobby.ID, err)
			continue
		}
	}
}

func TestAddMultipleInGamePlayers(t *testing.T) {
	// Ensure GameProfiles exist
	profiles := []postgres.GameProfile{
		{Username: "testuser1"},
		{Username: "testuser2"},
	}

	for _, profile := range profiles {
		if recordExists(&postgres.GameProfile{}, "username = ?", profile.Username) {
			log.Printf("GameProfile with username %s already exists", profile.Username)
			continue
		}
		err := db.Create(&profile).Error
		if err != nil {
			log.Printf("Error creating GameProfile %s: %v", profile.Username, err)
			continue
		}
	}

	// Ensure GameLobbies exist
	lobbies := []postgres.GameLobby{
		{ID: "lobby1", CreatorUsername: "testuser1", NumberOfRounds: 5, TotalPoints: 100, CreatedAt: time.Now()},
		{ID: "lobby2", CreatorUsername: "testuser2", NumberOfRounds: 3, TotalPoints: 50, CreatedAt: time.Now()},
	}

	for _, lobby := range lobbies {
		if recordExists(&postgres.GameLobby{}, "id = ?", lobby.ID) {
			log.Printf("GameLobby with ID %s already exists", lobby.ID)
			continue
		}
		err := db.Create(&lobby).Error
		if err != nil {
			log.Printf("Error creating GameLobby %s: %v", lobby.ID, err)
			continue
		}
	}

	// Create in-game players
	players := []postgres.InGamePlayer{
		{LobbyID: "lobby1", Username: "testuser1", PlayersMoney: 1000},
		{LobbyID: "lobby2", Username: "testuser2", PlayersMoney: 1500},
		{LobbyID: "lobby1", Username: "testuser2", PlayersMoney: 2000},
		{LobbyID: "lobby2", Username: "testuser1", PlayersMoney: 2500},
	}

	for _, player := range players {
		if recordExists(&postgres.InGamePlayer{}, "lobby_id = ? AND username = ?", player.LobbyID, player.Username) {
			log.Printf("InGamePlayer with LobbyID %s and Username %s already exists", player.LobbyID, player.Username)
			continue
		}
		err := db.Create(&player).Error
		if err != nil {
			log.Printf("Error creating InGamePlayer for lobby %s and user %s: %v", player.LobbyID, player.Username, err)
			continue
		}
	}
}

func TestAddMultipleGameInvitations(t *testing.T) {
	// Ensure GameProfiles exist
	profiles := []postgres.GameProfile{
		{Username: "testuser1"},
		{Username: "testuser2"},
	}

	for _, profile := range profiles {
		if recordExists(&postgres.GameProfile{}, "username = ?", profile.Username) {
			log.Printf("GameProfile with username %s already exists", profile.Username)
			continue
		}
		err := db.Create(&profile).Error
		if err != nil {
			log.Printf("Error creating GameProfile %s: %v", profile.Username, err)
			continue
		}
	}

	// Ensure GameLobbies exist
	lobbies := []postgres.GameLobby{
		{ID: "lobby1", CreatorUsername: "testuser1", NumberOfRounds: 5, TotalPoints: 100, CreatedAt: time.Now()},
		{ID: "lobby2", CreatorUsername: "testuser2", NumberOfRounds: 3, TotalPoints: 50, CreatedAt: time.Now()},
	}

	for _, lobby := range lobbies {
		if recordExists(&postgres.GameLobby{}, "id = ?", lobby.ID) {
			log.Printf("GameLobby with ID %s already exists", lobby.ID)
			continue
		}
		err := db.Create(&lobby).Error
		if err != nil {
			log.Printf("Error creating GameLobby %s: %v", lobby.ID, err)
			continue
		}
	}

	// Create game invitations
	invitations := []postgres.GameInvitation{
		{LobbyID: "lobby1", InvitedUsername: "testuser2", SenderUsername: "testuser1", CreatedAt: time.Now()},
		{LobbyID: "lobby2", InvitedUsername: "testuser1", SenderUsername: "testuser2", CreatedAt: time.Now()},
	}

	for _, invitation := range invitations {
		if recordExists(&postgres.GameInvitation{}, "lobby_id = ? AND invited_username = ?", invitation.LobbyID, invitation.InvitedUsername) {
			log.Printf("GameInvitation for LobbyID %s and InvitedUsername %s already exists", invitation.LobbyID, invitation.InvitedUsername)
			continue
		}
		err := db.Create(&invitation).Error
		if err != nil {
			log.Printf("Error creating GameInvitation for lobby %s and user %s: %v", invitation.LobbyID, invitation.InvitedUsername, err)
			continue
		}
	}
}
