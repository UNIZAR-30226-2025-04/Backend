package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

func BuyModifierHandler(client *socket.Socket, db *gorm.DB, args ...interface{}) {
	fmt.Println("Buying a modifier...")
	client.Emit("buy_modifier_success", gin.H{"message": "You bought a modifier"})
}
