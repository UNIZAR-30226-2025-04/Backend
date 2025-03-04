package sync

import (
	"Nogler/services/redis"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// Helper function to print game lobby table
func printGameLobbyTable(t *testing.T, db *sql.DB) {
	rows, err := db.Query(`
		SELECT id, creator_username, number_of_rounds, total_points, created_at 
		FROM game_lobbies
		ORDER BY id`)
	if err != nil {
		t.Fatalf("Failed to query game_lobbies: %v", err)
	}
	defer rows.Close()

	fmt.Println("\nGame Lobbies Table:")
	fmt.Printf("%-15s %-15s %-15s %-15s %-25s\n",
		"ID", "Creator", "Rounds", "Points", "Created At")
	fmt.Println(strings.Repeat("-", 85))

	for rows.Next() {
		var id, creator string
		var rounds, points int
		var createdAt time.Time
		if err := rows.Scan(&id, &creator, &rounds, &points, &createdAt); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		fmt.Printf("%-15s %-15s %-15d %-15d %-25s\n",
			id, creator, rounds, points, createdAt.Format("2006-01-02 15:04:05"))
	}
}

// Helper function to print in-game players table
func printInGamePlayersTable(t *testing.T, db *sql.DB) {
	rows, err := db.Query(`
		SELECT lobby_id, username, players_money, most_played_hand, winner 
		FROM in_game_players
		ORDER BY lobby_id, username`)
	if err != nil {
		t.Fatalf("Failed to query in_game_players: %v", err)
	}
	defer rows.Close()

	fmt.Println("\nIn Game Players Table:")
	fmt.Printf("%-15s %-15s %-15s %-40s %-8s\n",
		"Lobby ID", "Username", "Money", "Most Played Hand", "Winner")
	fmt.Println(strings.Repeat("-", 93))

	for rows.Next() {
		var lobbyId, username string
		var money int
		var hand json.RawMessage
		var winner bool
		if err := rows.Scan(&lobbyId, &username, &money, &hand, &winner); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		fmt.Printf("%-15s %-15s %-15d %-40s %-8v\n",
			lobbyId, username, money, string(hand), winner)
	}
}

func TestSyncManager(t *testing.T) {
	// Configure connections
	redisClient, err := redis.InitRedis("localhost:6379", 0)
	if err != nil {
		t.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer redis.CloseRedis(redisClient)

	db, err := sql.Open("postgres", "postgresql://nogler:nogler@localhost:5432/nogler?sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	syncManager := NewSyncManager(redisClient, db)

	// Test data
	testLobbyId := "test_lobby_1"
	testUsername := "test_user_1"

	// Helper function to clean Redis
	cleanupRedis := func() {
		keys := []string{
			"game_lobby:test_lobby_1",
			"player:test_user_1",
			"player_lobby:test_user_1",
			"chat_history:test_lobby_1",
		}
		if err := redisClient.CleanupKeys(keys); err != nil {
			t.Fatalf("Failed to cleanup Redis keys: %v", err)
		}
	}

	// Helper function to restore PostgreSQL data
	restoreInitialData := func(db *sql.DB) {
		// Execute populate.sql script to restore original data
		_, err := db.Exec(`
			UPDATE in_game_players 
			SET players_money = 0,
				most_played_hand = '{}'::jsonb
			WHERE username = $1 AND lobby_id = $2`,
			testUsername, testLobbyId)
		if err != nil {
			t.Fatalf("Failed to restore in_game_players: %v", err)
		}

		_, err = db.Exec(`
			UPDATE game_lobbies 
			SET number_of_rounds = 0,
				total_points = 0
			WHERE id = $1`,
			testLobbyId)
		if err != nil {
			t.Fatalf("Failed to restore game_lobbies: %v", err)
		}
	}

	t.Run("Test Game State Sync", func(t *testing.T) {
		cleanupRedis()
		restoreInitialData(db)
		fmt.Println("\n=== Testing Game State Synchronization ===")
		fmt.Println("\nInitial States:")
		fmt.Println("PostgreSQL Initial State:")
		printInGamePlayersTable(t, db)

		// Set initial data in PostgreSQL
		initialSetupQuery := `
			UPDATE in_game_players 
			SET players_money = 500,
				most_played_hand = '{"cards":["2♠","3♥"]}'
			WHERE username = $1 AND lobby_id = $2
		`
		_, err = db.Exec(initialSetupQuery, testUsername, testLobbyId)
		if err != nil {
			t.Fatalf("Failed to set initial PostgreSQL state: %v", err)
		}

		// Create data in Redis
		player := &redis.InGamePlayer{
			Username:     testUsername,
			LobbyId:      testLobbyId,
			PlayersMoney: 1000,
			MostPlayedHand: json.RawMessage(`{
				"hand_type": "full_house",
				"cards": ["AH", "AD", "AC", "KH", "KD"]
			}`),
		}

		fmt.Println("\nRedis Initial State:")
		fmt.Printf("%+v\n", player)

		// Save to Redis and sync
		err := redisClient.SaveInGamePlayer(player)
		if err != nil {
			t.Fatalf("Failed to save player in Redis: %v", err)
		}

		err = syncManager.SyncPlayerGameState(testUsername, testLobbyId)
		if err != nil {
			t.Fatalf("Failed to sync player state: %v", err)
		}

		// Silent verifications
		var dbMoney int
		var dbHand json.RawMessage
		err = db.QueryRow(`
			SELECT players_money, most_played_hand 
			FROM in_game_players 
			WHERE username = $1 AND lobby_id = $2`,
			testUsername, testLobbyId).Scan(&dbMoney, &dbHand)

		if err != nil {
			t.Fatalf("Failed to verify PostgreSQL data: %v", err)
		}

		if dbMoney != player.PlayersMoney {
			t.Errorf("Money mismatch. Got %d, want %d", dbMoney, player.PlayersMoney)
		}

		fmt.Println("\nFinal PostgreSQL State:")
		printInGamePlayersTable(t, db)
	})

	t.Run("Test Lobby State Sync", func(t *testing.T) {
		cleanupRedis()
		restoreInitialData(db)
		fmt.Println("\n=== Testing Lobby State Synchronization ===")
		fmt.Println("\nInitial States:")
		fmt.Println("PostgreSQL Initial State:")
		printGameLobbyTable(t, db)

		// First, set different initial data in PostgreSQL
		initialSetupQuery := `
			UPDATE game_lobbies 
			SET number_of_rounds = 1,
				total_points = 100
			WHERE id = $1
		`
		_, err = db.Exec(initialSetupQuery, testLobbyId)
		if err != nil {
			t.Fatalf("Failed to set initial PostgreSQL state: %v", err)
		}

		// Create test data in Redis (different from initial)
		lobby := &redis.GameLobby{
			Id:             testLobbyId,
			NumberOfRounds: 5,
			TotalPoints:    2000,
			ChatHistory:    []redis.ChatMessage{},
		}

		fmt.Printf("\nData in Redis before sync:\n%+v\n", lobby)

		// Save to Redis
		err := redisClient.SaveGameLobby(lobby)
		if err != nil {
			t.Fatalf("Failed to save lobby in Redis: %v", err)
		}

		// Show initial PostgreSQL state
		var initialRounds, initialPoints int
		err = db.QueryRow(`
			SELECT number_of_rounds, total_points 
			FROM game_lobbies 
			WHERE id = $1`,
			testLobbyId).Scan(&initialRounds, &initialPoints)

		fmt.Printf("\nInitial data in PostgreSQL:\nRounds: %d\nPoints: %d\n",
			initialRounds, initialPoints)

		// Sync with PostgreSQL
		err = syncManager.SyncLobbyState(testLobbyId)
		if err != nil {
			t.Fatalf("Failed to sync lobby state: %v", err)
		}

		// Verify in PostgreSQL
		var dbRounds, dbPoints int
		err = db.QueryRow(`
			SELECT number_of_rounds, total_points 
			FROM game_lobbies 
			WHERE id = $1`,
			testLobbyId).Scan(&dbRounds, &dbPoints)

		if err != nil {
			t.Fatalf("Failed to verify PostgreSQL data: %v", err)
		}

		if dbRounds != lobby.NumberOfRounds {
			t.Errorf("Rounds mismatch. Got %d, want %d", dbRounds, lobby.NumberOfRounds)
		}
		if dbPoints != lobby.TotalPoints {
			t.Errorf("Points mismatch. Got %d, want %d", dbPoints, lobby.TotalPoints)
		}

		fmt.Println("\nFinal State:")
		printGameLobbyTable(t, db)
	})
}
