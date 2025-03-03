package postgres

type Friendship struct {
	Username1 string `gorm:"primaryKey;type:varchar(50);index:idx_friendships_username2"`
	Username2 string `gorm:"primaryKey;type:varchar(50)"`

	// Relationships
	User1 GameProfile `gorm:"foreignKey:Username1;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	User2 GameProfile `gorm:"foreignKey:Username2;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
