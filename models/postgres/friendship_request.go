package postgres

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

/*
 * The type 'FriendshipRequest' represents a friendship request
 */
type FriendshipRequest struct {
  Sender    string    `gorm:"primaryKey;size:50;not null;index:idx_friend_request_sender"`
	Recipient string    `gorm:"primaryKey;size:50;not null;index:idx_friend_request_recipient"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`

	// Relationships
	GameProfile1 GameProfile `gorm:"foreignKey:Sender"`
	GameProfile2 GameProfile `gorm:"foreignKey:Recipient"`
}

// GORM hook to ensure that both user's usernames are different
func (fr *FriendshipRequest) BeforeSave(tx *gorm.DB) error {
	if fr.Sender == fr.Recipient {
		return errors.New("Cannot request friendship to oneself")
	}
	return nil
}
