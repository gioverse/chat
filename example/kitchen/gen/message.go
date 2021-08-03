// Package gen implements data generators for the chat kitchen.
package gen

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"git.sr.ht/~gioverse/chat/example/kitchen/model"
	matchat "git.sr.ht/~gioverse/chat/widget/material"
	lorem "github.com/drhodes/golorem"
	"github.com/lucasb-eyer/go-colorful"
)

// inflection point in the theoretical message timeline.
// Messages with serial before inflection are older, and messages after it
// are newer.
const inflection = math.MaxInt64 / 2

type Generator struct {
	// FetchImage callback fetches an image of the given size.
	FetchImage func(image.Point) image.Image
	// old is the serial counter for old messages.
	old syncInt
	// new is the serial counter for new messages.
	new syncInt
}

// GenUsers generates some random number of users between min and max.
func (g *Generator) GenUsers(min, max int) *model.Users {
	return GenUsers(min, max, g.FetchImage)
}

// GenRooms generates some random number of rooms between min and max.
func (g *Generator) GenRooms(min, max int) *model.Rooms {
	return GenRooms(min, max, g.FetchImage)
}

// GenHistoryMessage generates an old message that theoretically exists at
// some point in history.
func (g *Generator) GenHistoricMessage(user *model.User) model.Message {
	var (
		serial = g.old.Increment()
		at     = time.Now().Add(time.Hour * time.Duration(serial) * -1)
	)
	return GenMessage(user, lorem.Paragraph(1, 5), inflection-serial, at, g.FetchImage)
}

// GenNewMessage generates a new message ready to be sent to the data model.
func (g *Generator) GenNewMessage(user *model.User, content string) model.Message {
	return GenMessage(user, content, inflection+g.new.Increment(), time.Now(), nil)
}

// GenMessage generates a message with sensible defaults.
func GenMessage(
	user *model.User,
	content string,
	serial int,
	at time.Time,
	fetchImage func(image.Point) image.Image,
) model.Message {
	return model.Message{
		SerialID: fmt.Sprintf("%05d", serial),
		Sender:   user.Name,
		Content:  content,
		SentAt:   at,
		Avatar:   user.Avatar,
		Image: func() image.Image {
			if fetchImage == nil {
				return nil
			}
			if rand.Float32() < 0.7 {
				return nil
			}
			sizes := []image.Point{
				image.Pt(1792, 828),
				image.Pt(828, 1792),
				image.Pt(600, 600),
				image.Pt(300, 300),
			}
			return fetchImage(sizes[rand.Intn(len(sizes))])
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
func GenUsers(min, max int, fetchImage func(image.Point) image.Image) *model.Users {
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
				if fetchImage != nil {
					return fetchImage(image.Pt(64, 64))
				}
				return nil
			}(),
			Color: func() color.NRGBA {
				return ToNRGBA(colorful.FastHappyColor().Clamped())
			}(),
		})
	}
	return &users
}

// GenRooms generates a random number of rooms between min and max.
func GenRooms(min, max int, fetchImage func(image.Point) image.Image) *model.Rooms {
	var rooms model.Rooms
	for ii := rand.Intn(max-min) + min; ii > 0; ii-- {
		rooms.Add(model.Room{
			Name: strings.Trim(lorem.Sentence(1, 5), "."),
			Image: func() image.Image {
				if fetchImage == nil {
					return nil
				}
				return fetchImage(image.Pt(64, 64))
			}(),
		})
	}
	return &rooms
}

// syncInt is a synchronized integer.
type syncInt struct {
	v int
	sync.Mutex
}

// Increment and return a copy of the underlying value.
func (si *syncInt) Increment() int {
	var v int
	si.Lock()
	si.v++
	v = si.v
	si.Unlock()
	return v
}

// ToNRGBA converts a colorful.Color to the nearest representable color.NRGBA.
func ToNRGBA(c colorful.Color) color.NRGBA {
	r, g, b, a := c.RGBA()
	return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
}
