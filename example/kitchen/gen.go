package main

import (
	"fmt"
	"image"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"git.sr.ht/~gioverse/chat/example/kitchen/model"
	matchat "git.sr.ht/~gioverse/chat/widget/material"
	lorem "github.com/drhodes/golorem"
)

// Messager generates messages asynchronously.
type Messager struct {
	// Users participating in this room.
	// This is the pool of users to select for generating messages.
	Users *model.Users
	// old is the serial counter for old messages.
	old SyncInt
	// new is the serial counter for new messages.
	new SyncInt
}

// inflection point in the theoretical message timeline.
// Messages with serial before inflection are older, and messages after it
// are newer.
const inflection = math.MaxInt64 / 2

// Send a new message for the given user, with the given content.
// If that user does not exist, an empty message is returned.
func (m *Messager) Send(sender, content string) model.Message {
	return m.msg(sender, content, inflection+m.new.Increment(), time.Now())
}

// Generate an historic message that occured some time in the past, from a
// randomly selected user with random content.
func (m *Messager) Generate() model.Message {
	var (
		user   = m.Users.Random()
		serial = m.old.Increment()
		at     = time.Now().Add(time.Hour * time.Duration(serial) * -1)
	)
	return m.msg(user.Name, lorem.Paragraph(1, 5), inflection-serial, at)
}

// msg generates a message with sensible defaults.
func (m *Messager) msg(sender, content string, serial int, at time.Time) model.Message {
	user, ok := m.Users.Lookup(sender)
	if !ok {
		return model.Message{}
	}
	return model.Message{
		SerialID: fmt.Sprintf("%05d", serial),
		Sender:   user.Name,
		Content:  content,
		SentAt:   at,
		Avatar:   user.Avatar,
		Image: func() image.Image {
			if rand.Float32() < 0.7 {
				return nil
			}
			sizes := []image.Point{
				image.Pt(1792, 828),
				image.Pt(828, 1792),
				image.Pt(600, 600),
				image.Pt(300, 300),
			}
			img, err := randomImage(sizes[rand.Intn(len(sizes))])
			if err != nil {
				log.Print(err)
				return nil
			}
			return img
		}(),
		Read: func() bool {
			return serial < inflection
		}(),
		Status: func() string {
			if rand.Int()%10 == 0 {
				return matchat.FailedToSend
			}
			return ""
		}(),
	}
}

// GenUsers will generate a random number of fake users.
func GenUsers(min, max int) *model.Users {
	var (
		users model.Users
	)
	for ii := rand.Intn(max-min) + min; ii > 0; ii-- {
		users.Add(model.User{
			Name: lorem.Word(4, 15),
			Theme: func() model.Theme {
				if rand.Float32() > 0.7 {
					if rand.Float32() > 0.5 {
						return model.ThemePlatoCookie
					}
					return model.ThemeHotdog
				}
				return model.ThemeEmpty
			}(),
			Avatar: func() image.Image {
				img, err := randomImage(image.Pt(64, 64))
				if err != nil {
					panic(err)
				}
				return img
			}(),
		})
	}
	return &users
}

// SyncInt is a synchronized integer.
type SyncInt struct {
	v int
	sync.Mutex
}

// Increment and return a copy of the underlying value.
func (si *SyncInt) Increment() int {
	var v int
	si.Lock()
	si.v++
	v = si.v
	si.Unlock()
	return v
}
