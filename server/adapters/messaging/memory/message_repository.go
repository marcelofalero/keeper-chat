package memory

import (
	"sort"
	"sync"
	"time"

	"keeper/server/core/ports"
	"keeper/server/models"
)

// Verify InMemoryMessageRepository implements ports.MessageRepository
var _ ports.MessageRepository = (*InMemoryMessageRepository)(nil)

// InMemoryMessageRepository implements the MessageRepository interface using in-memory slice.
type InMemoryMessageRepository struct {
	mu       sync.RWMutex
	messages []models.Message
	nextID   int64
}

// NewInMemoryMessageRepository creates a new instance of InMemoryMessageRepository.
func NewInMemoryMessageRepository() *InMemoryMessageRepository {
	return &InMemoryMessageRepository{
		messages: make([]models.Message, 0),
		nextID:   1,
	}
}

// SaveMessage adds a new message to the in-memory store.
// It assigns an ID and a timestamp if the message doesn't have one.
// It returns a copy of the saved message.
func (r *InMemoryMessageRepository) SaveMessage(msg models.Message) (models.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Assign a new ID
	msg.ID = r.nextID
	r.nextID++

	// Set timestamp if not already set
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now().UTC()
	}

	// Append a copy of the message to the slice
	// This is important to avoid external modifications to the stored message.
	messageCopy := msg 
	r.messages = append(r.messages, messageCopy)
	
	// Sort messages by timestamp after adding, to maintain order for GetRecentMessages
	sort.Slice(r.messages, func(i, j int) bool {
		return r.messages[i].Timestamp.Before(r.messages[j].Timestamp)
	})

	return messageCopy, nil
}

// GetMessages retrieves all messages from the in-memory store.
// Returns copies of the messages.
func (r *InMemoryMessageRepository) GetMessages() ([]models.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy of the slice to prevent external modification
	msgsCopy := make([]models.Message, len(r.messages))
	copy(msgsCopy, r.messages)
	return msgsCopy, nil
}

// GetRecentMessages retrieves a specified number of recent messages.
// Messages are returned in chronological order (oldest of the recent ones first).
func (r *InMemoryMessageRepository) GetRecentMessages(limit int) ([]models.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		return []models.Message{}, nil
	}

	numMessages := len(r.messages)
	if numMessages == 0 {
		return []models.Message{}, nil
	}

	start := numMessages - limit
	if start < 0 {
		start = 0
	}
	
	// The slice r.messages is already sorted chronologically by SaveMessage.
	// We need to return a copy of the relevant sub-slice.
	recentSlice := r.messages[start:numMessages]
	
	msgsCopy := make([]models.Message, len(recentSlice))
	copy(msgsCopy, recentSlice)
	
	return msgsCopy, nil
}

// Initialize is a no-op for the in-memory repository but satisfies the interface.
func (r *InMemoryMessageRepository) Initialize() error {
	// No schema initialization needed for in-memory store.
	return nil
}
