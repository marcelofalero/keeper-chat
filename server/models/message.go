package models

import "time"

// Message represents a chat message
type Message struct {
	ID        int64     `json:"id"`
	User      string    `json:"user"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}
