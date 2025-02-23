package sync

import (
	"Nogler/redis"
	"database/sql"
	"fmt"
)

type SyncManager struct {
	redisClient *redis.RedisClient
	db          *sql.DB
}

// NewSyncManager creates a new instance of the synchronization manager
func NewSyncManager(redisClient *redis.RedisClient, db *sql.DB) *SyncManager {
	return &SyncManager{
		redisClient: redisClient,
		db:          db,
	}
}

// SyncPlayerGameState synchronizes the player state from Redis to PostgreSQL
func (sm *SyncManager) SyncPlayerGameState(username string, lobbyId string) error {
	// Get player state from Redis
	player, err := sm.redisClient.GetInGamePlayer(username)
	if err != nil {
		return fmt.Errorf("error getting player state from Redis: %v", err)
	}

	// Update in_game_players
	playerQuery := `
		UPDATE in_game_players 
		SET 
			players_money = $1,
			most_played_hand = $2
		WHERE username = $3 AND lobby_id = $4
	`

	_, err = sm.db.Exec(playerQuery,
		player.PlayersMoney,
		player.MostPlayedHand,
		username,
		lobbyId)

	if err != nil {
		return fmt.Errorf("error updating player state in PostgreSQL: %v", err)
	}

	return nil
}

// SyncLobbyState synchronizes the lobby state and its players
func (sm *SyncManager) SyncLobbyState(lobbyId string) error {
	// Get lobby state from Redis
	lobby, err := sm.redisClient.GetGameLobby(lobbyId)
	if err != nil {
		return fmt.Errorf("error getting lobby state from Redis: %v", err)
	}

	// Start transaction
	tx, err := sm.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback()

	// Update game_lobbies
	lobbyQuery := `
		UPDATE game_lobbies 
		SET 
			number_of_rounds = $1,
			total_points = $2
		WHERE id = $3
	`

	_, err = tx.Exec(lobbyQuery,
		lobby.NumberOfRounds,
		lobby.TotalPoints,
		lobbyId)

	if err != nil {
		return fmt.Errorf("error updating lobby state in PostgreSQL: %v", err)
	}

	// Confirm transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	return nil
}

// CleanupGameData synchronizes the final state and cleans Redis
func (sm *SyncManager) CleanupGameData(lobbyId string) error {
	// Get lobby state from Redis
	_, err := sm.redisClient.GetGameLobby(lobbyId)
	if err != nil {
		return fmt.Errorf("error getting lobby state from Redis: %v", err)
	}

	// Start transaction
	tx, err := sm.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback()

	// Sync final state for each player
	// TODO: Implement logic to get lobby players list in the future
	players := []string{} // Get players list
	for _, username := range players {
		if err := sm.SyncPlayerGameState(username, lobbyId); err != nil {
			return fmt.Errorf("error syncing final player state: %v", err)
		}
	}

	// Sync final lobby state
	if err := sm.SyncLobbyState(lobbyId); err != nil {
		return fmt.Errorf("error syncing final lobby state: %v", err)
	}

	// Clean Redis data
	// TODO: Implement Redis data cleanup in the future
	
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error committing final state: %v", err)
	}

	return nil
} 