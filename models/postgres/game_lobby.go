package postgres

import (
	"time"
)

/*
 * 'GameLobby' defines the structure of a Balatro game lobby.
 * It contains references to GameProfile and InGamePlayer
 */
type GameLobby struct {
	ID              string    `gorm:"primaryKey;size:50;not null"`
	CreatorUsername string    `gorm:"size:50;index"` // Clave ajena hacia GameProfile, con índice
	NumberOfRounds  int       `gorm:"default:0"`
	TotalPoints     int       `gorm:"default:0"`
	CreatedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP"`

	// Relationships
	Creator GameProfile `gorm:"foreignKey:CreatorUsername;constraint:OnDelete:CASCADE"`
	// Relationship with players in the game
	InGamePlayers []InGamePlayer `gorm:"foreignKey:LobbyID;constraint:OnDelete:CASCADE"`
}
