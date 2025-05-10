package utils

/**
 * This file contains utility functions to format the keys for Redis
 * (key, value) pairs. It avoids having to call "fmt.Sprintf(...)"
 * with the same format spec every time, potentially confusing the key format.
 */

import "fmt"

func FormatInGamePlayerKey(username string) string {
	return fmt.Sprintf("player:%s:game", username)
}

func FormatLobbyKey(lobbyId string) string {
	return fmt.Sprintf("lobby:%s", lobbyId)
}

func FormatPackKey(lobbyId string, currentRound int, itemId int) string {
	return fmt.Sprintf("lobby:%s:round:%d:item_id:%d", lobbyId, currentRound, itemId)
}
