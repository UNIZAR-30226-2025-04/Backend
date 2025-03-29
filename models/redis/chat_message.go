package redis

import "time"

// ChatMessage represents a message in the game chat
type ChatMessage struct {
	Message   string    `json:"message"`
	Username  string    `json:"username"`
	Timestamp time.Time `json:"timestamp"`
}
