package config

import (
	"Nogler/redis"
	"log"
	"os"
)

// Connect to Redis
func Connect_redis() (*redis.RedisClient, error) {
	redisUri := os.Getenv("REDIS_URL")
	redisClient, err := redis.InitRedis(redisUri, 0)
	if err != nil {
		log.Fatalf("Error connecting to Redis: %v", err)
		return nil, err
	}
	log.Println("Redis connection established")
	return redisClient, nil
}
