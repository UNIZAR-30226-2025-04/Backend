package postgres

import (
	"gorm.io/datatypes"
)

type GameProfile struct {
	Username  string         `gorm:"primaryKey;size:50;not null"`
	UserStats datatypes.JSON `gorm:"default:'{}'"` // Usamos `datatypes.JSON` para manejar JSONB en Gorm
	UserIcon  int            `gorm:"default:0"`
	IsInAGame bool           `gorm:"default:false"` // Campo booleano
	// Relaciones con otras tablas
	Users           []User              `gorm:"foreignKey:Username"`
	Friendships1    []Friendship        `gorm:"foreignKey:Username1"`
	Friendships2    []Friendship        `gorm:"foreignKey:Username2"`
	FriendRequests1 []FriendshipRequest `gorm:"foreignKey:Username1"`
	FriendRequests2 []FriendshipRequest `gorm:"foreignKey:Username2"`
	GameLobbies     []GameLobby         `gorm:"foreignKey:CreatorUsername"`
	InGamePlayers   []InGamePlayer      `gorm:"foreignKey:Username"`
	GameInvitations []GameInvitation    `gorm:"foreignKey:InvitedUsername"`
}
