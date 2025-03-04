package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ChatMessage represents a message in the game chat
type ChatMessage struct {
	Message   string    `json:"message"`
	Username  string    `json:"username"`
	Timestamp time.Time `json:"timestamp"`
}

// InGamePlayer represents a player's state during a game
type InGamePlayer struct {
	Username       string          `json:"username"`         // Matches game_profiles.username
	LobbyId        string          `json:"lobby_id"`         // Matches game_lobbies.id
	PlayersMoney   int             `json:"players_money"`    // Matches in_game_players.players_money
	CurrentDeck    json.RawMessage `json:"current_deck"`     // Temporary Redis field
	Modifiers      json.RawMessage `json:"modifiers"`        // Temporary Redis field
	CurrentJokers  json.RawMessage `json:"current_jokers"`   // Temporary Redis field
	MostPlayedHand json.RawMessage `json:"most_played_hand"` // Matches in_game_players.most_played_hand
}

// GameLobby represents a game lobby state
type GameLobby struct {
	Id             string        `json:"id"`               // Matches game_lobbies.id
	NumberOfRounds int           `json:"number_of_rounds"` // Matches game_lobbies.number_of_rounds
	TotalPoints    int           `json:"total_points"`     // Matches game_lobbies.total_points
	ChatHistory    []ChatMessage `json:"chat_history"`     // Chat history for the lobby
}

// RedisClient handles Redis operations
type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisClient creates a new Redis client instance
func NewRedisClient(Addr string, DB int) *RedisClient {
	var client *redis.Client
	if Addr != "localhost:6379" {
		opt, err := redis.ParseURL(Addr)
		if err != nil {
			panic("Error parsing Redis URL")
		}
		client = redis.NewClient(opt)
	} else {
		client = redis.NewClient(&redis.Options{
			Addr: Addr,
			DB:   DB,
		})
	}
	return &RedisClient{
		client: client,
		ctx:    context.Background(),
	}
}

// SaveInGamePlayer stores a player's game state in Redis
// Key format: "player:{username}:game"
// TTL: 24 hours
func (rc *RedisClient) SaveInGamePlayer(player *InGamePlayer) error {
	key := fmt.Sprintf("player:%s:game", player.Username)
	data, err := json.Marshal(player)
	if err != nil {
		return fmt.Errorf("error marshaling player data: %v", err)
	}

	playerLobbyKey := fmt.Sprintf("player:%s:current_lobby", player.Username)
	pipe := rc.client.Pipeline()
	pipe.Set(rc.ctx, key, data, 24*time.Hour)
	pipe.Set(rc.ctx, playerLobbyKey, player.LobbyId, 24*time.Hour)
	_, err = pipe.Exec(rc.ctx)
	return err
}

// GetInGamePlayer retrieves a player's game state from Redis
// Key format: "player:{username}:game"
// Returns: InGamePlayer struct or error
func (rc *RedisClient) GetInGamePlayer(username string) (*InGamePlayer, error) {
	key := fmt.Sprintf("player:%s:game", username)
	data, err := rc.client.Get(rc.ctx, key).Bytes()
	if err != nil {
		return nil, fmt.Errorf("error getting player data: %v", err)
	}

	var player InGamePlayer
	if err := json.Unmarshal(data, &player); err != nil {
		return nil, fmt.Errorf("error unmarshaling player data: %v", err)
	}
	return &player, nil
}

// GetPlayerCurrentLobby retrieves the current lobby of a player
func (rc *RedisClient) GetPlayerCurrentLobby(playerName string) (string, error) {
	key := fmt.Sprintf("player:%s:current_lobby", playerName)
	lobbyID, err := rc.client.Get(rc.ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("error getting player's current lobby: %v", err)
	}
	return lobbyID, nil
}

// SaveGameLobby stores a game lobby state in Redis
// Key format: "lobby:{id}"
// TTL: 24 hours
func (rc *RedisClient) SaveGameLobby(lobby *GameLobby) error {
	key := fmt.Sprintf("lobby:%s", lobby.Id)
	data, err := json.Marshal(lobby)
	if err != nil {
		return fmt.Errorf("error marshaling lobby data: %v", err)
	}
	return rc.client.Set(rc.ctx, key, data, 24*time.Hour).Err()
}

// GetGameLobby retrieves a game lobby state from Redis
// Key format: "lobby:{id}"
// Returns: GameLobby struct or error
func (rc *RedisClient) GetGameLobby(lobbyId string) (*GameLobby, error) {
	key := fmt.Sprintf("lobby:%s", lobbyId)
	data, err := rc.client.Get(rc.ctx, key).Bytes()
	if err != nil {
		return nil, fmt.Errorf("error getting lobby data: %v", err)
	}

	var lobby GameLobby
	if err := json.Unmarshal(data, &lobby); err != nil {
		return nil, fmt.Errorf("error unmarshaling lobby data: %v", err)
	}
	return &lobby, nil
}

// AddChatMessage adds a new message to the lobby's chat history
// Key format: "lobby:{id}"
// Updates the chat_history field of the GameLobby
func (rc *RedisClient) AddChatMessage(lobbyId string, message *ChatMessage) error {
	lobby, err := rc.GetGameLobby(lobbyId)
	if err != nil {
		return fmt.Errorf("error getting lobby for chat: %v", err)
	}

	lobby.ChatHistory = append(lobby.ChatHistory, *message)
	return rc.SaveGameLobby(lobby)
}

// GetChatHistory retrieves the chat history for a lobby
// Key format: "lobby:{id}"
// Returns: Array of ChatMessage or error
func (rc *RedisClient) GetChatHistory(lobbyId string) ([]ChatMessage, error) {
	lobby, err := rc.GetGameLobby(lobbyId)
	if err != nil {
		return nil, fmt.Errorf("error getting lobby for chat history: %v", err)
	}
	return lobby.ChatHistory, nil
}
