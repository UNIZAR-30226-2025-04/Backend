package main

import (
	_ "Nogler/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Nogler API
// @version 1.0
// @description Gin-Gonic server for the "Nogler" game API
// @host localhost:8080
// @BasePath /
// @paths

// localhost only for now, logically it should be the server's IP

// @Summary Endpoint just pings the server
// @Description Returns a basic message
// @Tags test
// @Produce json
// @Success 200 {object} string
// @Router /ping [get]
func main() {
	r := gin.Default()

	// Swagger route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Testing a basic endpoint, and the auto-docs

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong, hola"})
	})

	r.Run(":8080") // Endpoint on port 8080 for now, test
}
