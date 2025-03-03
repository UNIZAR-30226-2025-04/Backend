package postgres

import (
	"time"
)

type User struct {
	Email        string    `gorm:"primaryKey;size:100;not null"`
	Username     string    `gorm:"size:50;not null"` // Clave ajena hacia GameProfile
	PasswordHash string    `gorm:"size:255;not null"`
	FullName     string    `gorm:"size:100"`
	MemberSince  time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	// Relación con el perfil de juego
	GameProfile GameProfile `gorm:"foreignKey:Username"`
}
