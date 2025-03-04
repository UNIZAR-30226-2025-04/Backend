package config

import (
	"Nogler/models/postgres"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq" // Fixed an error
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ConnectGORM returns a GORM DB instance connected to PostgreSQL
func ConnectGORM() (*gorm.DB, error) {
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	database := os.Getenv("POSTGRES_DATABASE")
	verbose := os.Getenv("VERBOSE_POSTGRES")

	// NOTE: https://stackoverflow.com/questions/57205060/how-to-connect-postgresql-database-using-gorm
	// NOTE: See https://github.com/go-gorm/gorm/issues/5409
	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s",
		user, password, host, port, database)

	sqlDB1, err := sql.Open("postgres", dsn)

	if err != nil {
		log.Printf("Error connecting to PostgreSQL: %v", err)
		return nil, err
	}

	sqlDB1.Exec("SELECT * FROM USERS;")

	gormConfig := &gorm.Config{}
	if verbose == "true" {
		newLogger := logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold:             time.Second, // Slow SQL threshold
				LogLevel:                  logger.Info, // Log level (Silent, Error, Warn, Info)
				IgnoreRecordNotFoundError: false,       // Ignore ErrRecordNotFound error for logger
				Colorful:                  true,        // Enable color
			},
		)
		gormConfig.Logger = newLogger
	}

	db, err := gorm.Open(pgdriver.New(pgdriver.Config{
		Conn:                 sqlDB1,
		PreferSimpleProtocol: true,
	}), gormConfig)

	if err != nil {
		log.Printf("Error connecting to PostgreSQL with GORM: %v", err)
		return nil, err
	}

	// Get the underlying SQL DB object
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Error getting underlying SQL DB: %v", err)
		return nil, err
	}

	// Verify connection
	if err := sqlDB.Ping(); err != nil {
		log.Printf("Error pinging PostgreSQL: %v", err)
		return nil, err
	}

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

// MigrateDatabase migrates the GORM models to the PostgreSQL database
func MigrateDatabase(db *gorm.DB) error {
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
	log.Println("PostgreSQL database migrated successfully")

	return nil
}
