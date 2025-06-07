package ports

import "keeper/server/models"

// AuthService defines the interface for user authentication and authorization.
type AuthService interface {
	Register(username, password string) (*models.User, error)
	Login(username, password string) (tokenString string, err error)
	ValidateToken(tokenString string) (*models.User, error)
}
