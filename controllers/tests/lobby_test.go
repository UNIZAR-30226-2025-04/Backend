package controllers_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// SetupLobbyTestData ensures all necessary test users exist in the database
func SetupLobbyTestData(t *testing.T) {
    // Reuse the friends setup that already creates the necessary users
    SetupFriendsTestData(t)
}

// TestCreateLobby tests all scenarios for creating a lobby
func TestCreateLobby(t *testing.T) {
    SetupLobbyTestData(t)
    client := &http.Client{
        Timeout: time.Second * 10,
    }
    baseURL := "https://nogler.ddns.net:443"
    token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJFbWFpbCI6ImpvcmRpQGdtYWlsLmNvbSJ9.ILJUkEuioZWRMkLADnERrO0JfGPiwhf5PQPpnIOEnps"

    // Test successful lobby creation
    t.Run("Create lobby successfully", func(t *testing.T) {
        req, err := http.NewRequest(http.MethodPost, baseURL+"/auth/CreateLobby", nil)
        assert.NoError(t, err)
        req.Header.Set("Authorization", "Bearer "+token)
        req.Header.Set("Accept", "application/json")

        resp, err := client.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusOK, resp.StatusCode)

        var response struct {
            LobbyID string `json:"lobby_id"`
            Message string `json:"message"`
        }
        err = json.NewDecoder(resp.Body).Decode(&response)
        assert.NoError(t, err)
        assert.Equal(t, "Lobby created sucessfully", response.Message)
        assert.NotEmpty(t, response.LobbyID)
    })

    // Test without authorization token
    t.Run("Create lobby without authorization", func(t *testing.T) {
        req, err := http.NewRequest(http.MethodPost, baseURL+"/auth/CreateLobby", nil)
        assert.NoError(t, err)
        req.Header.Set("Accept", "application/json")

        resp, err := client.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
    })

    // Test with invalid token
    t.Run("Create lobby with invalid token", func(t *testing.T) {
        req, err := http.NewRequest(http.MethodPost, baseURL+"/auth/CreateLobby", nil)
        assert.NoError(t, err)
        req.Header.Set("Authorization", "Bearer invalid_token")
        req.Header.Set("Accept", "application/json")

        resp, err := client.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
    })
}

func TestGetLobbyInfo(t *testing.T) {
    SetupLobbyTestData(t)
    client := &http.Client{
        Timeout: time.Second * 10,
    }
    baseURL := "https://nogler.ddns.net:443"
    token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJFbWFpbCI6ImpvcmRpQGdtYWlsLmNvbSJ9.ILJUkEuioZWRMkLADnERrO0JfGPiwhf5PQPpnIOEnps"

    // Primero creamos un lobby para obtener su ID
    req, err := http.NewRequest(http.MethodPost, baseURL+"/auth/CreateLobby", nil)
    assert.NoError(t, err)
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Accept", "application/json")

    resp, err := client.Do(req)
    assert.NoError(t, err)
    defer resp.Body.Close()

    var createResponse struct {
        LobbyID string `json:"lobby_id"`
        Message string `json:"message"`
    }
    err = json.NewDecoder(resp.Body).Decode(&createResponse)
    assert.NoError(t, err)
    lobbyID := createResponse.LobbyID

    t.Run("Get lobby info successfully", func(t *testing.T) {
        req, err := http.NewRequest(http.MethodGet, baseURL+"/auth/lobbyInfo/"+lobbyID, nil)
        assert.NoError(t, err)
        req.Header.Set("Authorization", "Bearer "+token)
        req.Header.Set("Accept", "application/json")

        resp, err := client.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusOK, resp.StatusCode)

        var response struct {
            LobbyID         string    `json:"lobby_id"`
            CreatorUsername string    `json:"creator_username"`
            NumberRounds    int       `json:"number_rounds"`
            TotalPoints     int       `json:"total_points"`
            CreatedAt       time.Time `json:"created_at"`
        }
        err = json.NewDecoder(resp.Body).Decode(&response)
        assert.NoError(t, err)
        assert.Equal(t, lobbyID, response.LobbyID)
        assert.NotEmpty(t, response.CreatorUsername)
    })

    t.Run("Get non-existent lobby info", func(t *testing.T) {
        req, err := http.NewRequest(http.MethodGet, baseURL+"/auth/lobbyInfo/XXXX", nil)
        assert.NoError(t, err)
        req.Header.Set("Authorization", "Bearer "+token)
        req.Header.Set("Accept", "application/json")

        resp, err := client.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusNotFound, resp.StatusCode)
    })

    t.Run("Get lobby info without authorization", func(t *testing.T) {
        req, err := http.NewRequest(http.MethodGet, baseURL+"/auth/lobbyInfo/"+lobbyID, nil)
        assert.NoError(t, err)
        req.Header.Set("Accept", "application/json")

        resp, err := client.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
    })
}

func TestGetAllLobbies(t *testing.T) {
    SetupLobbyTestData(t)
    client := &http.Client{
        Timeout: time.Second * 10,
    }
    baseURL := "https://nogler.ddns.net:443"
    token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJFbWFpbCI6ImpvcmRpQGdtYWlsLmNvbSJ9.ILJUkEuioZWRMkLADnERrO0JfGPiwhf5PQPpnIOEnps"

    t.Run("Get all lobbies successfully", func(t *testing.T) {
        req, err := http.NewRequest(http.MethodGet, baseURL+"/auth/getAllLobbies", nil)
        assert.NoError(t, err)
        req.Header.Set("Authorization", "Bearer "+token)
        req.Header.Set("Accept", "application/json")

        resp, err := client.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusOK, resp.StatusCode)

        var response []struct {
            LobbyID         string    `json:"lobby_id"`
            CreatorUsername string    `json:"creator_username"`
            NumberRounds    int       `json:"number_rounds"`
            TotalPoints     int       `json:"total_points"`
            CreatedAt       time.Time `json:"created_at"`
        }
        err = json.NewDecoder(resp.Body).Decode(&response)
        assert.NoError(t, err)
    })

    t.Run("Get all lobbies with invalid token", func(t *testing.T) {
        req, err := http.NewRequest(http.MethodGet, baseURL+"/auth/getAllLobbies", nil)
        assert.NoError(t, err)
        req.Header.Set("Authorization", "Bearer invalid_token")
        req.Header.Set("Accept", "application/json")

        resp, err := client.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
    })

    t.Run("Get all lobbies without authorization", func(t *testing.T) {
        req, err := http.NewRequest(http.MethodGet, baseURL+"/auth/getAllLobbies", nil)
        assert.NoError(t, err)
        req.Header.Set("Accept", "application/json")

        resp, err := client.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
    })
}
