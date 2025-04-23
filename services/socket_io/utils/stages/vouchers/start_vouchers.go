package vouchers

import (
	"Nogler/services/redis"
	socketio_types "Nogler/services/socket_io/types"
	"Nogler/utils"
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// MulticastStartingVouchers sends a notification to all players in the lobby
// that the vouchers phase has begun
func MulticastStartingVouchers(sio *socketio_types.SocketServer, redisClient *redis.RedisClient, db *gorm.DB, lobbyID string, timeout int) {
	log.Printf("[VOUCHER-MULTICAST] Sending vouchers phase start event for lobby %s", lobbyID)

	// Get the game lobby from Redis
	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[VOUCHERS-ERROR] Error getting lobby: %v", err)
		return
	}

	// Get all players in the lobby
	players, err := redisClient.GetAllPlayersInLobby(lobbyID)
	if err != nil {
		log.Printf("[VOUCHERS-ERROR] Error getting players: %v", err)
		return
	}

	// Prepare the users_in_lobby array with username and icon
	usersInLobby := make([]gin.H, 0, len(players))
	for _, player := range players {
		// Get the player icon from PostgreSQL
		icon := utils.UserIcon(db, player.Username)

		usersInLobby = append(usersInLobby, gin.H{
			"username": player.Username,
			"icon":     icon,
		})
	}

	// Prepare and send the voucher information to each player
	for _, player := range players {
		// Extract player's modifiers as vouchers
		var modifiers json.RawMessage
		if player.Modifiers != nil {
			modifiers = player.Modifiers
		} else {
			modifiers = json.RawMessage(`[]`)
		}

		// Get the player's socket connection
		socket, exists := sio.GetConnection(player.Username)
		if exists && socket != nil {
			// Emit personalized event to this player
			socket.Emit("starting_vouchers", gin.H{
				"vouchers":           modifiers,
				"timeout":            timeout,
				"timeout_start_date": lobby.VouchersTimeout.Format(time.RFC3339),
				"current_round":      lobby.CurrentRound,
				"users_in_lobby":     usersInLobby,
			})
		}
	}

	// Also broadcast to the lobby room for any observers
	// TODO: gpt suggestion, creo que sobra socio
	/*sio.Sio_server.To(socket.Room(lobbyID)).Emit("voucher_phase_started", gin.H{
		"timeout_seconds":    timeout,
		"current_round":      lobby.CurrentRound,
		"users_in_lobby":     usersInLobby,
		"timeout_start_date": lobby.VouchersTimeout,
		"message":            "Starting the vouchers phase!",
	})*/

	log.Printf("[VOUCHER-MULTICAST] Vouchers phase start multicast sent to lobby %s", lobbyID)
}
