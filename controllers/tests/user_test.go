package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSignUp(t *testing.T) {
    client := &http.Client{
        Timeout: time.Second * 10,
    }
    baseURL := "https://nogler.ddns.net:443"

    t.Run("Sign up successfully", func(t *testing.T) {
        formData := url.Values{
            "username": {"testuser"},
            "email":    {"test@example.com"},
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
        assert.Equal(t, "testuser", response.User.Username)
        assert.Equal(t, "test@example.com", response.User.Email)
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
        formData := url.Values{
            "username": {"testuser"},
            "email":    {"test@example.com"},
            "password": {"testpass123"},
            "icono":    {"1"},
        }

        req, err := http.NewRequest(http.MethodPost, baseURL+"/signup", strings.NewReader(formData.Encode()))
        assert.NoError(t, err)
        req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

        resp, err := client.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusConflict, resp.StatusCode)
    })

    t.Run("Sign up with invalid icon", func(t *testing.T) {
        formData := url.Values{
            "username": {"newuser"},
            "email":    {"new@example.com"},
            "password": {"newpass123"},
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

func SetupTestUser(t *testing.T) {
    // Primero registramos un usuario para las pruebas
    client := &http.Client{
        Timeout: time.Second * 10,
    }
    baseURL := "https://nogler.ddns.net:443"

    formData := url.Values{
        "username": {"testuser"},
        "email":    {"test@example.com"},
        "password": {"testpass123"},
        "icono":    {"1"},
    }

    req, err := http.NewRequest(http.MethodPost, baseURL+"/signup", strings.NewReader(formData.Encode()))
    assert.NoError(t, err)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := client.Do(req)
    assert.NoError(t, err)
    defer resp.Body.Close()
}

func TestLogin(t *testing.T) {
    SetupTestUser(t)
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

