package config

import (
	"Nogler/models/postgres"
	"database/sql"
	"fmt"
	"log"
	"os"
	"reflect"
	"time"

	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level (Silent, Error, Warn, Info)
			IgnoreRecordNotFoundError: false,       // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Enable color
		},
	)
	db, err := gorm.Open(pgdriver.New(pgdriver.Config{
		Conn: sqlDB1,
	}), &gorm.Config{
		Logger: newLogger, // Add the logger to the configuration
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

	all_models := AllModels()

	for _, model := range all_models {
		modelName := reflect.TypeOf(model).Elem().Name()
		log.Printf("Dropping table for model: %s", modelName)

		if err := db.Migrator().DropTable(model); err != nil {
			return fmt.Errorf("failed to drop table for %s: %w", modelName, err)
		}
	}

	err := db.AutoMigrate(
		&postgres.GameProfile{},
		&postgres.User{},
		&postgres.Friendship{},
		&postgres.FriendshipRequest{},
		&postgres.GameLobby{},
		&postgres.InGamePlayer{},
		&postgres.GameInvitation{})

	if err != nil {
		return fmt.Errorf("auto migration failed: %w", err)
	}
	log.Println("Database migrated successfully")

	return nil
}
