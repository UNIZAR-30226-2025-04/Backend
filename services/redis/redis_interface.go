package redis

import (
	redis_models "Nogler/models/redis"
	redis_utils "Nogler/services/redis/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient handles Redis operations
type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisClient creates a new Redis client instance
func NewRedisClient(Addr string, DB int) *RedisClient {
	var client *redis.Client
	if Addr != "localhost:6379" {
		log.Println("Connecting to remote Redis...")
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
func (rc *RedisClient) SaveInGamePlayer(player *redis_models.InGamePlayer) error {
	key := redis_utils.FormatInGamePlayerKey(player.Username)
	data, err := json.Marshal(player)
	if err != nil {
		return fmt.Errorf("error marshaling player data: %v", err)
	}

	playerLobbyKey := redis_utils.FormatPlayerCurrentLobbyKey(player.Username)

	pipe := rc.client.Pipeline()
	pipe.Set(rc.ctx, key, data, 24*time.Hour)
	pipe.Set(rc.ctx, playerLobbyKey, player.LobbyId, 24*time.Hour)
	_, err = pipe.Exec(rc.ctx)
	return err
}

// GetInGamePlayer retrieves a player's game state from Redis
// Key format: "player:{username}:game"
// Returns: InGamePlayer struct or error
func (rc *RedisClient) GetInGamePlayer(username string) (*redis_models.InGamePlayer, error) {
	key := redis_utils.FormatInGamePlayerKey(username)
	data, err := rc.client.Get(rc.ctx, key).Bytes()
	if err != nil {
		return nil, fmt.Errorf("error getting player data: %v", err)
	}

	var player redis_models.InGamePlayer
	if err := json.Unmarshal(data, &player); err != nil {
		return nil, fmt.Errorf("error unmarshaling player data: %v", err)
	}
	return &player, nil
}

// DeleteInGamePlayer removes a player's game state from Redis
// Deletes both "player:{username}:game" and "player:{username}:current_lobby" keys
func (rc *RedisClient) DeleteInGamePlayer(username string, lobbyId string) error {
	// Create pipeline for atomic operation
	pipe := rc.client.Pipeline()

	// Delete player game state
	gameKey := redis_utils.FormatInGamePlayerKey(username)
	pipe.Del(rc.ctx, gameKey)

	// Delete player's current lobby reference
	lobbyKey := redis_utils.FormatPlayerCurrentLobbyKey(username)
	pipe.Del(rc.ctx, lobbyKey)

	// Execute pipeline
	_, err := pipe.Exec(rc.ctx)
	if err != nil {
		return fmt.Errorf("error deleting player data: %v", err)
	}

	return nil
}

// GetPlayerCurrentLobby retrieves the current lobby of a player
func (rc *RedisClient) GetPlayerCurrentLobby(playerName string) (string, error) {
	key := redis_utils.FormatPlayerCurrentLobbyKey(playerName)
	lobbyID, err := rc.client.Get(rc.ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("error getting player's current lobby: %v", err)
	}
	return lobbyID, nil
}

// SaveGameLobby stores a game lobby state in Redis
// Key format: "lobby:{id}"
// TTL: 24 hours
func (rc *RedisClient) SaveGameLobby(lobby *redis_models.GameLobby) error {
	key := redis_utils.FormatLobbyKey(lobby.Id)
	data, err := json.Marshal(lobby)
	if err != nil {
		return fmt.Errorf("error marshaling lobby data: %v", err)
	}
	return rc.client.Set(rc.ctx, key, data, 24*time.Hour).Err()
}

// GetGameLobby retrieves a game lobby state from Redis
// Key format: "lobby:{id}"
// Returns: GameLobby struct or error
func (rc *RedisClient) GetGameLobby(lobbyId string) (*redis_models.GameLobby, error) {
	key := redis_utils.FormatLobbyKey(lobbyId)
	data, err := rc.client.Get(rc.ctx, key).Bytes()
	if err != nil {
		return nil, fmt.Errorf("error getting lobby data: %v", err)
	}

	var lobby redis_models.GameLobby
	if err := json.Unmarshal(data, &lobby); err != nil {
		return nil, fmt.Errorf("error unmarshaling lobby data: %v", err)
	}
	return &lobby, nil
}

// AddChatMessage adds a new message to the lobby's chat history
// Key format: "lobby:{id}"
// Updates the chat_history field of the GameLobby
func (rc *RedisClient) AddChatMessage(lobbyId string, message *redis_models.ChatMessage) error {
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
func (rc *RedisClient) GetChatHistory(lobbyId string) ([]redis_models.ChatMessage, error) {
	lobby, err := rc.GetGameLobby(lobbyId)
	if err != nil {
		return nil, fmt.Errorf("error getting lobby for chat history: %v", err)
	}
	return lobby.ChatHistory, nil
}
