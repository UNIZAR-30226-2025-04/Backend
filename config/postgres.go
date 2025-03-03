package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"
)

// Connect_postgres returns a connection string for PostgreSQL
func Connect_postgres() (*sql.DB, error) {
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
}
