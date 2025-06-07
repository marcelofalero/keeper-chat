package ports

import "keeper/server/models"

// MessageRepository defines the interface for message persistence.
type MessageRepository interface {
	SaveMessage(msg models.Message) error
	GetMessages() ([]models.Message, error)
}
