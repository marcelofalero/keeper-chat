package ports

import (
	"context"
	// "keeper/server/models" // Old model
	usersmanagement "keeper/server/users-management" // New model
)

// AuthService defines the interface for authentication operations.
type AuthService interface {
	// Register(username, password string) (*models.User, error) // Now handled by Kratos
	// Login(username, password string) (string, error)          // Now handled by Kratos, returns session cookie

	// ValidateToken now validates a Kratos session token/cookie.
	// It takes a context and returns the new usersmanagement.User.
	ValidateToken(ctx context.Context, tokenString string) (*usersmanagement.User, error)
}
