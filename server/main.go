package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"errors"
	"keeper/server/adapters/sqlite"
	"keeper/server/core/ports"
	"keeper/server/core/services"
	"keeper/server/models"
	// "strings" // No longer needed directly in wsHandler for Bearer token

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
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

type UserResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// --- Helper Functions ---

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

		tokenString, err := authSvc.Login(req.Username, req.Password)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrInvalidCredentials):
				respondError(w, http.StatusUnauthorized, "Invalid username or password")
			default:
				log.Printf("Error during login for user '%s': %v", req.Username, err)
				respondError(w, http.StatusInternalServerError, "Failed to login")
			}
			return
		}
		respondJSON(w, http.StatusOK, AuthResponse{Token: tokenString})
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
	log.Printf("WebSocket connection established for user: %s", authUser.Username)

	// Simulate receiving and processing a message (conceptual)
	// In a real app, you'd have a loop here: conn.ReadJSON(&msg), process, conn.WriteJSON(response)
	sampleMessage := models.Message{
		User:      authUser.Username, // Use authenticated user's username
		Text:      "Hello from the authenticated server!",
		Timestamp: time.Now(),
		// ID will be generated by DB
	}

	// TODO: Implement actual command processing or message relay logic here.
	// For now, we'll just save the sample message.
	err = messageRepo.SaveMessage(sampleMessage)
	if err != nil {
		log.Printf("Error saving message from user %s: %v", authUser.Username, err)
		// Inform client about the error (optional, depends on WebSocket protocol)
		// conn.WriteJSON(ErrorResponse{Error: "Failed to save message"})
		// return // Or continue if it's not a critical error
	} else {
		log.Printf("Message from %s saved successfully", authUser.Username)
	}

	// Conceptually, send the processed (and now ID'd) message back or to other clients
	// Fetch the message again if you need the ID generated by the database
	// For simplicity, we'll use the current sampleMessage struct.
	// This part is illustrative as we don't have a client expecting this specific message format back yet.
	if err := conn.WriteJSON(sampleMessage); err != nil {
		log.Printf("Error writing JSON to WebSocket for user %s: %v", authUser.Username, err)
	}
}

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./keeper.db" // Default path
	}

	// Open SQLite database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database at %s: %v", dbPath, err)
	}
	defer db.Close()

	// Create and initialize message repository
	messageRepo := sqlite.NewSQLiteRepository(db)
	if err := messageRepo.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize message database schema: %v", err)
	}

	// Create and initialize user repository
	userRepo := sqlite.NewSQLiteUserRepository(db)
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

	// Setup HTTP handlers
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler(w, r, messageRepo, authSvc) // Pass authSvc, though wsHandler doesn't use it yet
	})
	http.HandleFunc("/api/register", registerHandler(authSvc))
	http.HandleFunc("/api/login", loginHandler(authSvc))

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
