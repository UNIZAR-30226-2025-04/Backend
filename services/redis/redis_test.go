package redis

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestRedisOperations(t *testing.T) {
	rc, err := InitRedis("localhost:6379", 0)
	if err != nil {
		t.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer CloseRedis(rc)

	// Helper function to clean Redis data
	cleanupRedis := func() {
		keys := []string{
			"game_lobby:test_lobby_123",
			"player:test_player",
			"player_lobby:test_player",
			"chat_history:test_lobby_123",
		}
		for _, key := range keys {
			if err := rc.client.Del(rc.ctx, key).Err(); err != nil {
				t.Fatalf("Failed to cleanup Redis key %s: %v", key, err)
			}
		}
	}

	t.Run("GameLobby Operations", func(t *testing.T) {
		cleanupRedis()
		lobby := &GameLobby{
			Id:             "test_lobby_123",
			NumberOfRounds: 15,
			TotalPoints:    2000,
			ChatHistory:    []ChatMessage{},
		}

		fmt.Printf("\nOriginal Lobby Data: %+v\n", lobby)
		
		if err := rc.SaveGameLobby(lobby); err != nil {
			t.Errorf("Failed to save lobby: %v", err)
		}

		retrieved, err := rc.GetGameLobby("test_lobby_123")
		if err != nil {
			t.Errorf("Failed to get lobby: %v", err)
		}
		fmt.Printf("Retrieved Lobby from Redis: %+v\n", retrieved)

		if lobby.Id != retrieved.Id || 
		   lobby.NumberOfRounds != retrieved.NumberOfRounds ||
		   lobby.TotalPoints != retrieved.TotalPoints {
			t.Errorf("Lobby data mismatch.")
		}
	})

	t.Run("InGamePlayer Operations", func(t *testing.T) {
		cleanupRedis()
		player := &InGamePlayer{
			Username:      "test_player",
			LobbyId:      "test_lobby_123",
			PlayersMoney: 500,
			CurrentDeck:  json.RawMessage(`{"cards":["ace_hearts", "king_spades"]}`),
			Modifiers:    json.RawMessage(`{"double_points": true}`),
			CurrentJokers: json.RawMessage(`{"joker1": "active"}`),
		}

		fmt.Printf("\nOriginal Player Data: %+v\n", player)

		if err := rc.SaveInGamePlayer(player); err != nil {
			t.Errorf("Failed to save player: %v", err)
		}

		// Verify player's lobby ID
		lobbyID, err := rc.GetPlayerCurrentLobby("test_player")
		if err != nil {
			t.Errorf("Failed to get player's lobby ID: %v", err)
		}
		fmt.Printf("Player's Current Lobby ID: %s\n", lobbyID)
		
		if lobbyID != player.LobbyId {
			t.Errorf("Lobby ID mismatch. Expected %s, got %s", player.LobbyId, lobbyID)
		}

		// Get and verify player data
		retrieved, err := rc.GetInGamePlayer("test_player")
		if err != nil {
			t.Errorf("Failed to get player: %v", err)
		}
		fmt.Printf("Retrieved Player from Redis: %+v\n", retrieved)

		// Verify individual fields
		if player.Username != retrieved.Username ||
		   player.LobbyId != retrieved.LobbyId ||
		   player.PlayersMoney != retrieved.PlayersMoney {
			t.Errorf("Basic player data mismatch")
		}
	})

	t.Run("Chat Operations", func(t *testing.T) {
		cleanupRedis()
		messages := []*ChatMessage{
			{
				Message:   "Hello!",
				Username:  "test_player",
				Timestamp: time.Now(),
			},
			{
				Message:   "Ready to play",
				Username:  "test_player",
				Timestamp: time.Now(),
			},
		}

		for _, msg := range messages {
			fmt.Printf("\nSending message: %+v\n", msg)
			if err := rc.AddChatMessage("test_lobby_123", msg); err != nil {
				t.Errorf("Failed to add message: %v", err)
			}
		}

		history, err := rc.GetChatHistory("test_lobby_123")
		if err != nil {
			t.Errorf("Failed to get chat history: %v", err)
		}

		fmt.Printf("\nRetrieved Chat History:\n")
		for i, msg := range history {
			fmt.Printf("Message %d: %+v\n", i+1, msg)
		}

		if len(history) != len(messages) {
			t.Errorf("Chat history length mismatch. Expected %d, got %d", 
				len(messages), len(history))
		}
	})
} 