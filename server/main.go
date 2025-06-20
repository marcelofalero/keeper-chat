package main

import (
	"context" // Added for ValidateToken
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"errors"
	// authsqlite "keeper/server/adapters/auth/sqlite" // Old user repo
	messagingsqlite "keeper/server/adapters/messaging/sqlite" // For messageRepo
	"keeper/server/core/ports"
	"keeper/server/core/services"
	"keeper/server/models"                           // Old user model, still used by wsHandler temporarily
	usersmanagement "keeper/server/users-management" // New user management package

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

// --- WebSocket Upgrader ---
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// --- Request/Response Structs ---
// AuthRequest might not be needed if login/register handled by Kratos UI
/*
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
*/

// UserResponse might change or be removed
/*
type UserResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}
*/

// AuthResponse for token might change (Kratos uses cookies)
/*
type AuthResponse struct {
	Token string `json:"token"`
}
*/

type ErrorResponse struct {
	Error string `json:"error"`
}

// --- Helper Functions ---
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
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
// registerHandler and loginHandler are now effectively deprecated as Kratos handles these.
// They are commented out in the main function where routes are set up.
/*
func registerHandler(authSvc ports.AuthService) http.HandlerFunc {
	// ...
}

func loginHandler(authSvc ports.AuthService) http.HandlerFunc {
	// ...
}
*/

func wsHandler(w http.ResponseWriter, r *http.Request, messageRepo ports.MessageRepository, authSvc *services.AuthServiceImpl) {
	// Extract Kratos session cookie
	// The actual cookie name is 'ory_kratos_session'.
	sessionCookie, err := r.Cookie("ory_kratos_session")
	if err != nil {
		// If cookie is not found, it means the user is not logged in via Kratos.
		log.Printf("WebSocket: Kratos session cookie not found: %v", err)
		respondError(w, http.StatusUnauthorized, "Authentication required: Missing session cookie")
		return
	}
	kratosSessionToken := sessionCookie.Value

	// Validate Kratos session token
	// Pass context.Background() for now, or a request-specific context if available.
	authUser, err := authSvc.ValidateToken(context.Background(), kratosSessionToken)
	if err != nil {
		log.Printf("WebSocket: Kratos session validation failed: %v", err)
		if errors.Is(err, services.ErrInvalidToken) {
			respondError(w, http.StatusUnauthorized, "Invalid or expired session")
		} else {
			respondError(w, http.StatusUnauthorized, "Authentication failed")
		}
		return
	}
	// authUser is now *usersmanagement.User. We need to adapt how its fields are used.
	log.Printf("User %s (ID: %s) authenticated via Kratos for WebSocket connection.", authUser.Email, authUser.ID)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		return
	}
	defer conn.Close()
	log.Printf("WebSocket connection established for user: %s (Kratos ID: %s)", authUser.Email, authUser.ID)

	// Adapt sampleMessage to use fields from usersmanagement.User
	sampleMessage := models.Message{ // This is the old models.Message
		User:      authUser.Email, // Using email as username, or choose another trait
		Text:      "Hello from the Kratos authenticated server!",
		Timestamp: time.Now(),
	}

	err = messageRepo.SaveMessage(sampleMessage)
	if err != nil {
		log.Printf("Error saving message from user %s: %v", authUser.Email, err)
	} else {
		log.Printf("Message from %s saved successfully", authUser.Email)
	}

	if err := conn.WriteJSON(sampleMessage); err != nil {
		log.Printf("Error writing JSON to WebSocket for user %s: %v", authUser.Email, err)
	}
}

// --- CORS Middleware ---
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8081") // Kratos UI / Test UI
		w.Header().Set("Access-Control-Allow-Credentials", "true")             // Important for Kratos cookies

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, X-Session-Token") // Added X-Session-Token
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./keeper.db"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database at %s: %v", dbPath, err)
	}
	defer db.Close()

	messageRepo := messagingsqlite.NewSQLiteRepository(db)
	if err := messageRepo.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize message database schema: %v", err)
	}

	// Remove old user repository
	// userRepo := authsqlite.NewSQLiteUserRepository(db)
	// if err := userRepo.InitUserSchema(); err != nil {
	// 	log.Fatalf("Failed to initialize user database schema: %v", err)
	// }

	// Initialize Kratos Client and User Service
	kratosAdminURL := os.Getenv("KRATOS_ADMIN_URL")
	if kratosAdminURL == "" {
		kratosAdminURL = "http://kratos:4434" // Default for Docker Compose
		log.Printf("KRATOS_ADMIN_URL not set, using default: %s", kratosAdminURL)
	}
	kratosPublicURL := os.Getenv("KRATOS_PUBLIC_URL")
	if kratosPublicURL == "" {
		// This should be the URL Kratos is reachable on from the perspective of this server
		// when making calls to /sessions/whoami (ToSession).
		// If running in Docker Compose, 'kratos:4433' (public port) is often correct.
		kratosPublicURL = "http://kratos:4433"
		log.Printf("KRATOS_PUBLIC_URL not set, using default: %s", kratosPublicURL)
	}

	kratosClient, err := usersmanagement.NewKratosClient(kratosAdminURL, kratosPublicURL)
	if err != nil {
		log.Fatalf("Failed to create Kratos client: %v", err)
	}
	kratosUserService := usersmanagement.NewUserService(kratosClient)

	// Initialize Auth Service with the new Kratos User Service
	// jwtSecret := os.Getenv("JWT_SECRET") // No longer needed for Kratos sessions
	// if jwtSecret == "" {
	// 	jwtSecret = "your-super-secret-key-for-dev"
	// 	log.Println("Warning: Using hardcoded JWT_SECRET. This is not secure for production.")
	// }
	authSvc := services.NewAuthService(kratosUserService /*, jwtSecret */) // Pass Kratos user service

	// Setup HTTP handlers with CORS middleware
	// http.Handle("/api/register", corsMiddleware(registerHandler(authSvc))) // Deprecated
	// http.Handle("/api/login", corsMiddleware(loginHandler(authSvc)))       // Deprecated

	// Ensure wsHandler gets the correctly typed authSvc
	http.Handle("/ws", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsHandler(w, r, messageRepo, authSvc) // authSvc is now *services.AuthServiceImpl
	})))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on :%s", port)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
