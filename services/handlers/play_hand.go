package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

func PlayHandHandler(client *socket.Socket, db *gorm.DB, args ...interface{}) {
	fmt.Println("Playing a hand...")
	client.Emit("play_hand_success", gin.H{"message": "You played a hand"})
}
