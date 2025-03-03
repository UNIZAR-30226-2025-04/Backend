package postgres

import (
	"time"
)

type GameInvitation struct {
	LobbyID         string    `gorm:"size:50;not null"` // Clave ajena hacia GameLobby
	InvitedUsername string    `gorm:"size:50;not null"` // Clave ajena hacia GameProfile
	CreatedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	// Relación con el lobby y el perfil invitado
	GameLobby   GameLobby   `gorm:"foreignKey:LobbyID"`
	GameProfile GameProfile `gorm:"foreignKey:InvitedUsername"`
	// Llave primaria compuesta
	PrimaryKey struct{} `gorm:"primaryKey;"`
}
