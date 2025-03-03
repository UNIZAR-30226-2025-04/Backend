package postgres

import (
	"time"
)

type FriendshipRequest struct {
	Username1 string    `gorm:"size:50;not null"` // Clave ajena hacia GameProfile
	Username2 string    `gorm:"size:50;not null"` // Clave ajena hacia GameProfile
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	// Relación con los perfiles de juego
	GameProfile1 GameProfile `gorm:"foreignKey:Username1"`
	GameProfile2 GameProfile `gorm:"foreignKey:Username2"`
	// Llave primaria compuesta
	PrimaryKey struct{} `gorm:"primaryKey;"`
}
