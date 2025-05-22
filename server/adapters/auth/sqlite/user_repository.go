package sqlite

import (
	"database/sql"
	"log"

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
func NewSQLiteUserRepository(db *sql.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{db: db}
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

	// Diagnostic INSERT
	diagQueryUserInsert := "INSERT INTO users (username, password_hash) VALUES ('_diag_test_user_', '_diag_hash_')"
	_, errInsert := s.db.Exec(diagQueryUserInsert)
	if errInsert != nil {
		log.Printf("Diagnostic: FAILED to INSERT into users table: %v", errInsert)
	} else {
		log.Printf("Diagnostic: Successfully INSERTED a temporary row into users table.")
		// Diagnostic DELETE
		diagQueryUserDelete := "DELETE FROM users WHERE username = '_diag_test_user_'"
		_, errDelete := s.db.Exec(diagQueryUserDelete)
		if errDelete != nil {
			log.Printf("Diagnostic: FAILED to DELETE temporary row from users table: %v", errDelete)
		} else {
			log.Printf("Diagnostic: Successfully DELETED temporary row from users table.")
		}
	}
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
