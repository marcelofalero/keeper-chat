package sqlite_test

import (
	"database/sql"
	"keeper/server/adapters/auth/sqlite" // Package being tested
	"keeper/server/models"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"golang.org/x/crypto/bcrypt"
)

// setupTestDB creates an in-memory SQLite database for testing.
// Redefined here for simplicity; in a larger project, share via a test helper package.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("Failed to ping in-memory database: %v", err)
	}
	return db
}

func TestMain(m *testing.M) {
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestInitUserSchema(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewSQLiteUserRepository(db)

	err := repo.InitUserSchema()
	if err != nil {
		t.Fatalf("InitUserSchema() failed: %v", err)
	}

	// Query sqlite_master to verify table creation
	var tableName string
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name='users';"
	err = db.QueryRow(query).Scan(&tableName)
	if err != nil {
		if err == sql.ErrNoRows {
			t.Fatal("'users' table not found in schema after InitUserSchema()")
		}
		t.Fatalf("Error querying schema for users table: %v", err)
	}
	if tableName != "users" {
		t.Errorf("Expected table name 'users', got '%s'", tableName)
	}

	// Verify columns
	rows, err := db.Query("PRAGMA table_info(users);")
	if err != nil {
		t.Fatalf("Failed to query table info for users: %v", err)
	}
	defer rows.Close()

	expectedColumns := map[string]struct{ Type string; Unique bool }{
		"id":            {Type: "INTEGER", Unique: false}, // PK implies unique and not null
		"username":      {Type: "TEXT", Unique: true},
		"password_hash": {Type: "TEXT", Unique: false},
	}
	foundColumns := make(map[string]string)
	for rows.Next() {
		var cid int
		var name string
		var typeName string
		var notnull int
		var dfltValue sql.NullString
		var pk int
		// For UNIQUE constraint, PRAGMA index_list('users') and PRAGMA index_info(index_name) would be needed.
		// For simplicity, we'll assume the CREATE TABLE statement's UNIQUE is effective.
		if err := rows.Scan(&cid, &name, &typeName, &notnull, &dfltValue, &pk); err != nil {
			t.Fatalf("Failed to scan row from table_info: %v", err)
		}
		foundColumns[name] = typeName
	}

	for colName, colDetails := range expectedColumns {
		foundType, exists := foundColumns[colName]
		if !exists {
			t.Errorf("Expected column '%s' not found", colName)
		}
		if foundType != colDetails.Type {
			t.Errorf("For column '%s', expected type '%s', got '%s'", colName, colDetails.Type, foundType)
		}
		// Verifying UNIQUE constraint directly via PRAGMA table_info is not straightforward.
		// It's typically tested by attempting to insert duplicate values (see TestCreateUser).
	}
}

func TestCreateUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewSQLiteUserRepository(db)
	if err := repo.InitUserSchema(); err != nil {
		t.Fatalf("InitUserSchema() failed: %v", err)
	}

	password := "password123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	t.Run("Successful User Creation", func(t *testing.T) {
		sampleUser := models.User{
			Username:     "testuser1",
			PasswordHash: string(hashedPassword),
		}
		err := repo.CreateUser(sampleUser)
		if err != nil {
			t.Fatalf("CreateUser() failed: %v", err)
		}

		// Query DB to verify insertion
		var (
			id           int64
			username     string
			passwordHash string
		)
		query := "SELECT id, username, password_hash FROM users WHERE username = ?;"
		err = db.QueryRow(query, sampleUser.Username).Scan(&id, &username, &passwordHash)
		if err != nil {
			t.Fatalf("Failed to query saved user: %v", err)
		}
		if id == 0 {
			t.Error("Expected auto-generated ID to be non-zero")
		}
		if username != sampleUser.Username {
			t.Errorf("Expected username '%s', got '%s'", sampleUser.Username, username)
		}
		if passwordHash != sampleUser.PasswordHash {
			t.Errorf("Expected password hash '%s', got '%s'", sampleUser.PasswordHash, passwordHash)
		}
	})

	t.Run("Create User with Existing Username", func(t *testing.T) {
		// First user already created in the sub-test above, or create one here if tests are fully isolated.
		// Let's assume subtests might share DB state if not careful, so make username unique for this subtest.
		initialUser := models.User{Username: "existinguser", PasswordHash: "somehash"}
		if err := repo.CreateUser(initialUser); err != nil {
			t.Fatalf("Setup: Failed to create initial user: %v", err)
		}
		
		duplicateUser := models.User{
			Username:     "existinguser", // Same username
			PasswordHash: string(hashedPassword),
		}
		err := repo.CreateUser(duplicateUser)
		if err == nil {
			t.Fatal("Expected error when creating user with duplicate username, got nil")
		}
		// The exact error might depend on the SQLite driver or GORM,
		// but it should indicate a constraint violation.
		// e.g., "UNIQUE constraint failed: users.username"
		// The repository method CreateUser doesn't wrap this error currently.
		// For now, just checking that an error occurred is a basic test.
		// log.Printf("Info: Received error for duplicate username: %v", err) // For debugging
		// A more robust test would check for a specific error type or message if the repo guarantees it.
	})
}

func TestGetUserByUsername(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewSQLiteUserRepository(db)
	if err := repo.InitUserSchema(); err != nil {
		t.Fatalf("InitUserSchema() failed: %v", err)
	}

	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	savedUser := models.User{
		Username:     "getmebyusername",
		PasswordHash: string(hashedPassword),
	}
	// We need the ID, but CreateUser doesn't return it.
	// So, after creating, we'd typically query to get the full user for comparison if ID is needed.
	// However, GetUserByUsername itself doesn't rely on knowing the ID beforehand.
	if err := repo.CreateUser(savedUser); err != nil {
		t.Fatalf("Failed to create sample user for GetUserByUsername test: %v", err)
	}
	// Let's retrieve the ID for completeness if we want to assert it.
	var actualSavedID int64
	err := db.QueryRow("SELECT id FROM users WHERE username = ?", savedUser.Username).Scan(&actualSavedID)
    if err != nil {
        t.Fatalf("Failed to query ID of saved user: %v", err)
    }
    savedUser.ID = actualSavedID


	t.Run("Existing User", func(t *testing.T) {
		retrievedUser, err := repo.GetUserByUsername(savedUser.Username)
		if err != nil {
			t.Fatalf("GetUserByUsername() failed: %v", err)
		}
		if retrievedUser == nil {
			t.Fatal("Expected user, got nil")
		}
		if retrievedUser.ID != savedUser.ID {
             t.Errorf("Expected ID %d, got %d", savedUser.ID, retrievedUser.ID)
        }
		if retrievedUser.Username != savedUser.Username {
			t.Errorf("Expected username '%s', got '%s'", savedUser.Username, retrievedUser.Username)
		}
		if retrievedUser.PasswordHash != savedUser.PasswordHash {
			t.Errorf("Expected password hash '%s', got '%s'", savedUser.PasswordHash, retrievedUser.PasswordHash)
		}
	})

	t.Run("Non-existent User", func(t *testing.T) {
		retrievedUser, err := repo.GetUserByUsername("nonexistentuser")
		if err != nil {
			// The repository method returns (nil, nil) for sql.ErrNoRows, so no error expected here from the method itself.
			// If it wrapped sql.ErrNoRows, then we would check for that.
			t.Fatalf("Expected no error for non-existent user from repo method, got %v", err)
		}
		if retrievedUser != nil {
			t.Fatalf("Expected nil user for non-existent username, got %+v", retrievedUser)
		}
		// To verify sql.ErrNoRows was the underlying cause, you'd need the repo to return it,
		// or you test against the DB directly. The current repo abstraction hides it.
		// The repo method `GetUserByUsername` returns `(nil, nil)` when `sql.ErrNoRows` occurs.
	})
}

func TestGetUserByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewSQLiteUserRepository(db)
	if err := repo.InitUserSchema(); err != nil {
		t.Fatalf("InitUserSchema() failed: %v", err)
	}

	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	sampleUser := models.User{
		Username:     "getmebyid",
		PasswordHash: string(hashedPassword),
	}
	if err := repo.CreateUser(sampleUser); err != nil {
		t.Fatalf("Failed to create sample user for GetUserByID test: %v", err)
	}
	
	// Retrieve the ID of the inserted user
	var userID int64
	err := db.QueryRow("SELECT id FROM users WHERE username = ?", sampleUser.Username).Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to retrieve ID of created user: %v", err)
	}
	sampleUser.ID = userID // Update sampleUser with the actual ID for comparison


	t.Run("Existing User", func(t *testing.T) {
		retrievedUser, err := repo.GetUserByID(sampleUser.ID)
		if err != nil {
			t.Fatalf("GetUserByID() failed: %v", err)
		}
		if retrievedUser == nil {
			t.Fatal("Expected user, got nil")
		}
		if retrievedUser.ID != sampleUser.ID {
			t.Errorf("Expected ID %d, got %d", sampleUser.ID, retrievedUser.ID)
		}
		if retrievedUser.Username != sampleUser.Username {
			t.Errorf("Expected username '%s', got '%s'", sampleUser.Username, retrievedUser.Username)
		}
		if retrievedUser.PasswordHash != sampleUser.PasswordHash {
			t.Errorf("Expected password hash '%s', got '%s'", sampleUser.PasswordHash, retrievedUser.PasswordHash)
		}
	})

	t.Run("Non-existent User", func(t *testing.T) {
		nonExistentID := int64(99999)
		retrievedUser, err := repo.GetUserByID(nonExistentID)
		if err != nil {
			// Similar to GetUserByUsername, the repo method returns (nil, nil) for sql.ErrNoRows.
			t.Fatalf("Expected no error for non-existent user ID from repo method, got %v", err)
		}
		if retrievedUser != nil {
			t.Fatalf("Expected nil user for non-existent ID, got %+v", retrievedUser)
		}
	})
}
```
