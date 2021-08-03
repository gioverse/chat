package main

import (
	"math/rand"
	"sync"

	"gioui.org/widget"
	"git.sr.ht/~gioverse/chat/example/kitchen/appwidget"
	"git.sr.ht/~gioverse/chat/example/kitchen/model"
	"git.sr.ht/~gioverse/chat/list"
)

// Rooms contains a selectable list of rooms.
type Rooms struct {
	active  int
	changed bool
	List    []Room
	sync.Mutex
}

// Room is a unique conversation context.
// Note(jfm): Allocates model and interact, not sure about that.
// Avoids the UI needing to allocate two lists (interact/model) for the
// rooms.
type Room struct {
	// Room model defines the backend data describing a room.
	*model.Room
	// Interact defines the interactive state for a room widget.
	Interact appwidget.Room
	// Messages implements what would be a backend data model.
	// This would be the facade to your business api.
	// This is the source of truth.
	// This type gets asked to create messages and queried for message history.
	Messages *RowTracker
	// ListState dynamically manages list state.
	// This lets us surf across a vast ocean of infinite messages, only ever
	// rendering what is actualy viewable.
	// The widget.List consumes this during layout.
	ListState *list.Manager
	// List implements the raw scrolling, adding scrollbars and responding
	// to mousewheel / touch fling gestures.
	List widget.List
	// Editor contains the edit buffer for composing messages.
	Editor widget.Editor
}

// SendMessage attempts to send the contents of the edit buffer as a
// to the model.
func (r *Room) SendMessage() {
	defer r.Editor.SetText("")
	// NOTE(jfm): the communication between backend and frontend is simulated
	// here, but not in a realistic way.
	// 1. sender and content are client-side data
	// 2. the data is passed to the backend: "hey I would like to send this message"
	// 3. the "finalized" message is returned by the backend, and then the pushed
	// in to the list, via it's Update method.
	row := r.Messages.Send(r.Editor.Text())
	r.Room.Latest = &row
	go func() {
		r.ListState.Modify([]list.Element{row}, nil, nil)
	}()
}

// NewRow generates a new row in the Room's RowTracker and inserts it
// into the list manager for the room.
func (r *Room) NewRow() {
	row := r.Messages.NewRow()
	go r.ListState.Modify([]list.Element{row}, nil, nil)
}

// DeleteRow removes the row with the provided serial from both the
// row tracker and the list manager for the room.
func (r *Room) DeleteRow(serial list.Serial) {
	r.Messages.Delete(serial)
	go r.ListState.Modify(nil, nil, []list.Serial{serial})
}

// Active returns the active room, empty if not rooms are available.
func (r *Rooms) Active() *Room {
	r.Lock()
	defer r.Unlock()
	if len(r.List) == 0 {
		return &Room{}
	}
	return &r.List[r.active]
}

// Latest returns a copy of the latest message for the room.
func (r *Room) Latest() model.Message {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.Room.Latest == nil {
		return model.Message{}
	}
	return *r.Room.Latest
}

// Select the room at the given index.
// Index is bounded by [0, len(rooms)).
func (r *Rooms) Select(index int) {
	r.Lock()
	defer r.Unlock()
	if index < 0 {
		index = 0
	}
	if index > len(r.List) {
		index = len(r.List) - 1
	}
	r.changed = true
	r.List[r.active].Interact.Active = false
	r.active = index
	r.List[r.active].Interact.Active = true
}

// Changed if the active room has changed since last call.
func (r *Rooms) Changed() bool {
	r.Lock()
	defer r.Unlock()
	defer func() { r.changed = false }()
	return r.changed
}

// Index returns a pointer to a Room at the given index.
// Index is bounded by [0, len(rooms)).
func (r *Rooms) Index(index int) *Room {
	r.Lock()
	defer r.Unlock()
	if index < 0 {
		index = 0
	}
	if index > len(r.List) {
		index = len(r.List) - 1
	}
	return &r.List[index]
}

// Index returns a pointer to a random Room in the list.
func (r *Rooms) Random() *Room {
	r.Lock()
	defer r.Unlock()
	return &r.List[rand.Intn(len(r.List)-1)]
}
