package postgres

import (
	"time"
)

/*
 * 'GameInvitation' represents an invitation to a Balatro game. It contains
 * a reference to GameLobby and GameProfile
 */
type GameInvitation struct {
  LobbyID         string    `gorm:"primaryKey;size:50;not null;index:idx_invitations_lobby_user, priority:1"`
  SenderUsername  string    `gorm:"primaryKey;size:50;not null;index:idx_invitations_sender"`
  InvitedUsername string    `gorm:"primaryKey;size:50;not null;index:idx_invitations_lobby_user,priority:2"`
	CreatedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP"`

	// Relationships
	GameLobby          GameLobby   `gorm:"foreignKey:LobbyID"`
	SenderGameProfile  GameProfile `gorm:"foreignKey:SenderUsername"`
	InvitedGameProfile GameProfile `gorm:"foreignKey:InvitedUsername"`
}
