package main

import (
	_ "Nogler/docs"
	"Nogler/redis"
	"Nogler/routes"
	"database/sql"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	//Setup DB conn
	pgConnStr := os.Getenv("DB_CONN_STR")
	if pgConnStr == "" {
		pgConnStr = "postgresql://nogler:nogler@localhost:5432/nogler?sslmode=disable"
	}

	//Start communication with Postgre DB
	db, err := sql.Open("postgres", pgConnStr)
	if err != nil {
		log.Fatalf("Error connecting to PostgreSQL: %v", err)
	}
	defer db.Close()

	// Verify connection to PostgreSQL
	if err := db.Ping(); err != nil {
		log.Fatalf("Error making ping to PostgreSQL: %v", err)
	}
	log.Println("PostgreSQL connection established")

	// Configure connection to Redis
	redisUri := os.Getenv("REDIS_URL")
	if redisUri == "" {
		redisUri = "localhost:6379"
	}

	redisClient, err := redis.InitRedis(redisUri, 0)
	if err != nil {
		log.Fatalf("Error connecting to Redis: %v", err)
	}
	defer redis.CloseRedis(redisClient)
	log.Println("Redis connection established")

	r := gin.Default()

	routes.SetupRoutes(r, db, redisClient)
	// Configure port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	// Start server
	log.Printf("Server started on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
