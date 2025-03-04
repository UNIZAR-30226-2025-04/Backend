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

	// Setup DB conn

	// NOTE: still keeping old raw sql DB instantiation commented
	/* db, err := config.Connect_postgres() */
	gormDB, err := config.ConnectGORM()
	if err != nil {
		log.Fatalf("Error connecting to PostgreSQL: %v", err)
	}

	// Only migrate in development or during deployment
	//if os.Getenv("ENVIRONMENT") == "development" {
	if err := config.MigrateDatabase(gormDB); err != nil {
		log.Printf("Warning: Database migration failed: %v", err)
		// Continue execution even if migration fails
	}
	//}

	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("Error reading GORM PostgreSQL instance: %v", err)
	}
	/* defer db.Close() */
	defer sqlDB.Close()

	// Connect to Redis
	/*redisClient, err := config.Connect_redis()
	if err != nil {
		log.Fatalf("Error connecting to Redis: %v", err)
	}
	defer redis.CloseRedis(redisClient)*/

	r := gin.Default()

	/* routes.SetupRoutes(r, db, redisClient) */
	routes.SetupRoutes(r, gormDB /*redisClient*/)

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
