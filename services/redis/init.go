package redis

import (
	"context"
	"fmt"
	"log"
)

// InitRedis initializes the Redis connection and basic configuration
func InitRedis(Addr string, DB int) (*RedisClient, error) {
	rc := NewRedisClient(Addr, DB)
	
	// Test connection
	err := rc.Client.Ping(rc.Ctx).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}
	
	log.Println("Successfully connected to Redis")
	
	// Clear any existing data (optional, comment if not needed)
	err = rc.Client.FlushDB(context.Background()).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to flush Redis DB: %v", err)
	}
	
	return rc, nil
}

// CloseRedis gracefully closes the Redis connection
func CloseRedis(rc *RedisClient) error {
	if err := rc.Client.Close(); err != nil {
		return fmt.Errorf("error closing Redis connection: %v", err)
	}
	return nil
} 