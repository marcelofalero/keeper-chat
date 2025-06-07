package memory_test

import (
	"testing"

	"keeper/server/adapters/auth/memory" // The package being tested
	"keeper/server/models"
	// "github.com/stretchr/testify/assert" // Not using for this task
	// "github.com/stretchr/testify/require" // Not using for this task
)

func TestCreateUser_Success(t *testing.T) {
	repo := memory.NewInMemoryUserRepository()
	userToCreate := models.User{
		Username:     "testuser1",
		PasswordHash: "hash1",
	}

	err := repo.CreateUser(userToCreate)
	if err != nil {
		t.Fatalf("CreateUser() error = %v, wantErr %v", err, false)
	}

	// Check if ID was assigned (should be 1 if it's the first user)
	if userToCreate.ID == 0 { //InMemoryRepo assigns ID in CreateUser, but the original model is not updated by pointer
		// We need to fetch the user to check its assigned ID
		createdUser, _ := repo.GetUserByUsername("testuser1")
		if createdUser == nil {
			t.Fatalf("User was not created or cannot be fetched by username")
		}
		if createdUser.ID == 0 {
			t.Errorf("Expected user ID to be non-zero after creation, got %d", createdUser.ID)
		}
		userToCreate.ID = createdUser.ID // update for later checks if needed
	}


	// Verify by getting user by username
	retrievedUserByUsername, err := repo.GetUserByUsername(userToCreate.Username)
	if err != nil {
		t.Fatalf("GetUserByUsername() after create error = %v", err)
	}
	if retrievedUserByUsername == nil {
		t.Fatalf("Expected to retrieve user '%s' by username, but got nil", userToCreate.Username)
	}
	if retrievedUserByUsername.Username != userToCreate.Username {
		t.Errorf("Retrieved user username = %s, want %s", retrievedUserByUsername.Username, userToCreate.Username)
	}
	if retrievedUserByUsername.PasswordHash != userToCreate.PasswordHash {
		t.Errorf("Retrieved user password hash = %s, want %s", retrievedUserByUsername.PasswordHash, userToCreate.PasswordHash)
	}
	if retrievedUserByUsername.ID == 0 { // ID should be assigned
		t.Errorf("Retrieved user ID is 0, expected a non-zero ID")
	}

	// Verify by getting user by ID
	retrievedUserByID, err := repo.GetUserByID(retrievedUserByUsername.ID) // Use the ID from the retrieved user
	if err != nil {
		t.Fatalf("GetUserByID() after create error = %v", err)
	}
	if retrievedUserByID == nil {
		t.Fatalf("Expected to retrieve user by ID %d, but got nil", retrievedUserByUsername.ID)
	}
	if retrievedUserByID.Username != userToCreate.Username {
		t.Errorf("Retrieved user by ID username = %s, want %s", retrievedUserByID.Username, userToCreate.Username)
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	repo := memory.NewInMemoryUserRepository()
	user1 := models.User{Username: "duplicateuser", PasswordHash: "hash1"}
	user2 := models.User{Username: "duplicateuser", PasswordHash: "hash2"}

	err := repo.CreateUser(user1)
	if err != nil {
		t.Fatalf("CreateUser() for first user failed: %v", err)
	}

	err = repo.CreateUser(user2)
	if err == nil {
		t.Errorf("Expected error when creating user with duplicate username, but got nil")
	}
	// Here you could check for a specific error type if your CreateUser returns a custom error for duplicates
	// For example: if !errors.Is(err, memory.ErrUserAlreadyExists) { ... }
	// But based on current InMemoryUserRepository, it returns fmt.Errorf.
}

func TestGetUserByUsername_Found(t *testing.T) {
	repo := memory.NewInMemoryUserRepository()
	userToCreate := models.User{Username: "findme", PasswordHash: "hash"}
	_ = repo.CreateUser(userToCreate) // Error handled in TestCreateUser_Success

	// Fetch the user to get its assigned ID for completeness, though not strictly needed for this test
	createdUser, _ := repo.GetUserByUsername(userToCreate.Username)
	if createdUser == nil {
		t.Fatalf("Setup for TestGetUserByUsername_Found failed: user not created.")
	}

	retrievedUser, err := repo.GetUserByUsername("findme")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v, wantErr %v", err, false)
	}
	if retrievedUser == nil {
		t.Fatalf("Expected to find user 'findme', but got nil")
	}
	if retrievedUser.Username != "findme" {
		t.Errorf("GetUserByUsername() username = %s, want %s", retrievedUser.Username, "findme")
	}
	if retrievedUser.PasswordHash != "hash" {
		t.Errorf("GetUserByUsername() password hash = %s, want %s", retrievedUser.PasswordHash, "hash")
	}
	if retrievedUser.ID != createdUser.ID { // Compare with the ID assigned during creation
		t.Errorf("GetUserByUsername() ID = %d, want %d", retrievedUser.ID, createdUser.ID)
	}
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	repo := memory.NewInMemoryUserRepository()
	retrievedUser, err := repo.GetUserByUsername("nonexistentuser")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v, wantErr %v", err, false)
	}
	if retrievedUser != nil {
		t.Errorf("Expected nil when getting non-existent user by username, but got user: %+v", retrievedUser)
	}
}

func TestGetUserByID_Found(t *testing.T) {
	repo := memory.NewInMemoryUserRepository()
	userToCreate := models.User{Username: "findbyid", PasswordHash: "hash"}
	_ = repo.CreateUser(userToCreate)

	// Fetch the user to get its assigned ID
	createdUser, _ := repo.GetUserByUsername(userToCreate.Username)
	if createdUser == nil {
		t.Fatalf("Setup for TestGetUserByID_Found failed: user not created or fetched.")
	}
	if createdUser.ID == 0 {
		t.Fatalf("Setup for TestGetUserByID_Found failed: user ID is 0.")
	}

	retrievedUser, err := repo.GetUserByID(createdUser.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v, wantErr %v", err, false)
	}
	if retrievedUser == nil {
		t.Fatalf("Expected to find user by ID %d, but got nil", createdUser.ID)
	}
	if retrievedUser.Username != "findbyid" {
		t.Errorf("GetUserByID() username = %s, want %s", retrievedUser.Username, "findbyid")
	}
	if retrievedUser.ID != createdUser.ID {
		t.Errorf("GetUserByID() ID = %d, want %d", retrievedUser.ID, createdUser.ID)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	repo := memory.NewInMemoryUserRepository()
	nonExistentID := int64(999)
	retrievedUser, err := repo.GetUserByID(nonExistentID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v, wantErr %v", err, false)
	}
	if retrievedUser != nil {
		t.Errorf("Expected nil when getting non-existent user by ID %d, but got user: %+v", nonExistentID, retrievedUser)
	}
}

func TestInitUserSchema_NoOp(t *testing.T) {
	repo := memory.NewInMemoryUserRepository()
	err := repo.InitUserSchema()
	if err != nil {
		t.Errorf("InitUserSchema() for in-memory repo should be a no-op and return nil, but got error: %v", err)
	}
	// No other state to check as it's a no-op.
}
