package main

import (
	"Nogler/api/routes"
	"Nogler/redis"
	"database/sql"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	// COnfigure connection to PostgreSQL
	dbConnStr := os.Getenv("DB_CONN_STR")
	if dbConnStr == "" {
		dbConnStr = "postgresql://nogler:nogler@localhost:5432/nogler?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbConnStr)
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
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisDB := 0
	redisClient, err := redis.InitRedis(redisAddr, redisDB)
	if err != nil {
		log.Fatalf("Error connecting to Redis: %v", err)
	}
	defer redis.CloseRedis(redisClient)
	log.Println("Redis connection established")

	// Start Gin
	router := gin.Default()

	// Configure routes
	routes.SetupRoutes(router, db, redisClient)

	// Configure port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	log.Printf("Server started on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
