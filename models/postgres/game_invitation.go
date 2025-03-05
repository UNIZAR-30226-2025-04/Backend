package postgres

import (
	"time"
)

/*
 * 'GameInvitation' represents an invitation to a Balatro game. It contains
 * a reference to GameLobby and GameProfile
 */
type GameInvitation struct {
	LobbyID         string    `gorm:"primaryKey;size:50;not null"`
	SenderUsername  string    `gorm:"primaryKey;size:50;not null"`
	InvitedUsername string    `gorm:"primaryKey;size:50;not null"`
	CreatedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP"`

	// Relationships
	GameLobby          GameLobby   `gorm:"foreignKey:LobbyID;constraint:OnDelete:CASCADE"`
	SenderGameProfile  GameProfile `gorm:"foreignKey:SenderUsername;constraint:OnDelete:CASCADE"`
	InvitedGameProfile GameProfile `gorm:"foreignKey:InvitedUsername;constraint:OnDelete:CASCADE"`
}
