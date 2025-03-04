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

	// Create DSN (Data Source Name)
	// "host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable TimeZone=Asia/Shanghai"
	/*dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
	host, user, password, database, port)*/

	// Connect to PostgreSQL with GORM
	// driver := pgdriver.Open(dsn)
	// fmt.Println("Opened dsn")

	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s",
		user, password, host, port, database)

	/* db, err := gorm.Open(driver, &gorm.Config{}) */
	sqlDB1, err := sql.Open("postgres", dsn)

	if err != nil {
		log.Printf("Error connecting to PostgreSQL: %v", err)
		return nil, err
	}

	fmt.Println("Opened pgx")

	// Verify connection to PostgreSQL
	if err := sqlDB1.Ping(); err != nil {
		log.Printf("Error making ping to PostgreSQL: %v", err)
		return nil, err
	}

	fmt.Println("Pinged PostgreSQL")

	db, err := gorm.Open(pgdriver.New(pgdriver.Config{
		Conn: sqlDB1,
	}), &gorm.Config{})

	fmt.Println("Opened GORM")

	if err != nil {
		log.Printf("Error connecting to PostgreSQL with GORM: %v", err)
		return nil, err
	}

	log.Println("Opened GORM driver")

	// Get the underlying SQL DB object
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Error getting underlying SQL DB: %v", err)
		return nil, err
	}

	log.Println("Retrieved sql DB")

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

type Product struct {
	Code  string
	Price uint
}

func MigrateDatabase(db *gorm.DB) error {
	// List all your models here
	all_models := AllModels()

	for _, model := range all_models {
		modelName := reflect.TypeOf(model).Elem().Name()
		log.Printf("Dropping table for model: %s", modelName)

		if err := db.Migrator().DropTable(model); err != nil {
			return fmt.Errorf("failed to drop table for %s: %w", modelName, err)
		}
	}

	// err := db.AutoMigrate(all_models)

	db.Exec("SET CONSTRAINTS ALL DEFERRED")
	db.AutoMigrate(&postgres.User{})
	db.AutoMigrate(&postgres.GameProfile{})
	db.AutoMigrate(&postgres.Friendship{})
	db.AutoMigrate(&postgres.FriendshipRequest{})
	db.AutoMigrate(&postgres.GameLobby{})
	db.AutoMigrate(&postgres.InGamePlayer{})
	err := db.AutoMigrate(&postgres.GameInvitation{})
	db.Exec("SET CONSTRAINTS ALL IMMEDIATE")

	/*db.Migrator().DropTable(&Product{})
	err := db.AutoMigrate(&Product{})*/

	if err != nil {
		return fmt.Errorf("auto migration failed: %w", err)
	}

	return nil
}
