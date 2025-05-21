package ports

import "keeper/server/models"

// AuthService defines the interface for user authentication and authorization.
type AuthService interface {
	Register(username, password string) (*models.User, error)
	VerifyUserCredentials(username, password string) (*models.User, error)
	ValidateToken(tokenString string) (*models.User, error)
}
