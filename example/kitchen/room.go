package main

import (
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
	Messages RowTracker
	List     *list.Manager
}

// Active returns the active room, empty if not rooms are available.
func (r Rooms) Active() Room {
	if len(r.List) == 0 {
		return Room{}
	}
	return r.List[r.active]
}

// Select the room at the given index.
// Index is bounded by [0, len(rooms)).
func (r *Rooms) Select(index int) {
	r.changed = true
	if index < 0 {
		index = 0
	}
	if index > len(r.List) {
		index = len(r.List) - 1
	}
	r.active = index
}

// Changed if the active room has changed since last call.
func (r *Rooms) Changed() bool {
	defer func() { r.changed = false }()
	return r.changed
}
