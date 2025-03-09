package controllers_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type Friend struct {
	Username string `json:"username"`
	Icon     int    `json:"icon"`
}

func TestListFriends(t *testing.T) {
	// Client with timeout
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	baseURL := "https://nogler.ddns.net:443"
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJFbWFpbCI6ImpvcmRpQGdtYWlsLmNvbSJ9.ILJUkEuioZWRMkLADnERrO0JfGPiwhf5PQPpnIOEnps"

	t.Run("List friends successfully", func(t *testing.T) {
		// Request to list friends
		req, err := http.NewRequest(http.MethodGet, baseURL+"/auth/friends", nil)
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Error haciendo la petición: %v", err)
		}
		defer resp.Body.Close()

		// Verify if the response status code is 200
		if resp.StatusCode != http.StatusOK {
			var errorResp struct {
				Error string `json:"error"`
			}
			json.NewDecoder(resp.Body).Decode(&errorResp)
			t.Fatalf("Expected status code 200, received %d. Error: %s", resp.StatusCode, errorResp.Error)
		}

		var friends []Friend
		err = json.NewDecoder(resp.Body).Decode(&friends)
		assert.NoError(t, err, "Error decoding the response")

		// Verify if the response is what we expect
		expectedFriends := []Friend{
			{Username: "Nico", Icon: 0},
			{Username: "yago", Icon: 999},
		}

		assert.Equal(t, expectedFriends, friends, "The friends list does not match the expected one")
	})

	t.Run("Without authorization token", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, baseURL+"/auth/friends", nil)
		assert.NoError(t, err)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid token", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, baseURL+"/auth/friends", nil)
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer token_invalido")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
