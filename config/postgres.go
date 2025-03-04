package config

import (
	"Nogler/models/postgres"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq" // Add this line - PostgreSQL driver
	"gorm.io/datatypes"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Return all models for migration
func AllModels() []interface{} {
	return []interface{}{
		&postgres.GameProfile{},
		&postgres.User{},
		&postgres.Friendship{},
		&postgres.FriendshipRequest{},
		&postgres.GameLobby{},
		&postgres.InGamePlayer{},
		&postgres.GameInvitation{},
	}
}

// Connect_postgres returns a connection string for PostgreSQL
/*func Connect_postgres() (*sql.DB, error) {
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	database := os.Getenv("POSRGRES_DATABASE")

	pg_connection_string := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, database)

	//Start communication with Postgre DB
	db, err := sql.Open("postgres", pg_connection_string)
	if err != nil {
		log.Printf("Error connecting to PostgreSQL: %v", err)
		return nil, err
	}

	// Verify connection to PostgreSQL
	if err := db.Ping(); err != nil {
		log.Printf("Error making ping to PostgreSQL: %v", err)
		return nil, err
	}

	return db, nil
}*/

// ConnectGORM returns a GORM DB instance connected to PostgreSQL
func ConnectGORM() (*gorm.DB, error) {
	// user := os.Getenv("POSTGRES_USER")
	// password := os.Getenv("POSTGRES_PASSWORD")
	// host := os.Getenv("POSTGRES_HOST")
	// port := os.Getenv("POSTGRES_PORT")
	// database := os.Getenv("POSTGRES_DATABASE")

	//TODO quitar este hardcode
	user := "nogler_admin"
	password := "N0gler1234"
	host := "nogler.postgres.database.azure.com"
	port := "5432"
	database := "postgres"

	// NOTE: https://stackoverflow.com/questions/57205060/how-to-connect-postgresql-database-using-gorm
	// NOTE: See https://github.com/go-gorm/gorm/issues/5409
	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s",
		user, password, host, port, database)

	sqlDB1, err := sql.Open("postgres", dsn)

	if err != nil {
		log.Printf("Error connecting to PostgreSQL: %v", err)
		return nil, err
	}
	log.Println("Successfully connected to PostgreSQL")

	sqlDB1.Exec("SELECT * FROM USERS;")
	// Verify connection to PostgreSQL
	if err := sqlDB1.Ping(); err != nil {
		log.Printf("Error making ping to PostgreSQL: %v", err)
		return nil, err
	}
	log.Println("Postgre ping OK")

	/*newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level (Silent, Error, Warn, Info)
			IgnoreRecordNotFoundError: false,       // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Enable color
		},
	)*/

	db, err := gorm.Open(pgdriver.New(pgdriver.Config{
		Conn:                 sqlDB1,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		// Logger: newLogger, // Add the logger to the configuration
		//DisableForeignKeyConstraintWhenMigrating: true,
	})

	if err != nil {
		log.Printf("Error connecting to PostgreSQL with GORM: %v", err)
		return nil, err
	}

	log.Println("GORM opened PostgreSQL")

	// Get the underlying SQL DB object
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Error getting underlying SQL DB: %v", err)
		return nil, err
	}
	log.Println("Underlying database recovered correctly")

	// Verify connection
	if err := sqlDB.Ping(); err != nil {
		log.Printf("Error pinging PostgreSQL: %v", err)
		return nil, err
	}
	log.Println("PostgreSQL ping OK")

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Successfully connected to PostgreSQL with GORM")
	return db, nil
}

type Category struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"not null"`
}

type Movie struct {
	ID         uint     `gorm:"primaryKey"`
	Title      string   `gorm:"not null"`
	CategoryID uint     `gorm:"not null;index"`
	Category   Category `gorm:"foreignKey:CategoryID"`
}

func MigrateDatabase(db *gorm.DB) error {

	/*all_models := AllModels()

	// NOTE: See https://github.com/go-gorm/gorm/blob/9f273777f58a247f7ae4176a014f6d59ac9fff8c/migrator/migrator.go#L614
	for _, model := range all_models {
		modelName := reflect.TypeOf(model).Elem().Name()
		log.Printf("Dropping table for model: %s", modelName)

		if err := db.Migrator().DropTable(model); err != nil {
			return fmt.Errorf("failed to drop table for %s: %w", modelName, err)
		}
	}*/

	// TODO: check https://forums.devart.com/viewtopic.php?t=28835
	// TODO: check https://github.com/go-gorm/gorm/issues/4154
	// TODO: check https://stackoverflow.com/questions/51471973/gorm-automigrate-and-createtable-not-working
	// TODO: check https://github.com/pilinux/gorest/issues/167 => FIXED!
	// 		EXACTLY, this (postgres driver v1.4.0): https://github.com/pilinux/gorest/issues/167#issuecomment-1947114560
	// https://github.com/go-gorm/postgres/tags
	// NOTE: for more info, execute db.Debug().AutoMigrate(...)
	err := db.AutoMigrate(
		postgres.GameProfile{},
		postgres.User{},
		postgres.Friendship{},
		postgres.FriendshipRequest{},
		postgres.GameLobby{},
		postgres.InGamePlayer{},
		postgres.GameInvitation{})

	if err != nil {
		return fmt.Errorf("auto migration failed: %w", err)
	}
	log.Println("Database migrated successfully")

	// First create GameProfiles
	var profileCount int64
	db.Model(&postgres.GameProfile{}).Count(&profileCount)
	if profileCount == 0 {
		initialProfile := postgres.GameProfile{
			Username:  "admin",
			UserStats: datatypes.JSON([]byte(`{"wins": 0, "losses": 0}`)),
			UserIcon:  1,
			IsInAGame: false,
		}
		if err := db.Create(&initialProfile).Error; err != nil {
			return fmt.Errorf("failed to create initial game profile: %w", err)
		}
		log.Println("Created initial game profile")

		secondProfile := postgres.GameProfile{
			Username:  "player2",
			UserStats: datatypes.JSON([]byte(`{"wins": 0, "losses": 0}`)),
			UserIcon:  2,
			IsInAGame: false,
		}
		if err := db.Create(&secondProfile).Error; err != nil {
			return fmt.Errorf("failed to create second game profile: %w", err)
		}
		log.Println("Created second game profile")
	}

	// Then create Users
	var userCount int64
	db.Model(&postgres.User{}).Count(&userCount)
	if userCount == 0 {
		initialUser := postgres.User{
			Email:         "admin@example.com",
			NiggaUsername: "admin", // References existing GameProfile
			PasswordHash:  "$2a$10$example_hash",
			FullName:      "Admin User",
			MemberSince:   time.Now(),
		}
		if err := db.Create(&initialUser).Error; err != nil {
			return fmt.Errorf("failed to create initial user: %w", err)
		}
		log.Println("Created initial user")

		secondUser := postgres.User{
			Email:         "player2@example.com",
			NiggaUsername: "player2", // References existing GameProfile
			PasswordHash:  "$2a$10$example_hash",
			FullName:      "Player Two",
			MemberSince:   time.Now(),
		}
		if err := db.Create(&secondUser).Error; err != nil {
			return fmt.Errorf("failed to create second user: %w", err)
		}
		log.Println("Created second user")
	}

	// Create initial GameLobby
	var lobbyCount int64
	db.Model(&postgres.GameLobby{}).Count(&lobbyCount)
	if lobbyCount == 0 {
		initialLobby := postgres.GameLobby{
			ID:              "LOBBY1",
			CreatorUsername: "admin",
			NumberOfRounds:  3,
			TotalPoints:     0,
			CreatedAt:       time.Now(),
		}
		if err := db.Create(&initialLobby).Error; err != nil {
			return fmt.Errorf("failed to create initial game lobby: %w", err)
		}
		log.Println("Created initial game lobby")
	}

	// Create initial InGamePlayer
	var playerCount int64
	db.Model(&postgres.InGamePlayer{}).Count(&playerCount)
	if playerCount == 0 {
		initialPlayer := postgres.InGamePlayer{
			LobbyID:        "LOBBY1",
			Username:       "admin",
			PlayersMoney:   1000,
			MostPlayedHand: datatypes.JSON([]byte(`{"rock": 0, "paper": 0, "scissors": 0}`)),
			Winner:         false,
		}
		if err := db.Create(&initialPlayer).Error; err != nil {
			return fmt.Errorf("failed to create initial in-game player: %w", err)
		}
		log.Println("Created initial in-game player")
	}

	// Create initial Friendship
	var friendshipCount int64
	db.Model(&postgres.Friendship{}).Count(&friendshipCount)
	if friendshipCount == 0 {
		initialFriendship := postgres.Friendship{
			Username1: "admin",
			Username2: "player2",
		}
		if err := db.Create(&initialFriendship).Error; err != nil {
			return fmt.Errorf("failed to create initial friendship: %w", err)
		}
		log.Println("Created initial friendship")
	}

	// Create initial FriendshipRequest
	var requestCount int64
	db.Model(&postgres.FriendshipRequest{}).Count(&requestCount)
	if requestCount == 0 {
		initialRequest := postgres.FriendshipRequest{
			Username1: "player2",
			Username2: "admin",
			CreatedAt: time.Now(),
		}
		if err := db.Create(&initialRequest).Error; err != nil {
			return fmt.Errorf("failed to create initial friendship request: %w", err)
		}
		log.Println("Created initial friendship request")
	}

	// Create initial GameInvitation
	var invitationCount int64
	db.Model(&postgres.GameInvitation{}).Count(&invitationCount)
	if invitationCount == 0 {
		initialInvitation := postgres.GameInvitation{
			LobbyID:         "LOBBY1",
			InvitedUsername: "player2",
			CreatedAt:       time.Now(),
		}
		if err := db.Create(&initialInvitation).Error; err != nil {
			return fmt.Errorf("failed to create initial game invitation: %w", err)
		}
		log.Println("Created initial game invitation")
	}

	return nil
}
