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
	err := rc.client.Ping(rc.ctx).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	log.Println("Successfully connected to Redis")

	// Clear any existing data (optional, comment if not needed)
	err = rc.client.FlushDB(context.Background()).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to flush Redis DB: %v", err)
	}
	return rc, nil
}

// CloseRedis gracefully closes the Redis connection
func CloseRedis(rc *RedisClient) error {
	if err := rc.client.Close(); err != nil {
		return fmt.Errorf("error closing Redis connection: %v", err)
	}
	return nil
}
