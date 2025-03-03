package postgres

import (
	"gorm.io/datatypes"
)

/*
 * 'GameProfile' defines the structure for a user's game profile. It is
 * referenced in User, Friendship, FriendshipRequest, GameLobby, InGamePlayer, GameInvitation
 */
type GameProfile struct {
	Username  string         `gorm:"primaryKey;size:50;not null"`
	UserStats datatypes.JSON `gorm:"default:'{}'"` // We use `datatypes.JSON` to handle JSONB on Gorm
	UserIcon  int            `gorm:"default:0"`
	IsInAGame bool           `gorm:"default:false"`

	// Relationships with other tables
	// Modelled as 0..1 for simplicity on GORM
	User            *User               `gorm:"foreignKey:Username"`
	Friendships1    []Friendship        `gorm:"foreignKey:Username1"`
	Friendships2    []Friendship        `gorm:"foreignKey:Username2"`
	FriendRequests1 []FriendshipRequest `gorm:"foreignKey:Username1"`
	FriendRequests2 []FriendshipRequest `gorm:"foreignKey:Username2"`
	GameLobbies     []GameLobby         `gorm:"foreignKey:CreatorUsername"`
	InGamePlayers   []InGamePlayer      `gorm:"foreignKey:Username"`
	GameInvitations []GameInvitation    `gorm:"foreignKey:InvitedUsername"`
}
