package postgres

import "gorm.io/datatypes"

/*
 * 'GameProfile' defines the structure for a user's game profile. It is
 * referenced in User, Friendship, FriendshipRequest, GameLobby, InGamePlayer, GameInvitation
 */
type GameProfile struct {
	Username  string         `gorm:"type:varchar(50);primaryKey;not null"`
	UserStats datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	UserIcon  int            `gorm:"type:integer;default:0"`
	IsInAGame bool           `gorm:"type:boolean;default:false"`

	// NOTE: was creating a circular dependency between GameProfile and User
	// User            *User               `gorm:"foreignKey:Username"`
	// NOTE: constraints better be on the parent table (here): https://github.com/go-gorm/gorm/issues/4289#issuecomment-943366275
	// NOTE: changed from regular slices ([]) to pointer slices ([]*), better praxis
	Friendships1    []*Friendship        `gorm:"foreignKey:Username1;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Friendships2    []*Friendship        `gorm:"foreignKey:Username2;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	FriendRequests1 []*FriendshipRequest `gorm:"foreignKey:Sender;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	FriendRequests2 []*FriendshipRequest `gorm:"foreignKey:Recipient;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	GameLobbies     []*GameLobby         `gorm:"foreignKey:CreatorUsername;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	InGamePlayers   []*InGamePlayer      `gorm:"foreignKey:Username;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	GameInvitations []*GameInvitation    `gorm:"foreignKey:InvitedUsername;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
