package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	authsqlite "keeper/server/adapters/auth/sqlite"
	messagingsqlite "keeper/server/adapters/messaging/sqlite"
	"keeper/server/core/ports"
	"keeper/server/core/services"
	"keeper/server/models"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3" // SQLite driver for database/sql
)

const (
	testJWTSecret = "test-super-secret-key-for-integration-tests"
	dbTimeout     = 5 * time.Second
)

var (
	testServer *httptest.Server
	testUserRepo ports.UserRepository
	testMessageRepo ports.MessageRepository
	testAuthSvc ports.AuthService
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Use in-memory SQLite database for tests
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		log.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	testUserRepo = authsqlite.NewSQLiteUserRepository(db)
	if err := testUserRepo.InitUserSchema(); err != nil {
		log.Fatalf("Failed to initialize user schema: %v", err)
	}

	testMessageRepo = messagingsqlite.NewSQLiteRepository(db)
	if err := testMessageRepo.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize message schema: %v", err)
	}

	// Initialize auth service
	testAuthSvc = services.NewAuthService(testUserRepo, testJWTSecret)

	// Setup router and test server
	// Note: This mirrors the setup in main.go but uses test instances
	mux := http.NewServeMux()
	mux.Handle("/api/register", corsMiddleware(registerHandler(testAuthSvc)))
	mux.Handle("/api/login", corsMiddleware(loginHandler(testAuthSvc)))
	mux.Handle("/ws", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsHandler(w, r, testMessageRepo, testAuthSvc) // wsHandler is from main.go
	})))

	testServer = httptest.NewServer(mux)
	defer testServer.Close()

	log.Printf("Test server started on %s", testServer.URL)

	// Run tests
	exitCode := m.Run()
	os.Exit(exitCode)
}

// --- Helper Functions ---

type AuthHelperResponse struct {
	Token string `json:"token"`
}

type RegisterHelperResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}


func registerUser(serverURL, username, password string) (RegisterHelperResponse, error) {
	var R RegisterHelperResponse
	reqBody, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := http.Post(fmt.Sprintf("%s/api/register", serverURL), "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return R, fmt.Errorf("failed to register user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp ErrorResponse // Assuming ErrorResponse is defined in main.go (it is)
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return R, fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, errResp.Error)
	}
	if err := json.NewDecoder(resp.Body).Decode(&R); err != nil {
		return R, fmt.Errorf("failed to decode registration response: %w", err)
	}
	return R, nil
}

func loginUser(serverURL, username, password string) (string, error) {
	reqBody, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := http.Post(fmt.Sprintf("%s/api/login", serverURL), "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to login user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, errResp.Error)
	}

	var authResp AuthHelperResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("failed to decode login response: %w", err)
	}
	return authResp.Token, nil
}

func connectWebSocket(wsURLBase, token string) (*websocket.Conn, *http.Response, error) {
	wsURL := strings.Replace(wsURLBase, "http", "ws", 1) + "/ws?token=" + token
	if token == "" { // For testing no token case
		wsURL = strings.Replace(wsURLBase, "http", "ws", 1) + "/ws"
	}
	
	header := http.Header{} // Can add origin if needed by CheckOrigin
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	return conn, resp, err
}

// Helper to read a ServerMessage from WebSocket, with timeout
func readServerMessage(conn *websocket.Conn, timeout time.Duration) (*ServerMessage, error) {
	var msg ServerMessage // ServerMessage is from main.go
	conn.SetReadDeadline(time.Now().Add(timeout))
	err := conn.ReadJSON(&msg)
	conn.SetReadDeadline(time.Time{}) // Clear deadline
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// Helper to send a "sendMessage" client message
func sendClientMessage(conn *websocket.Conn, text string) error {
	payload := SendMessagePayload{Text: text} // SendMessagePayload from main.go
	payloadBytes, _ := json.Marshal(payload)
	clientMsg := ClientMessage{ // ClientMessage from main.go
		Type:    "sendMessage",
		Payload: json.RawMessage(payloadBytes),
	}
	return conn.WriteJSON(clientMsg)
}


// --- Test Functions ---

func TestWebSocketConnection_Authentication(t *testing.T) {
	t.Run("NoToken", func(t *testing.T) {
		_, resp, err := connectWebSocket(testServer.URL, "")
		if err == nil {
			t.Fatal("Expected WebSocket connection to fail without token, but it succeeded.")
		}
		// gorilla/websocket returns an error if the handshake fails.
		// The HTTP response might be available in resp.
		if resp == nil {
			t.Fatalf("Expected non-nil HTTP response on handshake failure, got nil. Error: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusBadRequest { // server might return 400 or 401
			t.Errorf("Expected status 401 or 400 for no token, got %d", resp.StatusCode)
		}
	})

	t.Run("InvalidToken", func(t *testing.T) {
		_, resp, err := connectWebSocket(testServer.URL, "invalid-jwt-token")
		if err == nil {
			t.Fatal("Expected WebSocket connection to fail with invalid token, but it succeeded.")
		}
		if resp == nil {
			t.Fatalf("Expected non-nil HTTP response on handshake failure, got nil. Error: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 401 or 400 for invalid token, got %d", resp.StatusCode)
		}
	})

	t.Run("ValidToken", func(t *testing.T) {
		_, err := registerUser(testServer.URL, "wsuser1", "password123")
		if err != nil {
			t.Fatalf("Failed to register user: %v", err)
		}
		token, err := loginUser(testServer.URL, "wsuser1", "password123")
		if err != nil {
			t.Fatalf("Failed to login user: %v", err)
		}

		conn, resp, err := connectWebSocket(testServer.URL, token)
		if err != nil {
			t.Fatalf("WebSocket connection failed with valid token: %v (HTTP Response: %+v)", err, resp)
		}
		defer conn.Close()
		t.Log("WebSocket connection successful with valid token.")
		// Optionally, try to read a message (e.g., history, if sent immediately) or send one
	})
}


func TestWebSocket_SendMessage_Broadcast(t *testing.T) {
	// Register and login UserA
	_, err := registerUser(testServer.URL, "broadcasterA", "passA")
	if err != nil {
		t.Fatalf("UserA registration failed: %v", err)
	}
	tokenA, err := loginUser(testServer.URL, "broadcasterA", "passA")
	if err != nil {
		t.Fatalf("UserA login failed: %v", err)
	}

	// Register and login UserB
	_, err = registerUser(testServer.URL, "receiverB", "passB")
	if err != nil {
		t.Fatalf("UserB registration failed: %v", err)
	}
	tokenB, err := loginUser(testServer.URL, "receiverB", "passB")
	if err != nil {
		t.Fatalf("UserB login failed: %v", err)
	}

	// Connect UserA
	connA, _, err := connectWebSocket(testServer.URL, tokenA)
	if err != nil {
		t.Fatalf("UserA WebSocket connection failed: %v", err)
	}
	defer connA.Close()

	// Connect UserB
	connB, _, err := connectWebSocket(testServer.URL, tokenB)
	if err != nil {
		t.Fatalf("UserB WebSocket connection failed: %v", err)
	}
	defer connB.Close()
	
	// UserA and UserB might receive history messages first. We need to drain them or account for them.
	// For simplicity in this specific test, we'll try to read them and ignore if they are history.
	// A more robust way would be to have a helper that waits for a specific message type.
	
	// Drain potential history for UserA
	if msg, err := readServerMessage(connA, 1*time.Second); err == nil && msg.Type == "history" {
		t.Log("UserA received history message.")
	}
	// Drain potential history for UserB
	if msg, err := readServerMessage(connB, 1*time.Second); err == nil && msg.Type == "history" {
		t.Log("UserB received history message.")
	}


	messageText := "Hello from broadcasterA!"
	var wg sync.WaitGroup
	wg.Add(2) // Expect two messages (one for A, one for B)

	var msgA, msgB *ServerMessage
	var errA, errB error

	// Listener for UserA
	go func() {
		defer wg.Done()
		msgA, errA = readServerMessage(connA, 5*time.Second)
	}()

	// Listener for UserB
	go func() {
		defer wg.Done()
		msgB, errB = readServerMessage(connB, 5*time.Second)
	}()

	// UserA sends a message
	if err := sendClientMessage(connA, messageText); err != nil {
		t.Fatalf("UserA failed to send message: %v", err)
	}

	wg.Wait() // Wait for both listeners to complete or timeout

	// Check UserA's received message
	if errA != nil {
		t.Fatalf("Error reading message for UserA: %v", errA)
	}
	if msgA == nil {
		t.Fatal("UserA did not receive any message.")
	}
	if msgA.Type != "newMessage" {
		t.Errorf("UserA: Expected message type 'newMessage', got '%s'", msgA.Type)
	}
	
	var payloadA NewMessagePayload
	payloadBytesA, _ := json.Marshal(msgA.Payload) // Marshal it back to bytes
	if err := json.Unmarshal(payloadBytesA, &payloadA); err != nil { // Then unmarshal into concrete type
		t.Fatalf("UserA: Failed to unmarshal newMessage payload: %v", err)
	}

	if payloadA.User != "broadcasterA" {
		t.Errorf("UserA: Expected message user 'broadcasterA', got '%s'", payloadA.User)
	}
	if payloadA.Text != messageText {
		t.Errorf("UserA: Expected message text '%s', got '%s'", messageText, payloadA.Text)
	}

	// Check UserB's received message
	if errB != nil {
		t.Fatalf("Error reading message for UserB: %v", errB)
	}
	if msgB == nil {
		t.Fatal("UserB did not receive any message.")
	}
	if msgB.Type != "newMessage" {
		t.Errorf("UserB: Expected message type 'newMessage', got '%s'", msgB.Type)
	}

	var payloadB NewMessagePayload
	payloadBytesB, _ := json.Marshal(msgB.Payload)
	if err := json.Unmarshal(payloadBytesB, &payloadB); err != nil {
		t.Fatalf("UserB: Failed to unmarshal newMessage payload: %v", err)
	}

	if payloadB.User != "broadcasterA" { // Message came from UserA
		t.Errorf("UserB: Expected message user 'broadcasterA', got '%s'", payloadB.User)
	}
	if payloadB.Text != messageText {
		t.Errorf("UserB: Expected message text '%s', got '%s'", messageText, payloadB.Text)
	}
}

func TestWebSocket_MessageHistory(t *testing.T) {
	// Register and login User HistTest
	_, err := registerUser(testServer.URL, "histUser", "passHist")
	if err != nil {
		t.Fatalf("histUser registration failed: %v", err)
	}
	tokenHist, err := loginUser(testServer.URL, "histUser", "passHist")
	if err != nil {
		t.Fatalf("histUser login failed: %v", err)
	}

	// Pre-populate a message using the repository directly for controlled state
	// This assumes direct access to testMessageRepo is fine for setting up test state.
	// Alternatively, connect another client, send a message, disconnect.
	preSavedMsg, err := testMessageRepo.SaveMessage(models.Message{
		User: "system", Text: "A pre-existing message for history.", Timestamp: time.Now().Add(-1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Failed to pre-save message: %v", err)
	}
	t.Logf("Pre-saved message ID: %d, Text: %s", preSavedMsg.ID, preSavedMsg.Text)


	// Connect User HistTest
	connHist, _, err := connectWebSocket(testServer.URL, tokenHist)
	if err != nil {
		t.Fatalf("histUser WebSocket connection failed: %v", err)
	}
	defer connHist.Close()

	// Expect history message first
	historyMsg, err := readServerMessage(connHist, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to read message from histUser: %v", err)
	}
	if historyMsg.Type != "history" {
		t.Fatalf("Expected first message to be 'history', got '%s'", historyMsg.Type)
	}

	var historyPayload HistoryPayload // HistoryPayload is from main.go
	payloadBytes, _ := json.Marshal(historyMsg.Payload)
	if err := json.Unmarshal(payloadBytes, &historyPayload); err != nil {
		t.Fatalf("Failed to unmarshal history payload: %v", err)
	}

	if len(historyPayload.Messages) == 0 {
		t.Error("Expected history messages, but got none.")
	} else {
		t.Logf("Received %d messages in history.", len(historyPayload.Messages))
	}

	foundPreSaved := false
	for _, msg := range historyPayload.Messages {
		if msg.Text == preSavedMsg.Text && msg.User == preSavedMsg.User {
			foundPreSaved = true
			if msg.ID != preSavedMsg.ID { // Check ID if your SaveMessage returns it and it's consistent
				t.Logf("Note: History message ID %d for pre-saved message, DB ID was %d (timestamps might differ slightly due to DB precision)", msg.ID, preSavedMsg.ID)
			}
			break
		}
	}
	if !foundPreSaved {
		t.Errorf("Pre-saved message with text '%s' not found in history.", preSavedMsg.Text)
	}


	// Optional: User HistTest sends a new message to ensure it's also added to DB for next client
	msgToSend := "A new message from histUser"
	if err := sendClientMessage(connHist, msgToSend); err != nil {
		t.Fatalf("histUser failed to send message: %v", err)
	}
	// Wait for this message to be broadcasted and saved
	if _, err := readServerMessage(connHist, 2*time.Second); err != nil { // Read own message back
		t.Logf("histUser did not read back its own message: %v (may be fine if test ends)", err)
	} else {
		t.Logf("histUser read back its own message.")
	}


	// Connect another user (histUser2) to check if histUser's new message is in subsequent history
	_, err = registerUser(testServer.URL, "histUser2", "passHist2")
	if err != nil {
		t.Fatalf("histUser2 registration failed: %v", err)
	}
	tokenHist2, err := loginUser(testServer.URL, "histUser2", "passHist2")
	if err != nil {
		t.Fatalf("histUser2 login failed: %v", err)
	}

	connHist2, _, err := connectWebSocket(testServer.URL, tokenHist2)
	if err != nil {
		t.Fatalf("histUser2 WebSocket connection failed: %v", err)
	}
	defer connHist2.Close()

	historyMsg2, err := readServerMessage(connHist2, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to read message from histUser2: %v", err)
	}
	if historyMsg2.Type != "history" {
		t.Fatalf("histUser2: Expected first message to be 'history', got '%s'", historyMsg2.Type)
	}
	
	var historyPayload2 HistoryPayload
	payloadBytes2, _ := json.Marshal(historyMsg2.Payload)
	if err := json.Unmarshal(payloadBytes2, &historyPayload2); err != nil {
		t.Fatalf("histUser2: Failed to unmarshal history payload: %v", err)
	}

	foundNewMsgFromHistUser := false
	for _, msg := range historyPayload2.Messages {
		if msg.Text == msgToSend && msg.User == "histUser" {
			foundNewMsgFromHistUser = true
			break
		}
	}
	if !foundNewMsgFromHistUser {
		t.Errorf("Message '%s' from histUser not found in histUser2's history.", msgToSend)
	}

}

// Note: The ServerMessage, ClientMessage, SendMessagePayload, NewMessagePayload, HistoryPayload, ErrorResponse
// structs are assumed to be defined in main.go and accessible here because this test file is in package main.
// If this file were in a different package (e.g., main_test), these types would need to be imported or redefined.
// The corsMiddleware, registerHandler, loginHandler, wsHandler are also used from main.go.
// This tight coupling is common for integration tests in the same package.
