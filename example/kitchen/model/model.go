/*
Package model provides the domain-specific data models for this list.
*/
package model

import (
	"image"
	"time"

	"git.sr.ht/~gioverse/chat/list"
)

// Message represents a chat message.
type Message struct {
	SerialID                string
	Sender, Content, Status string
	SentAt                  time.Time
	Local                   bool
	Theme                   string
	Image                   image.Image
	Avatar                  image.Image
	Read                    bool
}

// Serial returns the unique identifier for this message.
func (m Message) Serial() list.Serial {
	return list.Serial(m.SerialID)
}

// DateBoundary represents a change in the date during a chat.
type DateBoundary struct {
	Date time.Time
}

// Serial returns the unique identifier of the message.
func (d DateBoundary) Serial() list.Serial {
	return list.NoSerial
}

// UnreadBoundary represents the boundary between the last read message
// in a chat and the next unread message.
type UnreadBoundary struct{}

// Serial returns the unique identifier for the boundary.
func (u UnreadBoundary) Serial() list.Serial {
	return list.NoSerial
}
