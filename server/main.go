package main

import (
	"database/sql"
	"encoding/json"
	"encoding/base64" // Added for Basic Auth
	"log"
	"net/http"
	"os"
	"path/filepath" // Added for directory operations
	"strings"       // Added for Basic Auth
	"time"

	"errors"
	authsqlite "keeper/server/adapters/auth/sqlite" // For userRepo
	messagingsqlite "keeper/server/adapters/messaging/sqlite" // For messageRepo
	"keeper/server/core/ports"
	"keeper/server/core/services"
	"keeper/server/models"
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
		_, err := authSvc.VerifyUserCredentials(username, password)
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
	dbPath := os.Getenv("KEEPER_DB_PATH") // Changed from DB_PATH
	if dbPath == "" {
		dbPath = "./keeper.db" // Default path
	}

	// Ensure the database directory exists
	dbDir := filepath.Dir(dbPath)
	// Check if dbDir is "." (current directory), no need to create if so,
	// unless dbPath itself is just a filename like "keeper.db" (dbDir will be ".")
	// or if it's like "./data/keeper.db" (dbDir will be "./data").
	// The os.MkdirAll will handle the "." case correctly (no-op).
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		log.Printf("Database directory %s does not exist, creating it.", dbDir)
		if err := os.MkdirAll(dbDir, 0755); err != nil { // 0755 gives rwx for owner, rx for group/other
			log.Fatalf("Failed to create database directory %s: %v", dbDir, err)
		}
	} else if err != nil {
		// This covers other errors from os.Stat, like permission issues.
		log.Fatalf("Failed to check database directory %s: %v", dbDir, err)
	}

	// Open SQLite database connection with foreign key support enabled
	dbDSN := dbPath + "?_foreign_keys=on"
	log.Printf("Opening database connection to: %s (using DSN: %s)", dbPath, dbDSN)
	db, err := sql.Open("sqlite3", dbDSN)
	if err != nil {
		log.Fatalf("Failed to open database at %s: %v", dbPath, err)
	}
	defer db.Close()

	// Ping the database to verify the connection
	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database at %s (DSN: %s): %v", dbPath, dbDSN, err)
	}
	log.Printf("Successfully connected to and pinged database: %s", dbPath)

	// Create and initialize message repository
	messageRepo := messagingsqlite.NewSQLiteRepository(db) // This is ports.MessageRepository
	log.Printf("Attempting to initialize message schema for database at: %s", dbPath)
	if err := messageRepo.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize message database schema: %v", err)
	}

	// Create and initialize user repository
	userRepo := authsqlite.NewSQLiteUserRepository(db) // This is ports.UserRepository
	log.Printf("Attempting to initialize user schema for database at: %s", dbPath)
	if err := userRepo.InitUserSchema(); err != nil {
		log.Fatalf("Failed to initialize user database schema: %v", err)
	}

	// Initialize Auth Service
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-super-secret-key-for-dev" // TODO: Move to config/env for production
		log.Println("Warning: Using hardcoded JWT_SECRET. This is not secure for production.")
	}
	authSvc := services.NewAuthService(userRepo, jwtSecret)

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
