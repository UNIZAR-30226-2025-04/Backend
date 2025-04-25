package postgres

import "gorm.io/datatypes"

/*
 * 'GameProfile' defines the structure for a user's game profile. It is
 * referenced in User, Friendship, FriendshipRequest, GameLobby, InGamePlayer, GameInvitation
 */
type GameProfile struct {
	// NOTE: not null also requires a default value: https://stackoverflow.com/questions/43633108/gorm-upgrading-columns-constraint-with-migration
	Username  string         `gorm:"type:varchar(50);primaryKey"`
	UserStats datatypes.JSON `gorm:"type:jsonb;default:'{}';index:idx_profile_stats,type:gin"`
	UserIcon  int            `gorm:"type:integer;default:0"`
	IsInAGame bool           `gorm:"type:boolean;default:false"`
	UserScore int            `gorm:"type:integer;default:0"`

	// NOTE: was creating a circular dependency between GameProfile and User
	// User            *User               `gorm:"foreignKey:Username"`
	// NOTE: constraints better be on the parent table (here): https://github.com/go-gorm/gorm/issues/4289#issuecomment-943366275
	// NOTE: changed from regular slices ([]) to pointer slices ([]*), better praxis
	// NOTE: See https://stackoverflow.com/a/59616996
	// NOTE: gormigrate for migrations: https://github.com/go-gormigrate/gormigrate
	// NOTE: constraints not being updated with AutoMigrate, see: https://github.com/go-gorm/gorm/issues/5559
	// NOTE: check https://stackoverflow.com/questions/73861414/cascading-delete-with-sqlite-isnt-working-with-a-join-table
	Friendships1     []*Friendship        `gorm:"foreignKey:Username1;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Friendships2     []*Friendship        `gorm:"foreignKey:Username2;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	FriendRequests1  []*FriendshipRequest `gorm:"foreignKey:Sender;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	FriendRequests2  []*FriendshipRequest `gorm:"foreignKey:Recipient;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	GameLobbies      []*GameLobby         `gorm:"foreignKey:CreatorUsername;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	InGamePlayers    []*InGamePlayer      `gorm:"foreignKey:Username;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	GameInvitations1 []*GameInvitation    `gorm:"foreignKey:SenderUsername;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	GameInvitations2 []*GameInvitation    `gorm:"foreignKey:InvitedUsername;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
