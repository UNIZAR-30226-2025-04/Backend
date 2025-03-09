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
		t.Logf("Respuesta completa del servidor: %s", string(body))
		deleteResp.Body.Close()
	}
}

func TestSignUp(t *testing.T) {
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
		// Generamos el timestamp una sola vez
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
		t.Logf("Respuesta completa del servidor: %s", string(body))
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
		// Primero creamos un usuario
		timestamp := time.Now().Format("20060102150405")
		formData := url.Values{
			"username": {"test_existing_user_" + timestamp},
			"email":    {"test_existing_" + timestamp + "@example.com"},
			"password": {"testpass123"},
			"icono":    {"1"},
		}

		// Creamos el usuario por primera vez
		req, err := http.NewRequest(http.MethodPost, baseURL+"/signup", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		assert.NoError(t, err)
		resp.Body.Close()

		// Intentamos crear el mismo usuario de nuevo
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

		// Debería ser exitoso ya que el código maneja iconos inválidos asignando el valor por defecto 0
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})
}

func SetupTestUser(t *testing.T) string {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	baseURL := "https://nogler.ddns.net:443"

	// Primero intentamos hacer login para ver si el usuario existe
	loginFormData := url.Values{
		"email":    {"test@example.com"},
		"password": {"testpass123"},
	}

	loginReq, err := http.NewRequest(http.MethodPost, baseURL+"/login", strings.NewReader(loginFormData.Encode()))
	assert.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	loginResp, err := client.Do(loginReq)
	assert.NoError(t, err)
	
	// Si el usuario existe, lo eliminamos primero
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

	// Ahora creamos el usuario de prueba
	formData := url.Values{
		"username": {"testuser_" + time.Now().Format("20060102150405")},
		"email":    {"test_" + time.Now().Format("20060102150405") + "@example.com"},
		"password": {"testpass123"},
		"icono":    {"1"},
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/signup", strings.NewReader(formData.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	assert.NoError(t, err)
	resp.Body.Close()

	// Hacemos login para obtener el token
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

func TestLogin(t *testing.T) {
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

	// Primero necesitamos hacer login para obtener un token
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

	// Ahora hacemos la petición a /allusers
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

	// Imprimir todos los usuarios
	t.Log("Usuarios existentes:")
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
		// Primero creamos un usuario de prueba
		timestamp := time.Now().Format("20060102150405")
		username := "test_public_info_" + timestamp
		
		formData := url.Values{
			"username": {username},
			"email":    {"test_public_" + timestamp + "@example.com"},
			"password": {"testpass123"},
			"icono":    {"2"},
		}

		req, err := http.NewRequest(http.MethodPost, baseURL+"/signup", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		assert.NoError(t, err)
		resp.Body.Close()

		// Ahora obtenemos la información pública
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
		"icono":    {"1"},
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
		// Verificar usuario inicial
		req, err := http.NewRequest(http.MethodGet, baseURL+"/auth/me", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		t.Logf("Usuario inicial: %s", string(body))
		resp.Body.Close()

		// Solo actualizar los campos necesarios
		updateData := url.Values{}
		newUsername := "updated_user_" + timestamp
		updateData.Set("username", newUsername)
		updateData.Set("icon", "2")  // Cambiar de "icono" a "icon" para coincidir con el frontend

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
		existingUsername := "existing_user_" + time.Now().Format("20060102150405")
		formData := url.Values{
			"username": {existingUsername},
			"email":    {"existing_" + time.Now().Format("20060102150405") + "@example.com"},
			"password": {"testpass123"},
			"icono":    {"1"},
		}

		req, err := http.NewRequest(http.MethodPost, baseURL+"/signup", strings.NewReader(formData.Encode()))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		assert.NoError(t, err)
		resp.Body.Close()

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
			"icono": {"icono_invalido"},
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
