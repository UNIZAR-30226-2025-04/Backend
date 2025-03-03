package main

import (
	"Nogler/config"
	_ "Nogler/config/swagger"
	"Nogler/routes"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {

	//Setup DB conn

	db, err := config.Connect_postgres()
	if err != nil {
		log.Fatalf("Error connecting to PostgreSQL: %v", err)
	}
	defer db.Close()

	// Configure connection to Redis
	redisUri := os.Getenv("REDIS_URL")
	if redisUri == "" {
		redisUri = "localhost:6379"
	}

	// Connect to Redis
	redisClient, err := config.Connect_redis()
	if err != nil {
		log.Fatalf("Error connecting to Redis: %v", err)
	}
	defer redisClient.Close()

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
