package memory_test

import (
	"testing"
	"time"

	"keeper/server/adapters/messaging/memory" // The package being tested
	"keeper/server/models"
	// "github.com/stretchr/testify/assert" // Not using for this task
	// "github.com/stretchr/testify/require" // Not using for this task
)

func TestSaveMessage_Success(t *testing.T) {
	repo := memory.NewInMemoryMessageRepository()
	msgToSave := models.Message{
		User: "user1",
		Text: "Hello world",
		// Timestamp left zero to be set by SaveMessage
	}

	savedMsg, err := repo.SaveMessage(msgToSave)
	if err != nil {
		t.Fatalf("SaveMessage() error = %v, wantErr %v", err, false)
	}

	if savedMsg.ID <= 0 {
		t.Errorf("Expected saved message ID to be > 0, got %d", savedMsg.ID)
	}
	if savedMsg.Timestamp.IsZero() {
		t.Errorf("Expected saved message Timestamp to be non-zero, but it was zero")
	}
	if savedMsg.User != msgToSave.User {
		t.Errorf("Saved message User = %s, want %s", savedMsg.User, msgToSave.User)
	}
	if savedMsg.Text != msgToSave.Text {
		t.Errorf("Saved message Text = %s, want %s", savedMsg.Text, msgToSave.Text)
	}

	// Verify the message count in the repo is 1
	allMessages, _ := repo.GetMessages()
	if len(allMessages) != 1 {
		t.Errorf("Expected 1 message in repository, got %d", len(allMessages))
	}
	if len(allMessages) == 1 && allMessages[0].ID != savedMsg.ID {
		t.Errorf("Stored message ID = %d, want %d", allMessages[0].ID, savedMsg.ID)
	}
}

func TestSaveMessage_WithTimestamp(t *testing.T) {
	repo := memory.NewInMemoryMessageRepository()
	specificTime := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)
	msgToSave := models.Message{
		User:      "user2",
		Text:      "Specific timestamp test",
		Timestamp: specificTime,
	}

	savedMsg, err := repo.SaveMessage(msgToSave)
	if err != nil {
		t.Fatalf("SaveMessage() error = %v, wantErr %v", err, false)
	}

	if !savedMsg.Timestamp.Equal(specificTime) {
		t.Errorf("Expected saved message Timestamp to be %v, got %v", specificTime, savedMsg.Timestamp)
	}
}

func TestGetMessages_Empty(t *testing.T) {
	repo := memory.NewInMemoryMessageRepository()
	messages, err := repo.GetMessages()
	if err != nil {
		t.Fatalf("GetMessages() on empty repo error = %v, wantErr %v", err, false)
	}
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages from empty repository, got %d", len(messages))
	}
}

func TestGetMessages_WithData(t *testing.T) {
	repo := memory.NewInMemoryMessageRepository()
	msg1Time := time.Now().Add(-2 * time.Minute).UTC()
	msg2Time := time.Now().Add(-1 * time.Minute).UTC()

	msg1 := models.Message{User: "user1", Text: "First message", Timestamp: msg1Time}
	msg2 := models.Message{User: "user2", Text: "Second message", Timestamp: msg2Time}

	savedMsg1, _ := repo.SaveMessage(msg1)
	savedMsg2, _ := repo.SaveMessage(msg2)

	messages, err := repo.GetMessages()
	if err != nil {
		t.Fatalf("GetMessages() with data error = %v, wantErr %v", err, false)
	}
	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}

	// Check order (SaveMessage sorts by timestamp)
	if messages[0].ID != savedMsg1.ID || !messages[0].Timestamp.Equal(savedMsg1.Timestamp) {
		t.Errorf("Expected first message to be ID %d (Timestamp %v), got ID %d (Timestamp %v)",
			savedMsg1.ID, savedMsg1.Timestamp, messages[0].ID, messages[0].Timestamp)
	}
	if messages[1].ID != savedMsg2.ID || !messages[1].Timestamp.Equal(savedMsg2.Timestamp) {
		t.Errorf("Expected second message to be ID %d (Timestamp %v), got ID %d (Timestamp %v)",
			savedMsg2.ID, savedMsg2.Timestamp, messages[1].ID, messages[1].Timestamp)
	}
}

func TestGetRecentMessages_Empty(t *testing.T) {
	repo := memory.NewInMemoryMessageRepository()
	messages, err := repo.GetRecentMessages(5)
	if err != nil {
		t.Fatalf("GetRecentMessages() on empty repo error = %v, wantErr %v", err, false)
	}
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages from empty repository for GetRecentMessages, got %d", len(messages))
	}
}

func TestGetRecentMessages_LessThanLimit(t *testing.T) {
	repo := memory.NewInMemoryMessageRepository()
	msg1Time := time.Now().Add(-2 * time.Minute).UTC()
	msg2Time := time.Now().Add(-1 * time.Minute).UTC()
	
	savedMsg1, _ := repo.SaveMessage(models.Message{User: "user1", Text: "Msg1", Timestamp: msg1Time})
	savedMsg2, _ := repo.SaveMessage(models.Message{User: "user2", Text: "Msg2", Timestamp: msg2Time})

	messages, err := repo.GetRecentMessages(5)
	if err != nil {
		t.Fatalf("GetRecentMessages() error = %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}
	// Check order - oldest of the recents first
	if messages[0].ID != savedMsg1.ID {
		t.Errorf("Expected first recent message ID to be %d, got %d", savedMsg1.ID, messages[0].ID)
	}
	if messages[1].ID != savedMsg2.ID {
		t.Errorf("Expected second recent message ID to be %d, got %d", savedMsg2.ID, messages[1].ID)
	}
}

func TestGetRecentMessages_EqualToLimit(t *testing.T) {
	repo := memory.NewInMemoryMessageRepository()
	var expectedIDs []int64
	baseTime := time.Now().Add(-10 * time.Minute).UTC()

	for i := 0; i < 3; i++ {
		msgTime := baseTime.Add(time.Duration(i) * time.Minute)
		savedMsg, _ := repo.SaveMessage(models.Message{User: "user" + string(rune(i)), Text: "Msg", Timestamp: msgTime})
		expectedIDs = append(expectedIDs, savedMsg.ID)
	}

	messages, err := repo.GetRecentMessages(3)
	if err != nil {
		t.Fatalf("GetRecentMessages() error = %v", err)
	}
	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(messages))
	}
	for i, id := range expectedIDs {
		if messages[i].ID != id {
			t.Errorf("Expected message ID %d at index %d, got %d", id, i, messages[i].ID)
		}
	}
}

func TestGetRecentMessages_MoreThanLimit(t *testing.T) {
	repo := memory.NewInMemoryMessageRepository()
	var allSavedMsgs []models.Message
	baseTime := time.Now().Add(-10 * time.Minute).UTC()

	for i := 0; i < 5; i++ {
		msgTime := baseTime.Add(time.Duration(i) * time.Minute)
		savedMsg, _ := repo.SaveMessage(models.Message{User: "user" + string(rune(i)), Text: "Msg", Timestamp: msgTime})
		allSavedMsgs = append(allSavedMsgs, savedMsg)
	}

	limit := 3
	messages, err := repo.GetRecentMessages(limit)
	if err != nil {
		t.Fatalf("GetRecentMessages() error = %v", err)
	}
	if len(messages) != limit {
		t.Fatalf("Expected %d messages, got %d", limit, len(messages))
	}

	// Expected messages are the last 'limit' number of messages from allSavedMsgs
	expectedRecentMsgs := allSavedMsgs[len(allSavedMsgs)-limit:]

	for i := 0; i < limit; i++ {
		if messages[i].ID != expectedRecentMsgs[i].ID {
			t.Errorf("Expected message ID %d at index %d, got %d", expectedRecentMsgs[i].ID, i, messages[i].ID)
		}
		if !messages[i].Timestamp.Equal(expectedRecentMsgs[i].Timestamp) {
			t.Errorf("Expected message Timestamp %v at index %d, got %v", expectedRecentMsgs[i].Timestamp, i, messages[i].Timestamp)
		}
	}
}

func TestGetRecentMessages_LimitZeroOrNegative(t *testing.T) {
	repo := memory.NewInMemoryMessageRepository()
	_, _ = repo.SaveMessage(models.Message{User: "user1", Text: "Msg1"})

	messagesZero, err := repo.GetRecentMessages(0)
	if err != nil {
		t.Fatalf("GetRecentMessages(0) error = %v", err)
	}
	if len(messagesZero) != 0 {
		t.Errorf("Expected 0 messages for limit 0, got %d", len(messagesZero))
	}

	messagesNegative, err := repo.GetRecentMessages(-1)
	if err != nil {
		t.Fatalf("GetRecentMessages(-1) error = %v", err)
	}
	if len(messagesNegative) != 0 {
		t.Errorf("Expected 0 messages for limit -1, got %d", len(messagesNegative))
	}
}

func TestInitSchema_NoOp(t *testing.T) {
	repo := memory.NewInMemoryMessageRepository()
	err := repo.InitSchema()
	if err != nil {
		t.Errorf("InitSchema() for in-memory repo should be a no-op and return nil, but got error: %v", err)
	}
}
