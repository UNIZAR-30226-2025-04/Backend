package config

import (
	"Nogler/models/postgres"
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"time"

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

	// Verify connection to PostgreSQL
	if err := sqlDB1.Ping(); err != nil {
		log.Printf("Error making ping to PostgreSQL: %v", err)
		return nil, err
	}

	db, err := gorm.Open(pgdriver.New(pgdriver.Config{
		Conn: sqlDB1,
	}), &gorm.Config{
		//DisableForeignKeyConstraintWhenMigrating: true,
	})

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

	return nil
}
