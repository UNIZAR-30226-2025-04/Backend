package redis

import (
	"fmt"
)

// CleanupKeys removes the specified keys from Redis
func (rc *RedisClient) CleanupKeys(keys []string) error {
    for _, key := range keys {
        if err := rc.Client.Del(rc.Ctx, key).Err(); err != nil {
            return fmt.Errorf("failed to cleanup Redis key %s: %v", key, err)
        }
    }
    return nil
} 