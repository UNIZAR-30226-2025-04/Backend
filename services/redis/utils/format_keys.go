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

func FormatPackKey(lobbyId string, maxRounds int, rerrolls int, itemId int, currentRound int) string {
	return fmt.Sprintf("lobby:%s:round:%d:reroll:%d:pack:%d:current:%d", lobbyId, maxRounds, rerrolls, itemId, currentRound)
}
