package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

func BuyJokerHandler(client *socket.Socket, db *gorm.DB, args ...interface{}) {
	fmt.Println("Buying a joker...")
	client.Emit("buy_joker_success", gin.H{"message": "You bought a joker"})
}
