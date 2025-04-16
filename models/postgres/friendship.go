package postgres

import (
	"errors"

	"gorm.io/gorm"
)

/*
 * 'Friendship' represents a friendship between two users.
 */
type Friendship struct {
	Username1 string `gorm:"primaryKey;type:varchar(50);index:idx_friendships_username2"`
	Username2 string `gorm:"primaryKey;type:varchar(50)"`

	// Relationships
	User1 GameProfile `gorm:"foreignKey:Username1"`
	User2 GameProfile `gorm:"foreignKey:Username2"`
}

// GORM hook to ensure that both user's usernames are different
func (f *Friendship) BeforeSave(tx *gorm.DB) error {
	if f.Username1 == f.Username2 {
		return errors.New("cannot create friendship between the same user")
	}
	return nil
}
