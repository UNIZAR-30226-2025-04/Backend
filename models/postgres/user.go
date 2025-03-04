package postgres

import (
	"time"
)

/*
 * 'User' contains the blueprint definition of a User. It contains a reference to GameProfile
 */
type User struct {
	Email         string    `gorm:"primaryKey;size:100;not null"`
	NiggaUsername string    `gorm:"size:50;not null;uniqueIndex"`
	PasswordHash  string    `gorm:"size:255;not null"`
	FullName      string    `gorm:"size:100"`
	MemberSince   time.Time `gorm:"default:CURRENT_TIMESTAMP"`

	// Relationship with the game profile
	GameProfile GameProfile `gorm:"foreignKey:NiggaUsername;constraint:OnDelete:CASCADE"`
}
