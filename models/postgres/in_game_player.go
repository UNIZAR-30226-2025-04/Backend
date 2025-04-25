package postgres

import (
	"gorm.io/datatypes"
)

/*
 * 'InGamePlayer' represents the state of a player in a game. It contains
 * references to GameLobby and GameProfile
 */
// TODO: use it when a player wins / loses a game
type InGamePlayer struct {
	// NOTE: composite primary key definition
	LobbyID        string         `gorm:"primaryKey;size:50;not null"`
	Username       string         `gorm:"primaryKey;size:50;not null;index"`
	PlayersMoney   int            `gorm:"default:0"`
	MostPlayedHand datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	Winner         bool           `gorm:"default:false"`

	// Relationship with the lobby and the user's game profile
	GameLobby   GameLobby   `gorm:"foreignKey:LobbyID"`
	GameProfile GameProfile `gorm:"foreignKey:Username"`
}
