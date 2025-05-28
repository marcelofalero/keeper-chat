package main

import (
	// "database/sql" // No longer directly used in main
	"encoding/json"
	"encoding/base64" // Added for Basic Auth
	"log"
	"net/http"
	"os"
	// "path/filepath" // No longer directly used in main
	"strings"       // Added for Basic Auth
	"time"

	"errors"
	authmemory "keeper/server/adapters/auth/memory"         // Added for in-memory user repo
	authsqlite "keeper/server/adapters/auth/sqlite"         // For userRepo
	messagingmemory "keeper/server/adapters/messaging/memory" // Added for in-memory message repo
	messagingsqlite "keeper/server/adapters/messaging/sqlite" // For messageRepo
	"keeper/server/core/ports"
	"keeper/server/core/services"
	"keeper/server/models"
	"keeper/server/seeder" // Added seeder package
	"sync"
	// "strings" // No longer needed directly in wsHandler for Bearer token

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

// --- WebSocket API Contract Structs ---

// ClientMessage is a generic wrapper for messages from the client.
type ClientMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// ServerMessage is a generic wrapper for messages to the client.
type ServerMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// SendMessagePayload is the expected payload for "sendMessage" type client messages.
type SendMessagePayload struct {
	Text string `json:"text"`
}

// NewMessagePayload is the payload for "newMessage" type server messages.
type NewMessagePayload struct {
	models.Message // Embed models.Message directly
}

// ErrorPayload is the payload for "error" type server messages.
type ErrorPayload struct {
	Message string `json:"message"`
}

// HistoryPayload is the payload for "history" type server messages.
type HistoryPayload struct {
	Messages []models.Message `json:"messages"`
}

// --- Connection Management ---
var (
	connections      = make(map[*websocket.Conn]string) // Map connection to username
	connectionsMutex = &sync.Mutex{}
)

// --- WebSocket Upgrader ---
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all connections by default for development.
		// TODO: Implement a proper origin check for production.
		return true
	},
}

// --- Request/Response Structs ---

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UserResponse is used for HTTP register endpoint
type UserResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// AuthResponse is used for HTTP login endpoint
type AuthResponse struct {
	Token string `json:"token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// --- Helper Functions ---

// broadcastMessage sends a message to all connected clients.
func broadcastMessage(message ServerMessage) {
	connectionsMutex.Lock()
	defer connectionsMutex.Unlock()

	for conn, username := range connections {
		log.Printf("Broadcasting message type %s to user %s", message.Type, username)
		if err := conn.WriteJSON(message); err != nil {
			log.Printf("Error broadcasting message to user %s: %v. Closing connection.", username, err)
			conn.Close() // Close the connection
			delete(connections, conn) // Remove from map
		}
	}
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`)) // Fallback error
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, ErrorResponse{Error: message})
}

// --- HTTP Handlers ---

func registerHandler(authSvc ports.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
			return
		}

		var req AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}
		defer r.Body.Close()

		if req.Username == "" || req.Password == "" {
			respondError(w, http.StatusBadRequest, "Username and password are required")
			return
		}

		user, err := authSvc.Register(req.Username, req.Password)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrUserAlreadyExists):
				respondError(w, http.StatusConflict, "User already exists")
			default:
				log.Printf("Error during registration for user '%s': %v", req.Username, err)
				respondError(w, http.StatusInternalServerError, "Failed to register user")
			}
			return
		}
		respondJSON(w, http.StatusCreated, UserResponse{ID: user.ID, Username: user.Username})
	}
}

func loginHandler(authSvc ports.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			respondError(w, http.StatusUnauthorized, "Authorization header is missing")
			return
		}

		if !strings.HasPrefix(authHeader, "Basic ") {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			respondError(w, http.StatusUnauthorized, "Invalid authorization method, Basic method required")
			return
		}

		encodedCredentials := strings.TrimPrefix(authHeader, "Basic ")
		decodedCredentials, err := base64.StdEncoding.DecodeString(encodedCredentials)
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			respondError(w, http.StatusUnauthorized, "Invalid Base64 encoding in authorization header")
			return
		}

		credentials := strings.SplitN(string(decodedCredentials), ":", 2)
		if len(credentials) != 2 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			respondError(w, http.StatusUnauthorized, "Invalid format for credentials in authorization header")
			return
		}

		username := credentials[0]
		password := credentials[1]

		if username == "" || password == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			respondError(w, http.StatusUnauthorized, "Username or password cannot be empty")
			return
		}
		
		// Call VerifyUserCredentials to check username and password.
		// The returned user object is not strictly needed here since we're issuing a hardcoded token.
		// 'err' is already declared from base64.StdEncoding.DecodeString, so use '=' for assignment.
		_, err = authSvc.VerifyUserCredentials(username, password)
		if err != nil {
			log.Printf("Basic Auth: VerifyUserCredentials failed for user '%s': %v", username, err)
			if errors.Is(err, services.ErrInvalidCredentials) {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				respondError(w, http.StatusUnauthorized, "Invalid username or password")
			} else {
				// For other unexpected errors from VerifyUserCredentials
				respondError(w, http.StatusInternalServerError, "User authentication failed due to a server error")
			}
			return
		}

		// Credentials are valid. Respond with the hardcoded token.
		// Using services.HardcodedUserAuthToken ensures consistency.
		log.Printf("Basic Auth: User '%s' credentials verified successfully.", username)
		respondJSON(w, http.StatusOK, AuthResponse{Token: services.HardcodedUserAuthToken})
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request, messageRepo ports.MessageRepository, authSvc ports.AuthService) {
	// Extract token from query parameters
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		respondError(w, http.StatusUnauthorized, "Missing authentication token")
		return
	}

	// Validate token
	authUser, err := authSvc.ValidateToken(tokenString)
	if err != nil {
		log.Printf("Token validation failed: %v", err)
		// Differentiate between token format errors and actual invalid/expired tokens
		if errors.Is(err, services.ErrInvalidToken) {
			respondError(w, http.StatusUnauthorized, "Invalid or expired token")
		} else {
			respondError(w, http.StatusUnauthorized, "Authentication failed")
		}
		return
	}
	log.Printf("User %s (ID: %d) authenticated for WebSocket connection.", authUser.Username, authUser.ID)

	// Upgrade connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		// respondError is not suitable here as the response might have already been partially written
		// by the upgrader. http.Error is a simpler fallback if the upgrader fails early.
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		return
	}
	defer conn.Close()
	log.Printf("WebSocket connection established for user: %s (ID: %d)", authUser.Username, authUser.ID)

	// Add connection to map
	connectionsMutex.Lock()
	connections[conn] = authUser.Username
	connectionsMutex.Unlock()
	log.Printf("User %s added to active connections. Total connections: %d", authUser.Username, len(connections))

	// Defer removal of connection from map and logging
	defer func() {
		connectionsMutex.Lock()
		delete(connections, conn)
		connectionsMutex.Unlock()
		log.Printf("User %s disconnected. Total connections: %d", authUser.Username, len(connections))
	}()

	// Send message history
	recentMessages, err := messageRepo.GetRecentMessages(50) // Get last 50 messages
	if err != nil {
		log.Printf("Error fetching message history for user %s (ID: %d): %v", authUser.Username, authUser.ID, err)
		// Do not send an error message to the client for this, as history is not critical for chat functionality.
		// The client can still send/receive new messages.
	} else if len(recentMessages) > 0 {
		historyMsg := ServerMessage{
			Type:    "history",
			Payload: HistoryPayload{Messages: recentMessages},
		}
		log.Printf("Sending %d historical messages to user %s (ID: %d)", len(recentMessages), authUser.Username, authUser.ID)
		if err := conn.WriteJSON(historyMsg); err != nil {
			log.Printf("Error sending message history to user %s (ID: %d): %v", authUser.Username, authUser.ID, err)
			// Don't close connection, history is best-effort.
		}
	} else {
		log.Printf("No message history to send to user %s (ID: %d)", authUser.Username, authUser.ID)
	}

	// Message reading loop
	for {
		var clientMsg ClientMessage
		if err := conn.ReadJSON(&clientMsg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading JSON from user %s: %v", authUser.Username, err)
			} else {
				log.Printf("User %s WebSocket connection closed: %v", authUser.Username, err)
			}
			break // Exit loop on error (connection closed or other error)
		}

		log.Printf("Received message type '%s' from user '%s'", clientMsg.Type, authUser.Username)

		switch clientMsg.Type {
		case "sendMessage":
			var sendPayload SendMessagePayload
			if err := json.Unmarshal(clientMsg.Payload, &sendPayload); err != nil {
				log.Printf("Error unmarshalling sendMessage payload from user %s: %v", authUser.Username, err)
				errMsg := ServerMessage{Type: "error", Payload: ErrorPayload{Message: "Invalid sendMessage payload"}}
				if writeErr := conn.WriteJSON(errMsg); writeErr != nil {
					log.Printf("Error sending error message to user %s: %v", authUser.Username, writeErr)
				}
				continue // Next message
			}

			if sendPayload.Text == "" {
				errMsg := ServerMessage{Type: "error", Payload: ErrorPayload{Message: "Message text cannot be empty"}}
				if writeErr := conn.WriteJSON(errMsg); writeErr != nil {
					log.Printf("Error sending error message to user %s: %v", authUser.Username, writeErr)
				}
				continue
			}

			msgToSave := models.Message{
				User:      authUser.Username,
				Text:      sendPayload.Text,
				Timestamp: time.Now(), // Timestamp will be set by DB or here before saving
			}

			// Save message to repository
			savedMsg, err := messageRepo.SaveMessage(msgToSave) // Assuming SaveMessage returns the message with ID/Timestamp
			if err != nil {
				log.Printf("Error saving message from user %s: %v", authUser.Username, err)
				errMsg := ServerMessage{Type: "error", Payload: ErrorPayload{Message: "Failed to save message to database"}}
				if writeErr := conn.WriteJSON(errMsg); writeErr != nil {
					log.Printf("Error sending error message to user %s: %v", authUser.Username, writeErr)
				}
				continue
			}

			// Broadcast the new message
			newMessage := ServerMessage{
				Type:    "newMessage",
				Payload: NewMessagePayload{Message: savedMsg},
			}
			broadcastMessage(newMessage)
			log.Printf("Message from %s saved and broadcasted: %s", authUser.Username, savedMsg.Text)

		default:
			log.Printf("Unknown message type '%s' from user '%s'", clientMsg.Type, authUser.Username)
			errMsg := ServerMessage{Type: "error", Payload: ErrorPayload{Message: "Unknown message type: " + clientMsg.Type}}
			if err := conn.WriteJSON(errMsg); err != nil {
				log.Printf("Error sending unknown type error to user %s: %v", authUser.Username, err)
			}
		}
	}
}

// --- CORS Middleware ---
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		// Allow requests from our Flutter UI development server
		// For production, you might want to make this configurable.
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8081") 

		// Allow credentials - useful if you were using cookies or session-based auth
		// For token-based auth in headers, this isn't strictly necessary but doesn't hurt.
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept") // Added Accept
			w.Header().Set("Access-Control-Max-Age", "86400") // Cache preflight response for 1 day
			w.WriteHeader(http.StatusNoContent) // 204 No Content is standard for preflight success
			return
		}

		// Call the next handler in the chain for actual requests
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Determine dbPath early, but log its usage only if SQLite is selected.
	dbPath := os.Getenv("KEEPER_DB_PATH")
	if dbPath == "" {
		dbPath = "./keeper.db" // Default path
		// Logging for default path usage will happen inside sqlite/default cases
	}

	storageAdapter := os.Getenv("STORAGE_ADAPTER")
	log.Printf("STORAGE_ADAPTER environment variable is set to: '%s'", storageAdapter)

	var userRepo ports.UserRepository
	var messageRepo ports.MessageRepository
	var err error // Declare error variable for SQLite constructors

	switch storageAdapter {
	case "memory":
		log.Println("Initializing with in-memory storage adapters.")
		userRepo = authmemory.NewInMemoryUserRepository()
		// In-memory repo New functions typically don't return an error for simple map init
		// but we call Initialize for consistency with the interface, though it's a no-op.
		if err = userRepo.Initialize(); err != nil { // Updated call
			log.Fatalf("Failed to initialize in-memory user repository (no-op should not fail): %v", err) // Updated log
		}
		messageRepo = messagingmemory.NewInMemoryMessageRepository()
		if err = messageRepo.Initialize(); err != nil { // Updated call
			log.Fatalf("Failed to initialize in-memory message repository (no-op should not fail): %v", err) // Updated log
		}
		log.Println("In-memory repositories initialized successfully.")
	case "sqlite":
		log.Println("Initializing with SQLite storage adapters.")
		if dbPath == "./keeper.db" && os.Getenv("KEEPER_DB_PATH") == "" {
			log.Printf("KEEPER_DB_PATH not set, using default for SQLite: %s", dbPath)
		} else if os.Getenv("KEEPER_DB_PATH") != "" { // dbPath would be this value
			log.Printf("Using database path from KEEPER_DB_PATH for SQLite: %s", dbPath)
		} else { // dbPath is default but KEEPER_DB_PATH was set to "./keeper.db"
			log.Printf("Using explicitly set KEEPER_DB_PATH for SQLite: %s", dbPath)
		}

		userRepo, err = authsqlite.NewSQLiteUserRepository(dbPath)
		if err != nil {
			log.Fatalf("Failed to initialize SQLite user repository: %v", err)
		}
		log.Println("SQLite User repository initialized successfully.")

		messageRepo, err = messagingsqlite.NewSQLiteRepository(dbPath)
		if err != nil {
			log.Fatalf("Failed to initialize SQLite message repository: %v", err)
		}
		log.Println("SQLite Message repository initialized successfully.")
	default:
		log.Printf("No valid STORAGE_ADAPTER specified or value is '%s'. Defaulting to SQLite.", storageAdapter)
		if dbPath == "./keeper.db" && os.Getenv("KEEPER_DB_PATH") == "" {
			log.Printf("KEEPER_DB_PATH not set, using default for SQLite: %s", dbPath)
		} else if os.Getenv("KEEPER_DB_PATH") != "" {
			log.Printf("Using database path from KEEPER_DB_PATH for SQLite: %s", dbPath)
		} else {
			log.Printf("Using explicitly set KEEPER_DB_PATH for SQLite: %s", dbPath)
		}

		userRepo, err = authsqlite.NewSQLiteUserRepository(dbPath)
		if err != nil {
			log.Fatalf("Failed to initialize SQLite user repository (default): %v", err)
		}
		log.Println("SQLite User repository initialized successfully (default).")

		messageRepo, err = messagingsqlite.NewSQLiteRepository(dbPath)
		if err != nil {
			log.Fatalf("Failed to initialize SQLite message repository (default): %v", err)
		}
		log.Println("SQLite Message repository initialized successfully (default).")
	}

	// Initialize Auth Service
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-super-secret-key-for-dev" // TODO: Move to config/env for production
		log.Println("Warning: Using hardcoded JWT_SECRET. This is not secure for production.")
	}
	authSvc := services.NewAuthService(userRepo, jwtSecret)

	// Load fixtures
	// The path "./server/fixtures" assumes the binary is run from the project root (/app in Docker).
	log.Println("Attempting to load fixtures...")
	if err := seeder.LoadFixtures(userRepo, messageRepo, authSvc, "./server/fixtures"); err != nil {
		// Log the error but don't make it fatal, as per current design.
		log.Printf("Warning: Fixture loading process encountered an error: %v", err)
	} else {
		log.Println("Fixture loading process completed successfully.")
	}

	// Setup HTTP handlers with CORS middleware
	http.Handle("/api/register", corsMiddleware(registerHandler(authSvc))) // authSvc is ports.AuthService
	http.Handle("/api/login", corsMiddleware(loginHandler(authSvc)))   // authSvc is ports.AuthService
	http.Handle("/ws", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// messageRepo is ports.MessageRepository, authSvc is ports.AuthService
		wsHandler(w, r, messageRepo, authSvc)
	})))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}

	log.Printf("Starting server on :%s", port)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
