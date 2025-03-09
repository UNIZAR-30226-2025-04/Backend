package controllers_test

import (
	"testing"
)

// SetupLobbyTestData ensures all necessary test users exist in the database
func SetupLobbyTestData(t *testing.T) {
    // Reuse the friends setup that already creates the necessary users
    SetupFriendsTestData(t)
}
