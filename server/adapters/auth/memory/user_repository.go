package memory

import (
	"fmt"
	"sync"

	"keeper/server/core/ports"
	"keeper/server/models"
)

// Verify InMemoryUserRepository implements ports.UserRepository
var _ ports.UserRepository = (*InMemoryUserRepository)(nil)

// InMemoryUserRepository implements the UserRepository interface using in-memory maps.
type InMemoryUserRepository struct {
	mu              sync.RWMutex
	usersByID       map[int64]models.User
	usersByUsername map[string]models.User
	nextID          int64
}

// NewInMemoryUserRepository creates a new instance of InMemoryUserRepository.
func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		usersByID:       make(map[int64]models.User),
		usersByUsername: make(map[string]models.User),
		nextID:          1,
	}
}

// CreateUser adds a new user to the in-memory store.
func (r *InMemoryUserRepository) CreateUser(user models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.usersByUsername[user.Username]; exists {
		return fmt.Errorf("user with username '%s' already exists", user.Username)
	}

	// Assign a new ID
	user.ID = r.nextID
	r.nextID++

	r.usersByID[user.ID] = user
	r.usersByUsername[user.Username] = user
	return nil
}

// GetUserByUsername retrieves a user by their username from the in-memory store.
// Returns a copy of the user to prevent direct modification of the stored map value.
func (r *InMemoryUserRepository) GetUserByUsername(username string) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.usersByUsername[username]
	if !exists {
		return nil, nil // User not found
	}
	// Return a pointer to a copy
	userCopy := user 
	return &userCopy, nil
}

// GetUserByID retrieves a user by their ID from the in-memory store.
// Returns a copy of the user.
func (r *InMemoryUserRepository) GetUserByID(id int64) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.usersByID[id]
	if !exists {
		return nil, nil // User not found
	}
	// Return a pointer to a copy
	userCopy := user
	return &userCopy, nil
}

// Initialize is a no-op for the in-memory repository but satisfies the interface.
func (r *InMemoryUserRepository) Initialize() error {
	// No schema initialization needed for in-memory store.
	return nil
}
