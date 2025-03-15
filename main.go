package main

import (
	"Nogler/services/redis"
	"Nogler/services/socket_io"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

// @title Nogler API
// @version 1.0
// @description Gin-Gonic server for the "Nogler" game API
// @host nogler.ddns.net:443
// @BasePath /
// @paths
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)  // Añadir número de línea a los logs
	log.Println("[MAIN] Setting up server...")

	if os.Getenv("PROD") == "true" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Inicializar Redis directamente
	redisClient := redis.NewRedisClient("localhost:6379", 0)
	if redisClient == nil {
		log.Fatal("[MAIN] Error inicializando Redis")
	}
	log.Println("[MAIN] Connection to Redis successful")
	defer redis.CloseRedis(redisClient)

	r := gin.Default()

	log.Println("[MAIN] Iniciando configuración de Socket.IO...")
	// NEW: socket.io setup
	var sio socket_io.SocketServer
	sio.Start(r, nil, redisClient)  // Pasamos nil como db para activar el modo test

	port := "8080"
	log.Printf("[MAIN] Server starting on port %s", port)
	
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("[MAIN] Error starting server: %v", err)
	}
}
