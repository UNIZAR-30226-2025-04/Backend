package controllers

import (
	"github.com/gin-gonic/gin"
)

// @Summary Endpoint just pings the server
// @Description Returns a basic message
// @Tags test
// @Produce json
// @Success 200 {object} object{message=string}
// @Router /ping [get]
func Ping(c *gin.Context) {
	c.JSON(200, gin.H{"message": "pong, hola"})
}
