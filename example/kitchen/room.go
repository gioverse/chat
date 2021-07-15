package main

import (
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
}

// Room is a unique conversation context.
// Note(jfm): Allocates model and interact, not sure about that.
// Avoids the UI needing to allocate two lists (interact/model) for the
// rooms.
type Room struct {
	model.Room
	Interact appwidget.Room
	Messages *RowTracker
	List     *list.Manager
	// Editor contains the edit buffer for composing messages.
	Editor widget.Editor
}

// NewRow generates a new row in the Room's RowTracker and inserts it
// into the list manager for the room.
func (r *Room) NewRow() {
	row := r.Messages.NewRow()
	go r.List.Update([]list.Element{row}, nil)
}

// DeleteRow removes the row with the provided serial from both the
// row tracker and the list manager for the room.
func (r *Room) DeleteRow(serial list.Serial) {
	r.Messages.Delete(serial)
	go r.List.Update(nil, []list.Serial{serial})
}

// Active returns the active room, empty if not rooms are available.
func (r Rooms) Active() *Room {
	if len(r.List) == 0 {
		return &Room{}
	}
	return &r.List[r.active]
}

// Select the room at the given index.
// Index is bounded by [0, len(rooms)).
func (r *Rooms) Select(index int) {
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
	defer func() { r.changed = false }()
	return r.changed
}
