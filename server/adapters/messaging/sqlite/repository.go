package sqlite

import (
	"database/sql"
	"log"
	"time" // Included for completeness, may be used by specific timestamp logic later

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
// It takes a *sql.DB database connection as input.
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// InitSchema creates the necessary database schema (tables) if they don't already exist.
func (s *SQLiteRepository) InitSchema() error {
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

	// Diagnostic INSERT
	diagQueryMsgInsert := "INSERT INTO messages (user, text, timestamp) VALUES ('_diag_test_msg_user_', '_diag_test_text_', CURRENT_TIMESTAMP)"
	_, errInsertMsg := s.db.Exec(diagQueryMsgInsert)
	if errInsertMsg != nil {
		log.Printf("Diagnostic: FAILED to INSERT into messages table: %v", errInsertMsg)
	} else {
		log.Printf("Diagnostic: Successfully INSERTED a temporary row into messages table.")
		// Diagnostic DELETE
		diagQueryMsgDelete := "DELETE FROM messages WHERE user = '_diag_test_msg_user_'"
		_, errDeleteMsg := s.db.Exec(diagQueryMsgDelete)
		if errDeleteMsg != nil {
			log.Printf("Diagnostic: FAILED to DELETE temporary row from messages table: %v", errDeleteMsg)
		} else {
			log.Printf("Diagnostic: Successfully DELETED temporary row from messages table.")
		}
	}
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
