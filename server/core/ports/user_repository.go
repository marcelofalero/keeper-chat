package ports

import "keeper/server/models"

// UserRepository defines the interface for user data persistence.
type UserRepository interface {
	Initialize() error // Renamed from InitUserSchema
	CreateUser(user models.User) error
	GetUserByUsername(username string) (*models.User, error)
	GetUserByID(id int64) (*models.User, error)
}
