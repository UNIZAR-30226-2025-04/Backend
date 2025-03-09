package main

import (
	"Nogler/config"
	pgconfig "Nogler/config/postgres"
	_ "Nogler/config/swagger"
	"Nogler/middleware"
	"Nogler/routes"
	"Nogler/services/redis"
	"Nogler/services/socket_io"
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

	if os.Getenv("PROD") == "true" {
		gin.SetMode(gin.ReleaseMode)
	}

	gormDB, err := pgconfig.ConnectGORM()
	if err != nil {
		log.Fatalf("Error connecting to PostgreSQL: %v", err)
	}
	log.Println("GORM Connected")

	// Only migrate in development or during deployment
	if os.Getenv("MIGRATE_POSTGRES") == "true" {
		log.Println("Migrating PostgreSQL database...")
		if err := pgconfig.MigrateDatabase(gormDB); err != nil {
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

	// NEW: socket.io setup
	var sio socket_io.SocketServer
	sio.Start(r, gormDB)

	// Configure port
	port := os.Getenv("PORT")
	log.Println("Puerto leido: ", port)
	if port == "443" && os.Getenv("USE_HTTPS") == "true" {
		port = "443"
	} else {
		port = "8080"
	}
	log.Println("Puerto =", port)

	if os.Getenv("USE_HTTPS") == "true" {
		log.Println("Using HTTPS on port 443")
		//SSL certification configuration for HTTPS
		certFile := os.Getenv("FULLCHAIN_PATH")
		keyFile := os.Getenv("KEY_PATH")

		// Start server
		if err := r.RunTLS(":"+port, certFile, keyFile); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
	} else {
		log.Println("Using HTTP on port 8080")
		if err := r.Run(":" + port); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
	}
	log.Printf("Server started on port %s", port)
}
