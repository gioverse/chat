/*
Package model provides the domain-specific data models for this list.
*/
package model

import (
	"image"
	"image/color"
	"math/rand"
	"sync"
	"time"

	"git.sr.ht/~gioverse/chat/list"
)

// Message represents a chat message.
type Message struct {
	SerialID                string
	Sender, Content, Status string
	SentAt                  time.Time
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

// Room is a unique conversation context.
// Room can have any number of participants, and any number of messages.
// Any participant of a room should be able to view the room, send messages to
// and recieve messages from the other participants.
type Room struct {
	// Image avatar for the room.
	Image image.Image
	// Name of the room.
	Name string
	// Latest message in the room, if any.
	Latest *Message
}

// User is a unique identity that can send messages and participate in rooms.
type User struct {
	// Name of user.
	Name string
	// Avatar is the image of the user.
	Avatar image.Image
	// Theme specifies the name of a 9patch theme to use for messages from this
	// user. If theme is specified it will be the preferred message surface.
	// Empty string indicates no theme.
	Theme Theme
	// Color to use for message bubbles of messages from this user.
	Color color.NRGBA
}

// Users structure manages a collection of user data.
type Users struct {
	list  []User
	index map[string]*User
	once  sync.Once
}

// Add user to collection.
func (us *Users) Add(u User) {
	us.once.Do(func() {
		us.index = map[string]*User{}
	})
	us.list = append(us.list, u)
	us.index[u.Name] = &us.list[len(us.list)-1]
}

// List returns an ordered list of user data.
func (us *Users) List() (list []*User) {
	list = make([]*User, len(us.list))
	for ii := range us.list {
		list[ii] = &us.list[ii]
	}
	return list
}

// Lookup user by name.
func (us *Users) Lookup(name string) (*User, bool) {
	v, ok := us.index[name]
	return v, ok
}

// Random returns a randomly selected user from the collection.
// If there are no users, nil is returned.
func (us *Users) Random() *User {
	if len(us.list) == 0 {
		return nil
	}
	return &us.list[rand.Intn(len(us.list)-1)]
}

// Theme enumerates the various 9patch themes.
type Theme int

const (
	ThemeEmpty Theme = iota
	ThemePlatoCookie
	ThemeHotdog
)
