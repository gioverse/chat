/*
Package model provides the domain-specific data models for this chat.
*/
package model

import (
	"time"

	"git.sr.ht/~gioverse/chat"
)

// Message represents a chat message.
type Message struct {
	Serial                  string
	Sender, Content, Status string
	SentAt                  time.Time
	Local                   bool
	Theme                   string
}

// ID returns the unique identifier for this message.
func (m Message) ID() chat.RowID {
	return chat.RowID(m.Serial)
}

// DateBoundary represents a change in the date during a chat.
type DateBoundary struct {
	Date time.Time
}

// ID returns the unique ID of the message.
func (d DateBoundary) ID() chat.RowID {
	return chat.NoID
}

// UnreadBoundary represents the boundary between the last read message
// in a chat and the next unread message.
type UnreadBoundary struct{}

// ID returns the unique identifier for the boundary.
func (u UnreadBoundary) ID() chat.RowID {
	return chat.NoID
}
