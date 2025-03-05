package main

import (
	"Nogler/config"
	_ "Nogler/config/swagger"
	"Nogler/middleware"
	"Nogler/routes"
	"Nogler/services/redis"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// @title Nogler API
// @version 1.0
// @description Gin-Gonic server for the "Nogler" game API
// @host nogler.ddns.net:8080
// @BasePath /
// @paths
func main() {
	godotenv.Load()
	// Setup DB conn
	log.Println("Setting up server...")

	gormDB, err := config.ConnectGORM()
	if err != nil {
		log.Fatalf("Error connecting to PostgreSQL: %v", err)
	}
	log.Println("GORM Connected")

	// Only migrate in development or during deployment
	if os.Getenv("MIGRATE_POSTGRES") == "true" {
		log.Println("Migrating PostgreSQL database...")
		if err := config.MigrateDatabase(gormDB); err != nil {
			log.Printf("Warning: Database migration failed: %v", err)
			// Continue execution even if migration fails
		} else {
			log.Println("Database migrated successfully")
		}
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("Error reading GORM PostgreSQL instance: %v", err)
	}
	defer sqlDB.Close()

	redisClient, err := config.Connect_redis()
	if err != nil {
		log.Fatalf("Error connecting to Redis: %v", err)
	}
	log.Println("Connection to Redis successful")
	defer redis.CloseRedis(redisClient)

	r := gin.Default()

	middleware.SetUpMiddleware(r)

	// TODO: pass in redisClient
	/* routes.SetupRoutes(r, db, redisClient) */
	routes.SetupRoutes(r, gormDB, redisClient)

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
