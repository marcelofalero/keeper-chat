package ports

import "keeper/server/models"

// UserRepository defines the interface for user data persistence.
type UserRepository interface {
	CreateUser(user models.User) error
	GetUserByUsername(username string) (*models.User, error)
	GetUserByID(id int64) (*models.User, error)
}
