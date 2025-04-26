package postgres

import (
	"math/rand"
	"time"

	"gorm.io/gorm"
)

/*
 * 'GameLobby' defines the structure of a Nogler game lobby.
 * It contains references to GameProfile and InGamePlayer
 */
type GameLobby struct {
	ID              string    `gorm:"primaryKey;size:50;not null"`
	CreatorUsername string    `gorm:"size:50;index:idx_game_lobbies_creator"` // Clave ajena hacia GameProfile, con índice
	NumberOfRounds  int       `gorm:"default:0"`
	TotalPoints     int       `gorm:"default:0"`
	CreatedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	GameHasBegun    bool      `gorm:"default:false;index_idx_game_lobbies_active"` // Indicates if the game has started
	IsPublic        int       `gorm:"default:0;index:idx_game_lobbies_public"` // Indicates if the lobby is public (1), private (0) or AI (2)

	// Relationships
	Creator GameProfile `gorm:"foreignKey:CreatorUsername"`
	// Relationship with players in the game
	InGamePlayers []*InGamePlayer `gorm:"foreignKey:LobbyID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// This needs to be defined here, since it is a "gamelobby" type funcion
// Wanted to put it in utils but not really working there... sry

// Random lobby id generation
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateLobbyID(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// Ensure the id is trully unique. We wont have problems, reduced number of ids
func (l *GameLobby) BeforeCreate(tx *gorm.DB) (err error) {
	// Ensure the generated ID is unique
	for {
		newID := generateLobbyID(4) // Example: "aB3dE9"
		var existing GameLobby
		if err := tx.Where("id = ?", newID).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// If no existing record, use this ID
				l.ID = newID
				return nil
			}
			// Return any unexpected error
			return err
		}
		// Otherwise, loop again to generate a new unique ID
	}
}
