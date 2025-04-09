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

// DE YAGO NOSE SI ESTÁ BIEN VALE???
//----------------------------------------------------------------------------------------------------

type HandLevel struct {
	Fichas      int `json:"fichas"`       // Score multiplier
	Mult        int `json:"mult"`         // XP needed for next level
	TimesPlayed int `json:"times_played"` // Tracking for stats
}

// Value supongo que usaremos en plan un int como "relacionador" de midifier con lo que hace pa aplicarlo
type Modifier struct {
	Value       float64   `json:"value"`
	ExpiresAt   time.Time `json:"expires_at"` // -1 if no acaba hasta final de partida?
	Description string    `json:"description"`
}

type Joker struct {
	ID string `json:"id"`
}

//----------------------------------------------------------------------------------------------------

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

	// Simply set the player data, lobby ID is contained within the player object
	return rc.client.Set(rc.ctx, key, data, 24*time.Hour).Err()
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
// Deletes the player's game state
func (rc *RedisClient) DeleteInGamePlayer(username string, lobbyId string) error {
	// Delete player game state only
	gameKey := redis_utils.FormatInGamePlayerKey(username)
	err := rc.client.Del(rc.ctx, gameKey).Err()
	if err != nil {
		return fmt.Errorf("error deleting player data: %v", err)
	}
	return nil
}

// GetPlayerCurrentLobby retrieves the current lobby of a player
// by extracting it from the player's game state
func (rc *RedisClient) GetPlayerCurrentLobby(playerName string) (string, error) {
	player, err := rc.GetInGamePlayer(playerName)
	if err != nil {
		return "", fmt.Errorf("error getting player's current lobby: %v", err)
	}
	return player.LobbyId, nil
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

// DeleteGameLobby removes a game lobby state from Redis
// Key format: "lobby:{id}"
// Deletes the lobby key
func (rc *RedisClient) DeleteGameLobby(lobbyId string) error {
	// Create pipeline for atomic operation
	pipe := rc.client.Pipeline()

	// Delete the lobby state
	lobbyKey := redis_utils.FormatLobbyKey(lobbyId)
	pipe.Del(rc.ctx, lobbyKey)

	// Execute pipeline
	_, err := pipe.Exec(rc.ctx)
	if err != nil {
		return fmt.Errorf("error deleting lobby data: %v", err)
	}
	return nil
}

// CloseLobby closes a lobby by updating its state in Redis
// Key format: "lobby:{id}"
// Updates the GameHasBegun field to true
func (rc *RedisClient) CloseLobby(lobbyId string) error {
	lobby, err := rc.GetGameLobby(lobbyId)
	if err != nil {
		return fmt.Errorf("error getting lobby for closing: %v", err)
	}

	lobby.GameHasBegun = true
	if err := rc.SaveGameLobby(lobby); err != nil {
		return fmt.Errorf("error saving closed lobby: %v", err)
	}
	return nil
}

// GetPackContents retrieves the PackContents for a specific key from Redis
func (rc *RedisClient) GetPackContents(key string) (*redis_models.PackContents, error) {
	data, err := rc.client.Get(rc.ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			// Key does not exist
			return nil, nil
		}
		return nil, fmt.Errorf("error getting pack contents from Redis: %v", err)
	}

	var contents redis_models.PackContents
	if err := json.Unmarshal(data, &contents); err != nil {
		return nil, fmt.Errorf("error unmarshaling pack contents: %v", err)
	}

	return &contents, nil
}

// SetPackContents saves the PackContents for a specific key in Redis with a TTL
func (rc *RedisClient) SetPackContents(key string, contents redis_models.PackContents, ttl time.Duration) error {
	data, err := json.Marshal(contents)
	if err != nil {
		return fmt.Errorf("error marshaling pack contents: %v", err)
	}

	if err := rc.client.Set(rc.ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("error setting pack contents in Redis: %v", err)
	}

	return nil
}
