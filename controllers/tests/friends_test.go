package controllers_test

/*
This test file contains integration tests for friend-related functionality:

TestListFriends:
- Tests listing friends successfully with valid credentials
- Tests behavior when no authorization token is provided
- Tests behavior with an invalid token

TestAddFriend:
- Tests adding a friend successfully
- Tests adding a non-existent friend
- Tests attempting to add yourself as a friend

TestDeleteFriend:
- Tests deleting a friend successfully
- Tests deleting a non-existent friend

TestSendFriendshipRequest:
- Tests sending a friendship request successfully
- Tests sending a request to a non-existent user
- Tests sending a request to yourself
- Tests sending a request to an existing friend
*/

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Friend represents the structure of a friend in the response
type Friend struct {
	Username string `json:"username"`
	Icon     int    `json:"icon"`
}

// SetupFriendsTestData ensures all necessary test users exist in the database
func SetupFriendsTestData(t *testing.T) {
	// Primero limpiar amistades existentes
	cleanupFriendships(t)
	
	// Luego crear usuarios de prueba
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	baseURL := "https://nogler.ddns.net:443"

	// Create test users if they don't exist
	testUsers := []struct {
		username string
		email    string
		password string
		icon     string
	}{
		{"pepito", "pepito@test.com", "password123", "34"},
		{"Nico", "nico@test.com", "password123", "0"},
		{"yago", "yago@test.com", "password123", "999"},
	}

	for _, user := range testUsers {
		formData := url.Values{}
		formData.Set("username", user.username)
		formData.Set("email", user.email)
		formData.Set("password", user.password)
		formData.Set("icono", user.icon)

		req, err := http.NewRequest(http.MethodPost, baseURL+"/signup", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		resp.Body.Close()

		// Ignore error if user already exists (409 Conflict)
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
			t.Fatalf("Failed to create test user %s: %d", user.username, resp.StatusCode)
		}
	}
}

func cleanupFriendships(t *testing.T) {
	client := &http.Client{Timeout: time.Second * 10}
	baseURL := "https://nogler.ddns.net:443"
	
	// Obtener token fresco
	token := getNewToken(t)
	
	// Eliminar todas las amistades existentes
	req, _ := http.NewRequest(http.MethodDelete, baseURL+"/auth/friends/all", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	client.Do(req)
}

func getNewToken(t *testing.T) string {
	return "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJFbWFpbCI6ImpvcmRpQGdtYWlsLmNvbSJ9.ILJUkEuioZWRMkLADnERrO0JfGPiwhf5PQPpnIOEnps"
}

// TestListFriends tests all friend listing scenarios
func TestListFriends(t *testing.T) {
	SetupFriendsTestData(t)
	// Setup HTTP client with timeout
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	baseURL := "https://nogler.ddns.net:443"
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJFbWFpbCI6ImpvcmRpQGdtYWlsLmNvbSJ9.ILJUkEuioZWRMkLADnERrO0JfGPiwhf5PQPpnIOEnps"

	// Test successful friend list retrieval
	t.Run("List friends successfully", func(t *testing.T) {
		// Create and configure request
		req, err := http.NewRequest(http.MethodGet, baseURL+"/auth/friends", nil)
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		// Send request and handle response
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Error making the request: %v", err)
		}
		defer resp.Body.Close()

		// Verify successful response
		if resp.StatusCode != http.StatusOK {
			var errorResp struct {
				Error string `json:"error"`
			}
			json.NewDecoder(resp.Body).Decode(&errorResp)
			t.Fatalf("Expected status code 200, received %d. Error: %s", resp.StatusCode, errorResp.Error)
		}

		// Parse and verify response content
		var friends []Friend
		err = json.NewDecoder(resp.Body).Decode(&friends)
		assert.NoError(t, err, "Error decoding the response")

		// Compare with expected friends list
		expectedFriends := []Friend{
			{Username: "Nico", Icon: 0},
			{Username: "yago", Icon: 999},
		}

		assert.Equal(t, expectedFriends, friends, "The friends list does not match the expected one")
	})

	// Test request without authorization token
	t.Run("Without authorization token", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, baseURL+"/auth/friends", nil)
		assert.NoError(t, err)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// Test request with invalid token
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

// TestAddFriend tests all friend addition scenarios
func TestAddFriend(t *testing.T) {
	SetupFriendsTestData(t)
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	baseURL := "https://nogler.ddns.net:443"
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJFbWFpbCI6ImpvcmRpQGdtYWlsLmNvbSJ9.ILJUkEuioZWRMkLADnERrO0JfGPiwhf5PQPpnIOEnps"

	// Test successful friend addition
	t.Run("Add friend successfully", func(t *testing.T) {
		// Prepare form data for request
		formData := url.Values{}
		formData.Set("friendUsername", "pepito")

		// Create and configure request
		req, err := http.NewRequest(http.MethodPost, baseURL+"/auth/addFriend", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		// Send request and verify response
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Check successful status code
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify success message
		var response struct {
			Message string `json:"message"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "Friend added successfully", response.Message)

		// Verify friend appears in friends list
		req, err = http.NewRequest(http.MethodGet, baseURL+"/auth/friends", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err = client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		var friends []Friend
		err = json.NewDecoder(resp.Body).Decode(&friends)
		assert.NoError(t, err)

		// Check if added friend exists in list
		found := false
		for _, friend := range friends {
			if friend.Username == "pepito" && friend.Icon == 34 {
				found = true
				break
			}
		}
		assert.True(t, found, "The added friend does not appear in the list")

		// Cleanup: Delete the friend we just added
		req, err = http.NewRequest(http.MethodDelete, baseURL+"/auth/deleteFriend/pepito", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err = client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test adding non-existent friend
	t.Run("Add friend that does not exist", func(t *testing.T) {
		formData := url.Values{}
		formData.Set("friendUsername", "usuario_inexistente")

		req, err := http.NewRequest(http.MethodPost, baseURL+"/auth/addFriend", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.NotEqual(t, http.StatusOK, resp.StatusCode, "Expected a different error code than 200")
	})

	// Test adding yourself as friend
	t.Run("Add yourself as a friend", func(t *testing.T) {
		formData := url.Values{}
		formData.Set("friendUsername", "Jordi")

		req, err := http.NewRequest(http.MethodPost, baseURL+"/auth/addFriend", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestDeleteFriend tests all friend deletion scenarios
func TestDeleteFriend(t *testing.T) {
	SetupFriendsTestData(t)
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	baseURL := "https://nogler.ddns.net:443"
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJFbWFpbCI6ImpvcmRpQGdtYWlsLmNvbSJ9.ILJUkEuioZWRMkLADnERrO0JfGPiwhf5PQPpnIOEnps"

	// Test successful friend deletion
	t.Run("Delete friend successfully", func(t *testing.T) {
		// First add a friend to ensure they exist
		formData := url.Values{}
		formData.Set("friendUsername", "pepito")

		req, err := http.NewRequest(http.MethodPost, baseURL+"/auth/addFriend", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Now delete the friend
		req, err = http.NewRequest(http.MethodDelete, baseURL+"/auth/deleteFriend/pepito", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err = client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response struct {
			Message string `json:"message"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "Friend removed successfully", response.Message)

		// Verify friend no longer appears in friends list
		req, err = http.NewRequest(http.MethodGet, baseURL+"/auth/friends", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err = client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		var friends []Friend
		err = json.NewDecoder(resp.Body).Decode(&friends)
		assert.NoError(t, err)

		// Check that deleted friend does not exist in list
		found := false
		for _, friend := range friends {
			if friend.Username == "pepito" {
				found = true
				break
			}
		}
		assert.False(t, found, "The deleted friend still appears in the list")
	})

	// Test deleting non-existent friend
	t.Run("Delete non-existent friend", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, baseURL+"/auth/deleteFriend/usuario_inexistente", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var response struct {
			Error string `json:"error"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "Friendship does not exist", response.Error)
	})
}

// TestSendFriendshipRequest tests all friendship request scenarios
func TestSendFriendshipRequest(t *testing.T) {
	SetupFriendsTestData(t)
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	baseURL := "https://nogler.ddns.net:443"
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJFbWFpbCI6ImpvcmRpQGdtYWlsLmNvbSJ9.ILJUkEuioZWRMkLADnERrO0JfGPiwhf5PQPpnIOEnps"

	// Test successful friendship request
	t.Run("Send friendship request successfully", func(t *testing.T) {
		formData := url.Values{}
		formData.Set("friendUsername", "pepito")

		req, err := http.NewRequest(http.MethodPost, baseURL+"/auth/sendFriendshipRequest", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response struct {
			Message string `json:"message"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "Friend request sent successfully", response.Message)

		// Cleanup: Delete the sent request
		req, err = http.NewRequest(http.MethodDelete, baseURL+"/auth/sent_friendship_request/pepito", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err = client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test sending request to non-existent user
	t.Run("Send request to non-existent user", func(t *testing.T) {
		formData := url.Values{}
		formData.Set("friendUsername", "usuario_inexistente")

		req, err := http.NewRequest(http.MethodPost, baseURL+"/auth/sendFriendshipRequest", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var response struct {
			Error string `json:"error"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "Receiver user not found", response.Error)
	})

	// Test sending request to yourself
	t.Run("Send request to yourself", func(t *testing.T) {
		formData := url.Values{}
		formData.Set("friendUsername", "Jordi")

		req, err := http.NewRequest(http.MethodPost, baseURL+"/auth/sendFriendshipRequest", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var response struct {
			Error string `json:"error"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "You cannot send a friend request to yourself", response.Error)
	})

	// Test sending request to someone who's already a friend
	t.Run("Send request to existing friend", func(t *testing.T) {
		// First add the friend
		formData := url.Values{}
		formData.Set("friendUsername", "pepito")

		req, err := http.NewRequest(http.MethodPost, baseURL+"/auth/addFriend", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Now try to send a friend request
		req, err = http.NewRequest(http.MethodPost, baseURL+"/auth/sendFriendshipRequest", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err = client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var response struct {
			Error string `json:"error"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "You are already friends", response.Error)

		// Cleanup: Delete the friend
		req, err = http.NewRequest(http.MethodDelete, baseURL+"/auth/deleteFriend/pepito", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err = client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
	})
}
