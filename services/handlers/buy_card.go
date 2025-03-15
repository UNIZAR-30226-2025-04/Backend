package handlers

import (
	"Nogler/utils"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

func BuyCardHandler(client *socket.Socket, db *gorm.DB, args ...interface{}) {
	cardID, ok := args[0].(string)
	if !ok {
		client.Emit("error", gin.H{"error": "Invalid card ID"})
		return
	}

	username, err := utils.GetUsernameFromClient(client)
	if err != nil {
		client.Emit("error", gin.H{"error": err.Error()})
		return
	}

	fmt.Println(username, "bought card:", cardID)
	client.Emit("buy_card_success", gin.H{"message": "Card bought successfully", "card_id": cardID})
}
