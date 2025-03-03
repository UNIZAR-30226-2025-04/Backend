package main

import (
	"Nogler/config"
	_ "Nogler/config/swagger"
	"Nogler/redis"
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

	// Connect to Redis
	redisClient, err := config.Connect_redis()
	if err != nil {
		log.Fatalf("Error connecting to Redis: %v", err)
	}
	defer redis.CloseRedis(redisClient)

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
