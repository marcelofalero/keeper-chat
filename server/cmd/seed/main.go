package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath" // For path manipulation

	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"golang.org/x/crypto/bcrypt"    // For password hashing
	// Assuming models are directly accessible if this cmd is within the server module context
	// For simplicity, we'll define simplified User and Message structs here,
	// or adjust import paths if running `go run` from server root makes `keeper/server/models` accessible.
	// Let's try to use existing models by adjusting relative paths for `go run`.
	// This might be tricky with `go run server/cmd/seed/main.go`.
	// A better way is to make this a proper command that builds and runs from server root,
	// or configure DB path via env var.
	// For now, let's assume DB path is fixed relative to project root.
)

const dbPath = "./data/keeper.db" // Path relative to project root

// Simplified User struct for seeding - ideally use models.User
type SeedUser struct {
	ID           int64
	Username     string
	PasswordHash string
}

// Simplified Message struct for seeding - ideally use models.Message
type SeedMessage struct {
	ID        int64
	UserID    int64  // Assuming we link messages to user IDs
	User      string // Username, denormalized for simplicity in this example
	Text      string
	Timestamp string // Use string for simplicity, format "YYYY-MM-DD HH:MM:SS"
}

func main() {
	log.Println("Seeding database...")

	// Ensure the data directory exists
	dataDir := filepath.Dir(dbPath)
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		log.Printf("Data directory %s does not exist. Creating...", dataDir)
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Fatalf("Error creating data directory %s: %v", dataDir, err)
		}
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Error opening database at %s: %v", dbPath, err)
	}
	defer db.Close()

	// Initialize schemas (taken from existing repository InitSchema methods)
	// This makes the seeder idempotent regarding schema.
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE,
		password_hash TEXT
	);`)
	if err != nil {
		log.Fatalf("Error creating users schema: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user TEXT, 
		text TEXT,
		timestamp DATETIME
	);`)
	if err != nil {
		log.Fatalf("Error creating messages schema: %v", err)
	}

	log.Println("Schemas initialized/verified.")

	// Seed Users
	usersToSeed := []struct {
		username string
		password string
	}{
		{"Alice", "password123"},
		{"Bob", "securePa$$"},
		{"Charlie", "charliePass"},
	}

	var seededUsers []SeedUser

	for _, u := range usersToSeed {
		// Check if user already exists
		var existingID int64
		err := db.QueryRow("SELECT id FROM users WHERE username = ?", u.username).Scan(&existingID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Error checking for user %s: %v. Skipping.", u.username, err)
			continue
		}
		if err == nil { // User exists
			log.Printf("User %s already exists with ID %d. Skipping.", u.username, existingID)
			// Add to seededUsers so messages can still be created if desired
			// Or fetch password hash if needed for other logic (not needed here)
			seededUsers = append(seededUsers, SeedUser{ID: existingID, Username: u.username})
			continue
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Error hashing password for %s: %v. Skipping.", u.username, err)
			continue
		}

		res, err := db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", u.username, string(hashedPassword))
		if err != nil {
			log.Printf("Error inserting user %s: %v. Skipping.", u.username, err)
			continue
		}
		id, _ := res.LastInsertId()
		log.Printf("Seeded user: %s (ID: %d)", u.username, id)
		seededUsers = append(seededUsers, SeedUser{ID: id, Username: u.username, PasswordHash: string(hashedPassword)})
	}

	if len(seededUsers) < 2 {
		log.Println("Not enough users were successfully seeded/found to create messages.")
		log.Println("Seeding complete.")
		return
	}

	// Seed Messages
	messagesToSeed := []SeedMessage{
		{User: seededUsers[0].Username, Text: "Hello Bob!", Timestamp: "2023-01-01 10:00:00"},
		{User: seededUsers[1].Username, Text: "Hi Alice! How are you?", Timestamp: "2023-01-01 10:01:00"},
		{User: seededUsers[0].Username, Text: "I'm good, thanks! Planning any TTRPG sessions?", Timestamp: "2023-01-01 10:02:00"},
		{User: seededUsers[1].Username, Text: "Yes! This weekend. You in?", Timestamp: "2023-01-01 10:03:00"},
	}
	if len(seededUsers) > 2 { // Add messages from Charlie if available
		messagesToSeed = append(messagesToSeed, SeedMessage{User: seededUsers[2].Username, Text: "Hey everyone, what's up?", Timestamp: "2023-01-01 10:00:30"})
		messagesToSeed = append(messagesToSeed, SeedMessage{User: seededUsers[0].Username, Text: "Hey Charlie! Just chatting.", Timestamp: "2023-01-01 10:01:30"})
	}

	for _, m := range messagesToSeed {
		// Basic check to avoid duplicate messages based on text and user (very simplistic)
		var existingID int64
		err := db.QueryRow("SELECT id FROM messages WHERE user = ? AND text = ? AND timestamp = ?", m.User, m.Text, m.Timestamp).Scan(&existingID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Error checking for message from %s: %v. Skipping.", m.User, err)
			continue
		}
		if err == nil { // Message exists
			log.Printf("Message from %s ('%s') already exists. Skipping.", m.User, m.Text)
			continue
		}

		_, err = db.Exec("INSERT INTO messages (user, text, timestamp) VALUES (?, ?, ?)", m.User, m.Text, m.Timestamp)
		if err != nil {
			log.Printf("Error inserting message from %s: %v. Skipping.", m.User, err)
			continue
		}
		log.Printf("Seeded message from %s: %s", m.User, m.Text)
	}

	log.Println("Seeding complete.")
}
