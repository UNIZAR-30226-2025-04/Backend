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
	UserStats datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	UserIcon  int            `gorm:"default:0"`
	IsInAGame bool           `gorm:"default:false"`

	// NOTE: was creating a circular dependency between GameProfile and User
	// User            *User               `gorm:"foreignKey:Username"`
	Friendships1    []Friendship        `gorm:"foreignKey:Username1"`
	Friendships2    []Friendship        `gorm:"foreignKey:Username2"`
	FriendRequests1 []FriendshipRequest `gorm:"foreignKey:Username1"`
	FriendRequests2 []FriendshipRequest `gorm:"foreignKey:Username2"`
	GameLobbies     []GameLobby         `gorm:"foreignKey:CreatorUsername"`
	InGamePlayers   []InGamePlayer      `gorm:"foreignKey:Username"`
	GameInvitations []GameInvitation    `gorm:"foreignKey:InvitedUsername"`
}
