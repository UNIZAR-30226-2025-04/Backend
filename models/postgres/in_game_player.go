package postgres

import (
	"gorm.io/datatypes"
)

type InGamePlayer struct {
	LobbyID        string         `gorm:"size:50;not null"`       // Clave ajena hacia GameLobby
	Username       string         `gorm:"size:50;not null;index"` // Clave ajena hacia GameProfile, con índice
	PlayersMoney   int            `gorm:"default:0"`
	MostPlayedHand datatypes.JSON `gorm:"default:'{}'"` // Usamos `datatypes.JSON` para manejar JSONB
	Winner         bool           `gorm:"default:false"`
	// Relación con el lobby y el perfil de jugador
	GameLobby   GameLobby   `gorm:"foreignKey:LobbyID"`
	GameProfile GameProfile `gorm:"foreignKey:Username"`
	// Llave primaria compuesta
	PrimaryKey struct{} `gorm:"primaryKey;"`
}
