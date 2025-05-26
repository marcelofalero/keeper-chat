package sqlite

import (
	"database/sql"
	"fmt" // For error wrapping
	"log"
	"os"             // Added for os.Stat and os.MkdirAll
	"path/filepath"  // Added for filepath.Dir

	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"keeper/server/core/ports"
	"keeper/server/models"
)

// Verify SQLiteUserRepository implements ports.UserRepository
var _ ports.UserRepository = (*SQLiteUserRepository)(nil)

// SQLiteUserRepository implements the ports.UserRepository interface using SQLite.
type SQLiteUserRepository struct {
	db *sql.DB
}

// NewSQLiteUserRepository creates a new instance of SQLiteUserRepository.
// It handles database directory creation, connection, and schema initialization.
func NewSQLiteUserRepository(dbPath string) (*SQLiteUserRepository, error) {
	// Ensure the database directory exists
	dbDir := filepath.Dir(dbPath)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		log.Printf("User repository: Database directory %s does not exist, creating it.", dbDir)
		if err = os.MkdirAll(dbDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create user DB directory %s: %w", dbDir, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to check user DB directory %s: %w", dbDir, err)
	}

	// Construct DSN and open database connection
	dbDSN := dbPath + "?_foreign_keys=on"
	log.Printf("User repository: Opening database connection to: %s (using DSN: %s)", dbPath, dbDSN)
	db, err := sql.Open("sqlite3", dbDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open user DB at %s: %w", dbPath, err)
	}

	// Ping the database to verify the connection
	if err = db.Ping(); err != nil {
		db.Close() // Close DB if ping fails
		return nil, fmt.Errorf("failed to ping user DB at %s (DSN: %s): %w", dbPath, dbDSN, err)
	}
	log.Printf("User repository: Successfully connected to and pinged database: %s", dbPath)

	repo := &SQLiteUserRepository{db: db}

	// Initialize schema
	log.Printf("User repository: Attempting to initialize user schema for database at: %s", dbPath)
	if err = repo.InitUserSchema(); err != nil {
		db.Close() // Close DB if schema initialization fails
		return nil, fmt.Errorf("failed to initialize user schema for DB at %s: %w", dbPath, err)
	}
	// The InitUserSchema method itself logs success/failure of diagnostics.

	return repo, nil
}

// InitUserSchema creates the `users` table if it doesn't already exist.
func (s *SQLiteUserRepository) InitUserSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE,
		password_hash TEXT
	);`
	_, err := s.db.Exec(query)
	if err != nil {
		log.Printf("Error initializing user schema: %v", err)
		return err
	}
	log.Println("User schema initialized successfully.")
	// Removed diagnostic INSERT/DELETE block for users table.
	return nil
}

// CreateUser adds a new user to the database.
func (s *SQLiteUserRepository) CreateUser(user models.User) error {
	query := "INSERT INTO users (username, password_hash) VALUES (?, ?)"
	stmt, err := s.db.Prepare(query)
	if err != nil {
		log.Printf("Error preparing create user statement: %v", err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(user.Username, user.PasswordHash)
	if err != nil {
		log.Printf("Error executing create user statement: %v", err)
		// Consider checking for specific errors like UNIQUE constraint violation
		return err
	}
	return nil
}

// GetUserByUsername retrieves a user by their username.
// Returns (nil, nil) if the user is not found.
func (s *SQLiteUserRepository) GetUserByUsername(username string) (*models.User, error) {
	query := "SELECT id, username, password_hash FROM users WHERE username = ?"
	row := s.db.QueryRow(query, username)

	var user models.User
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found
		}
		log.Printf("Error scanning user row by username '%s': %v", username, err)
		return nil, err
	}
	return &user, nil
}

// GetUserByID retrieves a user by their ID.
// Returns (nil, nil) if the user is not found.
func (s *SQLiteUserRepository) GetUserByID(id int64) (*models.User, error) {
	query := "SELECT id, username, password_hash FROM users WHERE id = ?"
	row := s.db.QueryRow(query, id)

	var user models.User
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found
		}
		log.Printf("Error scanning user row by ID '%d': %v", id, err)
		return nil, err
	}
	return &user, nil
}
