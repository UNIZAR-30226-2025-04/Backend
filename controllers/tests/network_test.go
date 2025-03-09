package controllers_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestPing verifies the server's ping endpoint
func TestPing(t *testing.T) {
    client := &http.Client{
        Timeout: time.Second * 10,
    }
    baseURL := "https://nogler.ddns.net:443"

	// Test ping server successfully
    t.Run("Ping server successfully", func(t *testing.T) {
        req, err := http.NewRequest(http.MethodGet, baseURL+"/ping", nil)
        assert.NoError(t, err)
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
        assert.Equal(t, "pong, hola", response.Message)
    })
}
