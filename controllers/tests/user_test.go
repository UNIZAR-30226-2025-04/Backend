package controllers_test

/*
This test file contains integration tests for user-related functionality:

TestSignUp:
- Tests successful user registration
- Tests registration with empty fields
- Tests registration with existing user
- Tests registration with invalid icon

TestLogin:
- Tests successful login
- Tests login with empty fields
- Tests login with invalid email
- Tests login with invalid password

TestLogout:
- Tests successful logout
- Tests logout without token
- Tests logout with invalid token

TestGetAllUsers:
- Tests retrieving all users list with valid token

TestGetUserPublicInfo:
- Tests retrieving public user information successfully
- Tests retrieving non-existent user information
- Tests retrieving user info without authorization

TestGetUserPrivateInfo:
- Tests retrieving private user information successfully
- Tests retrieving private info without authorization
- Tests retrieving private info with invalid token

TestUpdateUserInfo:
- Tests updating user information successfully
- Tests updating with existing username
- Tests updating with invalid icon
- Tests updating without authorization
- Tests updating with invalid token

Helper Functions:
- CleanupTestData: Cleans up test data after tests
- SetupTestUser: Sets up a test user and returns authentication token
*/

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// CleanupTestData deletes the test user account using the provided token
func CleanupTestData(t *testing.T, token string) {
	// Skip if no token provided
	if token == "" {
		return
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	baseURL := "https://nogler.ddns.net:443"

	deleteReq, err := http.NewRequest(http.MethodDelete, baseURL+"/auth/deleteAccount", nil)
	assert.NoError(t, err)
	deleteReq.Header.Set("Authorization", "Bearer "+token)

	deleteResp, err := client.Do(deleteReq)
	if err == nil {
		body, err := io.ReadAll(deleteResp.Body)
		assert.NoError(t, err)
		t.Logf("Complete server response: %s", string(body))
		deleteResp.Body.Close()
	}
}

// TestSignUp verifies user registration functionality
func TestSignUp(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			cleanupAllTestUsers(t)
			panic(err)
		}
	}()
	defer cleanupAllTestUsers(t)
	// Initialize HTTP client with timeout
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	baseURL := "https://nogler.ddns.net:443"

	// Eliminar el usuario de prueba si existe
	loginFormData := url.Values{
		"email":    {"test_signup@example.com"},
		"password": {"testpass123"},
	}

	loginReq, err := http.NewRequest(http.MethodPost, baseURL+"/login", strings.NewReader(loginFormData.Encode()))
	assert.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	loginResp, err := client.Do(loginReq)
	assert.NoError(t, err)
	
	if loginResp.StatusCode == http.StatusOK {
		var loginResponse struct {
			Token string `json:"token"`
		}
		err = json.NewDecoder(loginResp.Body).Decode(&loginResponse)
		loginResp.Body.Close()
		assert.NoError(t, err)
		
		if loginResponse.Token != "" {
			deleteReq, err := http.NewRequest(http.MethodDelete, baseURL+"/auth/me", nil)
			assert.NoError(t, err)
			deleteReq.Header.Set("Authorization", "Bearer "+loginResponse.Token)
			
			deleteResp, err := client.Do(deleteReq)
			assert.NoError(t, err)
			deleteResp.Body.Close()
		}
	}

	t.Run("Sign up successfully", func(t *testing.T) {
		// Generate unique timestamp for test data
		timestamp := time.Now().Format("20060102150405")
		username := "test_signup_user_" + timestamp
		email := "test_signup_" + timestamp + "@example.com"
		
		formData := url.Values{
			"username": {username},
			"email":    {email},
			"password": {"testpass123"},
			"icono":    {"1"},
		}

		req, err := http.NewRequest(http.MethodPost, baseURL+"/signup", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response struct {
			Message string `json:"message"`
			User    struct {
				Username string `json:"username"`
				Email    string `json:"email"`
			} `json:"user"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "User created successfully", response.Message)
		assert.Equal(t, username, response.User.Username)
		assert.Equal(t, email, response.User.Email)

		body, err := io.ReadAll(resp.Body)
		t.Logf("Complete server response: %s", string(body))
	})

	t.Run("Sign up with empty fields", func(t *testing.T) {
		formData := url.Values{
			"username": {""},
			"email":    {""},
			"password": {""},
			"icono":    {"1"},
		}

		req, err := http.NewRequest(http.MethodPost, baseURL+"/signup", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Sign up with existing user", func(t *testing.T) {
		timestamp := time.Now().Format("20060102150405")
		email := "test_existing_" + timestamp + "@example.com"
		formData := url.Values{
			"username": {"test_existing_user_" + timestamp},
			"email":    {email},
			"password": {"testpass123"},
			"icono":    {"1"},
		}

		// Crear usuario y obtener token para limpieza
		req, err := http.NewRequest(http.MethodPost, baseURL+"/signup", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		assert.NoError(t, err)
		resp.Body.Close()

		// Login para obtener token
		loginFormData := url.Values{
			"email":    {email},
			"password": {"testpass123"},
		}
		token := getTokenForCleanup(t, loginFormData)
		defer CleanupTestData(t, token)

		// Intentar crear el mismo usuario de nuevo
		req, err = http.NewRequest(http.MethodPost, baseURL+"/signup", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err = client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("Sign up with invalid icon", func(t *testing.T) {
		formData := url.Values{
			"username": {"test_signup_user2_" + time.Now().Format("20060102150405")},
			"email":    {"test_signup2_" + time.Now().Format("20060102150405") + "@example.com"},
			"password": {"testpass123"},
			"icono":    {"invalid"},
		}

		req, err := http.NewRequest(http.MethodPost, baseURL+"/signup", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should be successful since the code handles invalid icons by assigning the default value 0
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})
}

// Función auxiliar para obtener token
func getTokenForCleanup(t *testing.T, loginFormData url.Values) string {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	baseURL := "https://nogler.ddns.net:443"

	loginReq, err := http.NewRequest(http.MethodPost, baseURL+"/login", strings.NewReader(loginFormData.Encode()))
	assert.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	loginResp, err := client.Do(loginReq)
	assert.NoError(t, err)
	defer loginResp.Body.Close()

	var loginResponse struct {
		Token string `json:"token"`
	}
	err = json.NewDecoder(loginResp.Body).Decode(&loginResponse)
	assert.NoError(t, err)

	return loginResponse.Token
}

// SetupTestUser creates a test user and returns its authentication token
func SetupTestUser(t *testing.T) string {
	timestamp := time.Now().Format("20060102150405")
	email := "test_" + timestamp + "@example.com"
	username := "testuser_" + timestamp
	password := "testpass123"

	formData := url.Values{
		"username": {username},
		"email":    {email},
		"password": {password},
		"icono":    {"1"},
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	baseURL := "https://nogler.ddns.net:443"

	// First try to login to check if the user exists
	loginFormData := url.Values{
		"email":    {email},
		"password": {password},
	}

	loginReq, err := http.NewRequest(http.MethodPost, baseURL+"/login", strings.NewReader(loginFormData.Encode()))
	assert.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	loginResp, err := client.Do(loginReq)
	assert.NoError(t, err)
	
	// If the user exists, delete it first
	if loginResp.StatusCode == http.StatusOK {
		var loginResponse struct {
			Token string `json:"token"`
		}
		err = json.NewDecoder(loginResp.Body).Decode(&loginResponse)
		loginResp.Body.Close()
		assert.NoError(t, err)
		
		if loginResponse.Token != "" {
			CleanupTestData(t, loginResponse.Token)
		}
	}

	// Now create the test user
	req, err := http.NewRequest(http.MethodPost, baseURL+"/signup", strings.NewReader(formData.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	assert.NoError(t, err)
	resp.Body.Close()

	// Login to get the token
	loginReq, err = http.NewRequest(http.MethodPost, baseURL+"/login", strings.NewReader(loginFormData.Encode()))
	assert.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	loginResp, err = client.Do(loginReq)
	assert.NoError(t, err)
	defer loginResp.Body.Close()

	var loginResponse struct {
		Token string `json:"token"`
	}
	err = json.NewDecoder(loginResp.Body).Decode(&loginResponse)
	assert.NoError(t, err)

	return loginResponse.Token
}

// TestLogin verifies user authentication functionality
func TestLogin(t *testing.T) {
	// Setup test user and defer cleanup
	token := SetupTestUser(t)
	defer CleanupTestData(t, token)
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	baseURL := "https://nogler.ddns.net:443"

	t.Run("Login successfully", func(t *testing.T) {
		formData := url.Values{
			"email":    {"test@example.com"},
			"password": {"testpass123"},
		}

		req, err := http.NewRequest(http.MethodPost, baseURL+"/login", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response struct {
			Message string `json:"message"`
			Token   string `json:"token"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "Successfully logged in.", response.Message)
		assert.NotEmpty(t, response.Token)
	})

	t.Run("Login with empty fields", func(t *testing.T) {
		formData := url.Values{
			"email":    {""},
			"password": {""},
		}

		req, err := http.NewRequest(http.MethodPost, baseURL+"/login", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Login with invalid email", func(t *testing.T) {
		formData := url.Values{
			"email":    {"nonexistent@example.com"},
			"password": {"testpass123"},
		}

		req, err := http.NewRequest(http.MethodPost, baseURL+"/login", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Login with invalid password", func(t *testing.T) {
		formData := url.Values{
			"email":    {"test@example.com"},
			"password": {"wrongpassword"},
		}

		req, err := http.NewRequest(http.MethodPost, baseURL+"/login", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestLogout(t *testing.T) {
	token := SetupTestUser(t)
	defer CleanupTestData(t, token)
	
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	baseURL := "https://nogler.ddns.net:443"

	t.Run("Logout successfully", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, baseURL+"/auth/logout", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response struct {
			Message string `json:"message"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "Successfully logged out", response.Message)
	})

	t.Run("Logout without token", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, baseURL+"/auth/logout", nil)
		assert.NoError(t, err)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Logout with invalid token", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, baseURL+"/auth/logout", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer invalid_token")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestGetAllUsers(t *testing.T) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	baseURL := "https://nogler.ddns.net:443"

	// First we need to login to get a token
	loginFormData := url.Values{
		"email":    {"test@example.com"},
		"password": {"testpass123"},
	}

	loginReq, err := http.NewRequest(http.MethodPost, baseURL+"/login", strings.NewReader(loginFormData.Encode()))
	assert.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	loginResp, err := client.Do(loginReq)
	assert.NoError(t, err)
	defer loginResp.Body.Close()

	var loginResponse struct {
		Token string `json:"token"`
	}
	err = json.NewDecoder(loginResp.Body).Decode(&loginResponse)
	assert.NoError(t, err)

	// Now make the request to /allusers
	req, err := http.NewRequest(http.MethodGet, baseURL+"/allusers", nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+loginResponse.Token)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var users []struct {
		Username string `json:"username"`
		Icon     int    `json:"icon"`
	}
	err = json.NewDecoder(resp.Body).Decode(&users)
	assert.NoError(t, err)

	// Print all existing users
	t.Log("Existing users:")
	for _, user := range users {
		t.Logf("Username: %s, Icon: %d", user.Username, user.Icon)
	}
}

func TestGetUserPublicInfo(t *testing.T) {
	token := SetupTestUser(t)
	defer CleanupTestData(t, token)

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	baseURL := "https://nogler.ddns.net:443"

	t.Run("Get user public info successfully", func(t *testing.T) {
		timestamp := time.Now().Format("20060102150405")
		email := "test_public_" + timestamp + "@example.com"
		username := "test_public_info_" + timestamp
		
		formData := url.Values{
			"username": {username},
			"email":    {email},
			"password": {"testpass123"},
			"icono":    {"2"},
		}

		// Crear usuario y obtener token para limpieza
		req, err := http.NewRequest(http.MethodPost, baseURL+"/signup", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		assert.NoError(t, err)
		resp.Body.Close()

		// Obtener token para limpieza
		loginFormData := url.Values{
			"email":    {email},
			"password": {"testpass123"},
		}
		newToken := getTokenForCleanup(t, loginFormData)
		defer CleanupTestData(t, newToken)

		// Now get the public information
		req, err = http.NewRequest(http.MethodGet, baseURL+"/users/"+username, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err = client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response struct {
			Username string `json:"username"`
			Icon     int    `json:"icon"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, username, response.Username)
		assert.Equal(t, 2, response.Icon)
	})

	t.Run("Get non-existent user info", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, baseURL+"/users/nonexistentuser", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Get user info without authorization", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, baseURL+"/users/someuser", nil)
		assert.NoError(t, err)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.NotEqual(t, http.StatusOK, resp.StatusCode)
	})
}

func TestGetUserPrivateInfo(t *testing.T) {
	token := SetupTestUser(t)
	defer CleanupTestData(t, token)

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	baseURL := "https://nogler.ddns.net:443"

	t.Run("Get user private info successfully", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, baseURL+"/auth/me", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response struct {
			Username string `json:"username"`
			Email    string `json:"email"`
			Icon     int    `json:"icon"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response.Username)
		assert.NotEmpty(t, response.Email)
		assert.Equal(t, 1, response.Icon) // El icono por defecto es 1 según SetupTestUser
	})

	t.Run("Get private info without authorization", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, baseURL+"/auth/me", nil)
		assert.NoError(t, err)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.NotEqual(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Get private info with invalid token", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, baseURL+"/auth/me", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer invalid_token")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestUpdateUserInfo(t *testing.T) {
	defer cleanupAllTestUsers(t)
	// Crear usuario de prueba
	timestamp := time.Now().Format("20060102150405")
	testUsername := "test_update_" + timestamp
	testEmail := "test_update_" + timestamp + "@example.com"
	testPassword := "testpass123"

	// Registrar usuario con el parámetro correcto "icono"
	formData := url.Values{
		"username": {testUsername},
		"email":    {testEmail},
		"password": {testPassword},
		"icon":    {"1"},
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	baseURL := "https://nogler.ddns.net:443"

	// Crear usuario
	req, err := http.NewRequest(http.MethodPost, baseURL+"/signup", strings.NewReader(formData.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	assert.NoError(t, err)
	resp.Body.Close()

	// Login para obtener token
	loginFormData := url.Values{
		"email":    {testEmail},
		"password": {testPassword},
	}

	loginReq, err := http.NewRequest(http.MethodPost, baseURL+"/login", strings.NewReader(loginFormData.Encode()))
	assert.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	loginResp, err := client.Do(loginReq)
	assert.NoError(t, err)
	
	var loginResponse struct {
		Token string `json:"token"`
	}
	err = json.NewDecoder(loginResp.Body).Decode(&loginResponse)
	loginResp.Body.Close()
	assert.NoError(t, err)
	token := loginResponse.Token

	defer CleanupTestData(t, token)

	t.Run("Update user info successfully", func(t *testing.T) {
		// Verify initial user
		req, err := http.NewRequest(http.MethodGet, baseURL+"/auth/me", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		t.Logf("Usuario inicial: %s", string(body))
		resp.Body.Close()

		// Only update the necessary fields
		updateData := url.Values{}
		newUsername := "updated_user_" + timestamp
		updateData.Set("username", newUsername)
		updateData.Set("icon", "2")  // Change "icono" to "icon" to match the frontend

		t.Logf("Datos a actualizar: %v", updateData)

		req, err = http.NewRequest(http.MethodPatch, baseURL+"/auth/update", strings.NewReader(updateData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err = client.Do(req)
		assert.NoError(t, err)
		body, err = io.ReadAll(resp.Body)
		t.Logf("Respuesta de actualización: %s", string(body))

		var response struct {
			Message string `json:"message"`
			User    struct {
				Username string `json:"username"`
				Email    string `json:"email"`
				Icon     int    `json:"icon"`
			} `json:"user"`
			Token string `json:"token"`
		}
		err = json.Unmarshal(body, &response)
		assert.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "User updated successfully", response.Message)
		assert.Equal(t, newUsername, response.User.Username)
		assert.Equal(t, 2, response.User.Icon)
	})

	t.Run("Update with existing username", func(t *testing.T) {
		// First create a user with a known username
		timestamp := time.Now().Format("20060102150405")
		existingEmail := "existing_" + timestamp + "@example.com"
		existingUsername := "existing_user_" + timestamp
		
		formData := url.Values{
			"username": {existingUsername},
			"email":    {existingEmail},
			"password": {"testpass123"},
			"icon":    {"1"},
		}

		// Create the user and get token for cleanup
		req, err := http.NewRequest(http.MethodPost, baseURL+"/signup", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		assert.NoError(t, err)
		resp.Body.Close()

		// Get token for cleanup
		loginFormData := url.Values{
			"email":    {existingEmail},
			"password": {"testpass123"},
		}
		existingToken := getTokenForCleanup(t, loginFormData)
		defer CleanupTestData(t, existingToken)

		// Try to update our test user with the existing username
		updateData := url.Values{
			"username": {existingUsername},
		}

		req, err = http.NewRequest(http.MethodPatch, baseURL+"/auth/update", strings.NewReader(updateData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err = client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("Update with invalid icon", func(t *testing.T) {
		formData := url.Values{
			"icon": {"icon_invalido"},
		}

		req, err := http.NewRequest(http.MethodPatch, baseURL+"/auth/update", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Update without authorization", func(t *testing.T) {
		formData := url.Values{
			"username": {"unauthorized_update"},
		}

		req, err := http.NewRequest(http.MethodPatch, baseURL+"/auth/update", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Update with invalid token", func(t *testing.T) {
		formData := url.Values{
			"username": {"invalid_token_update"},
		}

		req, err := http.NewRequest(http.MethodPatch, baseURL+"/auth/update", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer invalid_token")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func cleanupAllTestUsers(t *testing.T) {
	// Obtener lista de usuarios
	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Get("https://nogler.ddns.net:443/users")
	if err != nil {
		t.Logf("Error getting users: %v", err)
		return
	}
	defer resp.Body.Close()

	var users []struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	json.NewDecoder(resp.Body).Decode(&users)

	// Limpiar usuarios de prueba
	for _, user := range users {
		if strings.HasPrefix(user.Username, "test_") ||
		   strings.HasPrefix(user.Username, "testuser") ||  // Eliminar el "_" para capturar también "testuser1", "testuser2", etc
		   strings.HasPrefix(user.Username, "existing_user_") ||
		   strings.HasPrefix(user.Username, "updated_user_") ||
		   strings.HasPrefix(user.Username, "test_signup_user") ||
		   strings.HasPrefix(user.Username, "test_public_info_") ||
		   strings.HasPrefix(user.Username, "test_existing_user_") ||
		   strings.HasPrefix(user.Username, "test_update_") {
			// Intentar login y eliminar
			loginFormData := url.Values{
				"email":    {user.Email},
				"password": {"testpass123"},
			}
			token := getTokenForCleanup(t, loginFormData)
			if token != "" {
				CleanupTestData(t, token)
			}
		}
	}
}
