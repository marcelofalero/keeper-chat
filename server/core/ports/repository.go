package ports

import "keeper/server/models"

// MessageRepository defines the interface for message persistence.
type MessageRepository interface {
	SaveMessage(msg models.Message) (models.Message, error) // Adjusted to return the saved message
	GetMessages() ([]models.Message, error)
	GetRecentMessages(limit int) ([]models.Message, error)
}
