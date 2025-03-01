package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

// Global configuration for the test
type TestConfig struct {
	DB       *sql.DB
	DBUser   string
	DBPass   string
	DBName   string
	Columns  []string
	MainPath string
	Cmd      *exec.Cmd
}

func TestMainAPIIntegration(t *testing.T) {
	fmt.Println("\n=== MAIN API INTEGRATION TEST ===")
	
	// Initialize configuration
	config := &TestConfig{
		DBUser: "nogler",
		DBPass: "nogler",
		DBName: "nogler",
	}
	
	// Connect to database
	connectToDatabase(t, config)
	defer config.DB.Close()
	
	// Verify table structure
	verifyTableStructure(t, config)
	
	// Prepare test data
	prepareTestData(t, config)
	
	// Start the main server
	startMainServer(t, config)
	defer stopServer(config)
	
	// Perform HTTP request
	response := performHTTPRequest(t, config)
	
	// Verify response
	verifyResponse(t, response)
	
	// Clean test data
	cleanTestData(t, config)
	
	fmt.Println("\n=== RESULT ===")
	fmt.Println("Main API integration test completed successfully")
}

// Connects to PostgreSQL database
func connectToDatabase(t *testing.T, config *TestConfig) {
	fmt.Println("\n=== CONFIGURING POSTGRESQL CONNECTION ===")
	dbConnStr := fmt.Sprintf("postgresql://%s:%s@localhost:5432/%s?sslmode=disable", 
		config.DBUser, config.DBPass, config.DBName)
	
	fmt.Printf("Connecting to database with user: %s\n", config.DBUser)
	db, err := sql.Open("postgres", dbConnStr)
	if err != nil {
		t.Fatalf("Error connecting to PostgreSQL: %v", err)
	}
	config.DB = db
	
	// Verify PostgreSQL connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Error pinging PostgreSQL: %v", err)
	}
	fmt.Println("PostgreSQL connection established successfully")
}

// Verifies the structure of the game_profiles table
func verifyTableStructure(t *testing.T, config *TestConfig) {
	fmt.Println("\n=== VERIFYING TABLE STRUCTURE ===")
	rows, err := config.DB.Query("SELECT column_name FROM information_schema.columns WHERE table_name = 'game_profiles'")
	if err != nil {
		fmt.Printf("Error verifying columns: %v\n", err)
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var column string
		rows.Scan(&column)
		config.Columns = append(config.Columns, column)
	}
	fmt.Printf("Columns in game_profiles: %v\n", config.Columns)
}

// Cleans and prepares test data in the database
func prepareTestData(t *testing.T, config *TestConfig) {
	fmt.Println("\n=== PREPARING TEST DATA ===")
	
	// Clean existing data
	cleanTestData(t, config)
	
	// Create test user if it doesn't exist
	createTestUser(t, config)
	
	// Update user icon
	updateUserIcon(t, config)
	
	// Create test lobby
	createTestLobby(t, config)
	
	fmt.Println("Test data prepared successfully")
}

// Creates the test user if it doesn't exist
func createTestUser(t *testing.T, config *TestConfig) {
	// Check if user exists
	var userExists bool
	err := config.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = 'testuser')").Scan(&userExists)
	if err != nil {
		fmt.Printf("Error checking if user exists: %v\n", err)
		return
	}

	if !userExists {
		_, err = config.DB.Exec(`
			INSERT INTO users (username, email, password_hash, full_name) 
			VALUES ('testuser', 'test@example.com', 'password123', 'Test User')
		`)
		if err != nil {
			fmt.Printf("Error inserting test user: %v\n", err)
		} else {
			fmt.Println("User created successfully")
		}
	} else {
		fmt.Println("User already exists")
	}
	
	// Check if game profile exists
	var profileExists bool
	err = config.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM game_profiles WHERE username = 'testuser')").Scan(&profileExists)
	if err != nil {
		fmt.Printf("Error checking if profile exists: %v\n", err)
		return
	}

	if !profileExists {
		_, err = config.DB.Exec(`
			INSERT INTO game_profiles (username) 
			VALUES ('testuser')
		`)
		if err != nil {
			fmt.Printf("Error inserting game profile: %v\n", err)
		} else {
			fmt.Println("Game profile created successfully")
		}
	}
}

// Updates the user icon
func updateUserIcon(t *testing.T, config *TestConfig) {
	// Check if user_icon column exists
	var hasUserIcon bool
	for _, col := range config.Columns {
		if col == "user_icon" {
			hasUserIcon = true
			break
		}
	}

	// Update user icon
	if hasUserIcon {
		_, err := config.DB.Exec(`
			UPDATE game_profiles SET user_icon = '5' WHERE username = 'testuser'
		`)
		if err != nil {
			fmt.Printf("Error updating user icon: %v\n", err)
		} else {
			fmt.Println("User icon updated successfully")
		}
	} else {
		// Add column if it doesn't exist
		_, err := config.DB.Exec(`ALTER TABLE game_profiles ADD COLUMN IF NOT EXISTS user_icon VARCHAR(10) DEFAULT '5'`)
		if err != nil {
			fmt.Printf("Error adding user_icon column: %v\n", err)
		} else {
			fmt.Println("User_icon column added successfully")
			// Update value
			_, err = config.DB.Exec(`UPDATE game_profiles SET user_icon = '5' WHERE username = 'testuser'`)
			if err != nil {
				fmt.Printf("Error updating user icon: %v\n", err)
			} else {
				fmt.Println("User icon updated successfully")
			}
		}
	}
}

// Creates the test lobby
func createTestLobby(t *testing.T, config *TestConfig) {
	// Insert test lobby
	_, err := config.DB.Exec(`
		INSERT INTO game_lobbies (id, creator_username) 
		VALUES ('test123', 'testuser')
	`)
	if err != nil {
		fmt.Printf("Error inserting test lobby: %v\n", err)
	} else {
		fmt.Println("Lobby created successfully")
	}

	// Insert test player
	_, err = config.DB.Exec(`
		INSERT INTO in_game_players (lobby_id, username) 
		VALUES ('test123', 'testuser')
	`)
	if err != nil {
		fmt.Printf("Error inserting test player: %v\n", err)
	} else {
		fmt.Println("Player added to lobby successfully")
	}
}

// Starts the main server (main.go)
func startMainServer(t *testing.T, config *TestConfig) {
	fmt.Println("\n=== STARTING MAIN SERVER (main.go) ===")
	
	// Find main.go file
	findMainFile(t, config)
	
	// Configure and run server
	runServer(t, config)
	
	// Wait for server to be ready
	waitForServerReady(t, config)
}

// Finds the main.go file
func findMainFile(t *testing.T, config *TestConfig) {
	mainLocations := []string{
		"../main.go",
		"../../main.go",
		"main.go",
		"./main.go",
		"api/main.go",
	}
	
	for _, path := range mainLocations {
		if _, err := os.Stat(path); err == nil {
			config.MainPath = path
			break
		}
	}
	
	if config.MainPath == "" {
		t.Fatalf("Could not find main.go file")
	}
	
	fmt.Printf("Main.go file found at: %s\n", config.MainPath)
}

// Runs the main server
func runServer(t *testing.T, config *TestConfig) {
	// Configure environment variables for the server
	cmd := exec.Command("go", "run", config.MainPath)
	cmd.Env = append(os.Environ(),
		"DB_USER="+config.DBUser,
		"DB_PASSWORD="+config.DBPass,
		"DB_NAME="+config.DBName,
		"DB_HOST=localhost",
		"DB_PORT=5432",
		"REDIS_ADDR=localhost:6379",
		"PORT=8082",
		"GIN_MODE=release",
	)
	config.Cmd = cmd
	
	// Capture server output
	serverOutput, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Error capturing server output: %v", err)
	}
	
	serverError, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Error capturing server errors: %v", err)
	}
	
	// Start server in background
	if err := cmd.Start(); err != nil {
		t.Fatalf("Error starting server: %v", err)
	}
	
	// Read server output in background
	go readServerOutput(serverOutput, "Server")
	go readServerOutput(serverError, "Server error")
}

// Reads server output
func readServerOutput(pipe interface{}, prefix string) {
	reader, ok := pipe.(interface{ Read([]byte) (int, error) })
	if !ok {
		return
	}
	
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			break
		}
		fmt.Printf("%s: %s", prefix, buf[:n])
	}
}

// Waits for the server to be ready
func waitForServerReady(t *testing.T, config *TestConfig) {
	fmt.Println("Waiting for main server to be ready...")
	time.Sleep(5 * time.Second)
	
	// Verify server is listening
	ready := false
	apiURL := "http://localhost:8082/api/v1/lobby/test123"
	
	for i := 0; i < 10; i++ {
		resp, err := http.Get(apiURL)
		if err == nil {
			resp.Body.Close()
			ready = true
			break
		}
		fmt.Printf("Attempt %d: Error connecting: %v\n", i+1, err)
		time.Sleep(1 * time.Second)
	}
	
	if !ready {
		t.Fatalf("Server is not responding after several attempts")
	}
	
	fmt.Println("Main server started successfully")
}

// Performs an HTTP request to the server
func performHTTPRequest(t *testing.T, config *TestConfig) map[string]interface{} {
	fmt.Println("\n=== PERFORMING HTTP REQUEST TO MAIN SERVER ===")
	apiURL := "http://localhost:8082/api/v1/lobby/test123"
	fmt.Printf("Request URL: %s\n", apiURL)
	
	// Create HTTP client
	client := &http.Client{}
	
	// Create request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		t.Fatalf("Error creating HTTP request: %v", err)
	}
	
	// Execute request
	fmt.Println("Sending request to main server...")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error performing HTTP request: %v", err)
	}
	defer resp.Body.Close()
	
	// Verify response
	fmt.Println("\n=== HTTP RESPONSE FROM MAIN SERVER ===")
	fmt.Printf("Status code: %d\n", resp.StatusCode)
	
	// Verify status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Decode JSON response
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Error deserializing JSON response: %v", err)
	}
	
	fmt.Printf("Response body: %v\n", response)
	return response
}

// Verifies the response fields
func verifyResponse(t *testing.T, response map[string]interface{}) {
	fmt.Println("\n=== FIELD VERIFICATION ===")
	fmt.Printf("Expected - code: %s, Received: %v\n", "test123", response["code"])
	fmt.Printf("Expected - host_name: %s, Received: %v\n", "testuser", response["host_name"])
	fmt.Printf("Expected - host_icon: %s, Received: %v\n", "5", response["host_icon"])
	
	assert.Equal(t, "test123", response["code"])
	assert.Equal(t, "testuser", response["host_name"])
	assert.Equal(t, "5", response["host_icon"])
	assert.Equal(t, float64(1), response["player_count"])
}

// Cleans the test data
func cleanTestData(t *testing.T, config *TestConfig) {
	fmt.Println("\n=== CLEANING TEST DATA ===")
	_, err := config.DB.Exec("DELETE FROM in_game_players WHERE lobby_id = 'test123'")
	if err != nil {
		fmt.Printf("Warning when cleaning players: %v\n", err)
	}

	_, err = config.DB.Exec("DELETE FROM game_lobbies WHERE id = 'test123'")
	if err != nil {
		fmt.Printf("Warning when cleaning lobby: %v\n", err)
	}
	fmt.Println("Test data cleaned successfully")
}

// Stops the main server
func stopServer(config *TestConfig) {
	fmt.Println("Stopping main server...")
	if config.Cmd != nil && config.Cmd.Process != nil {
		if err := config.Cmd.Process.Kill(); err != nil {
			fmt.Printf("Error stopping server: %v\n", err)
		}
	}
} 