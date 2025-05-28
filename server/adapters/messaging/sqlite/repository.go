package sqlite

import (
	"database/sql"
	"fmt" // For error wrapping
	"log"
	"os"             // Added for os.Stat and os.MkdirAll
	"path/filepath"  // Added for filepath.Dir
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"keeper/server/core/ports"
	"keeper/server/models"
)

// Verify SQLiteRepository implements ports.MessageRepository
var _ ports.MessageRepository = (*SQLiteRepository)(nil)

// SQLiteRepository implements the ports.MessageRepository interface using SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new instance of SQLiteRepository.
// It handles database directory creation, connection, and schema initialization.
func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	// Ensure the database directory exists
	dbDir := filepath.Dir(dbPath)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		log.Printf("Message repository: Database directory %s does not exist, creating it.", dbDir)
		if err = os.MkdirAll(dbDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create message DB directory %s: %w", dbDir, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to check message DB directory %s: %w", dbDir, err)
	}

	// Construct DSN and open database connection
	dbDSN := dbPath + "?_foreign_keys=on"
	log.Printf("Message repository: Opening database connection to: %s (using DSN: %s)", dbPath, dbDSN)
	db, err := sql.Open("sqlite3", dbDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open message DB at %s: %w", dbPath, err)
	}

	// Ping the database to verify the connection
	if err = db.Ping(); err != nil {
		db.Close() // Close DB if ping fails
		return nil, fmt.Errorf("failed to ping message DB at %s (DSN: %s): %w", dbPath, dbDSN, err)
	}
	log.Printf("Message repository: Successfully connected to and pinged database: %s", dbPath)

	repo := &SQLiteRepository{db: db}

	// Initialize schema
	log.Printf("Message repository: Attempting to initialize schema for database at: %s", dbPath) // Updated log
	if err = repo.Initialize(); err != nil { // Updated call
		db.Close() // Close DB if schema initialization fails
		return nil, fmt.Errorf("failed to initialize schema for DB at %s: %w", dbPath, err) // Updated error message
	}
	// The Initialize method itself logs success/failure of diagnostics.

	return repo, nil
}

// Initialize creates the necessary database schema (tables) if they don't already exist.
func (s *SQLiteRepository) Initialize() error {
	query := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user TEXT,
		text TEXT,
		timestamp DATETIME
	);`
	_, err := s.db.Exec(query)
	if err != nil {
		log.Printf("Error initializing schema: %v", err)
		return err
	}
	log.Println("Database schema initialized successfully.")
	// Removed diagnostic INSERT/DELETE block for messages table.
	return nil
}

// SaveMessage saves a new message to the SQLite database.
// It returns the saved message with its ID and server-set timestamp.
func (s *SQLiteRepository) SaveMessage(msg models.Message) (models.Message, error) {
	// Ensure timestamp is set (UTC)
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now().UTC()
	} else {
		msg.Timestamp = msg.Timestamp.UTC() // Ensure it's UTC
	}

	query := "INSERT INTO messages (user, text, timestamp) VALUES (?, ?, ?) RETURNING id"
	stmt, err := s.db.Prepare(query)
	if err != nil {
		log.Printf("Error preparing save message statement: %v", err)
		return msg, err
	}
	defer stmt.Close()

	// SQLite's RETURNING clause for INSERT gives back the specified columns.
	// Here, we ask for 'id'.
	err = stmt.QueryRow(msg.User, msg.Text, msg.Timestamp).Scan(&msg.ID)
	if err != nil {
		log.Printf("Error executing save message statement and retrieving ID: %v", err)
		return msg, err
	}
	log.Printf("Message saved with ID: %d, Timestamp: %s", msg.ID, msg.Timestamp.Format(time.RFC3339))
	return msg, nil
}

// GetMessages retrieves all messages from the SQLite database, ordered by timestamp ASC.
func (s *SQLiteRepository) GetMessages() ([]models.Message, error) {
	query := "SELECT id, user, text, timestamp FROM messages ORDER BY timestamp ASC"
	rows, err := s.db.Query(query)
	if err != nil {
		log.Printf("Error querying all messages: %v", err)
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		// Using DATETIME in SQLite, so it should be directly scannable to time.Time
		// if the driver supports it, or to a string then parse.
		// The go-sqlite3 driver handles time.Time conversion automatically for DATETIME columns.
		if err := rows.Scan(&msg.ID, &msg.User, &msg.Text, &msg.Timestamp); err != nil {
			log.Printf("Error scanning message row: %v", err)
			return nil, err
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating all message rows: %v", err)
		return nil, err
	}

	return messages, nil
}

// GetRecentMessages retrieves a specified number of recent messages from the SQLite database,
// ordered by timestamp chronologically (oldest first).
func (s *SQLiteRepository) GetRecentMessages(limit int) ([]models.Message, error) {
	// Query for messages in reverse chronological order (newest first)
	query := "SELECT id, user, text, timestamp FROM messages ORDER BY timestamp DESC LIMIT ?"
	rows, err := s.db.Query(query, limit)
	if err != nil {
		log.Printf("Error querying recent messages: %v", err)
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		// The go-sqlite3 driver handles time.Time conversion automatically for DATETIME columns.
		if err := rows.Scan(&msg.ID, &msg.User, &msg.Text, &msg.Timestamp); err != nil {
			log.Printf("Error scanning recent message row: %v", err)
			return nil, err
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating recent message rows: %v", err)
		return nil, err
	}

	// Reverse the slice to return messages in chronological order (oldest first)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
