package main

import (
	"log"
	"math/rand"
	"sort"
	"sync"
	"time"

	"git.sr.ht/~gioverse/chat/example/kitchen/gen"
	"git.sr.ht/~gioverse/chat/example/kitchen/model"
	"git.sr.ht/~gioverse/chat/list"
	lorem "github.com/drhodes/golorem"
)

// RowTracker is a stand-in for an application's data access logic.
// It stores a set of chat messages and can load them on request.
// It simulates network latency during the load operations for
// realism.
type RowTracker struct {
	// SimulateLatency specifies whether to sleep for some random duration to
	// simulate blocking on a network call.
	SimulateLatency bool
	sync.Mutex
	Rows          []list.Element
	SerialToIndex map[list.Serial]int
	Users         *model.Users
	Local         *model.User
	Generator     *gen.Generator
}

// NewExampleData constructs a RowTracker populated with the provided
// quantity of messages.
func NewExampleData(users *model.Users, local *model.User, g *gen.Generator, size int) *RowTracker {
	rt := &RowTracker{
		SerialToIndex: make(map[list.Serial]int),
		Generator:     g,
		Local:         local,
		Users:         users,
	}
	for i := 0; i < size; i++ {
		rt.Add(g.GenHistoricMessage(rt.Users.Random()))
	}
	return rt
}

// SendMessage adds the message to the data model.
// This is analogous to interacting with the backend api.
func (rt *RowTracker) Send(user, content string) model.Message {
	u, ok := rt.Users.Lookup(user)
	if !ok {
		return model.Message{}
	}
	msg := rt.Generator.GenNewMessage(u, content)
	rt.Add(msg)
	return msg
}

// Add a list element as a row of data to track.
func (rt *RowTracker) Add(r list.Element) {
	rt.Lock()
	rt.Rows = append(rt.Rows, r)
	rt.reindex()
	rt.Unlock()
}

// Latest returns the latest element, or nil.
func (r *RowTracker) Latest() list.Element {
	r.Lock()
	final := len(r.Rows) - 1
	// Unlock because index will lock again.
	r.Unlock()
	return r.Index(final)
}

// Index returns the element at the given index, or nil.
func (r *RowTracker) Index(ii int) list.Element {
	r.Lock()
	defer r.Unlock()
	if len(r.Rows) == 0 || len(r.Rows) < ii {
		return nil
	}
	if ii < 0 {
		return r.Rows[0]
	}
	return r.Rows[ii]
}

// NewRow generates a new row.
func (r *RowTracker) NewRow() list.Element {
	el := r.Generator.GenNewMessage(r.Users.Random(), lorem.Paragraph(1, 4))
	r.Add(el)
	return el
}

// Load simulates loading chat history from a database or API. It
// sleeps for a random number of milliseconds and then returns
// some messages.
func (r *RowTracker) Load(dir list.Direction, relativeTo list.Serial) (loaded []list.Element) {
	if r.SimulateLatency {
		duration := time.Millisecond * time.Duration(rand.Intn(1000))
		log.Println("sleeping", duration)
		time.Sleep(duration)
	}
	r.Lock()
	defer r.Unlock()
	defer func() {
		// Ensure the slice we return is backed by different memory than the underlying
		// RowTracker's slice, to avoid data races when the RowTracker sorts its storage.
		loaded = dupSlice(loaded)
	}()
	numRows := len(r.Rows)
	if relativeTo == list.NoSerial {
		// If loading relative to nothing, likely the chat interface is empty.
		// We should load the most recent messages first in this case, regardless
		// of the direction parameter.
		return r.Rows[numRows-min(10, numRows):]
	}
	idx := r.SerialToIndex[relativeTo]
	if dir == list.After {
		return r.Rows[idx+1 : min(numRows, idx+11)]
	}
	return r.Rows[maximum(0, idx-11):idx]
}

// Delete removes the element with the provided serial from storage.
func (r *RowTracker) Delete(serial list.Serial) {
	r.Lock()
	defer r.Unlock()
	idx := r.SerialToIndex[serial]
	sliceRemove(&r.Rows, idx)
	r.reindex()
}

func (r *RowTracker) reindex() {
	sort.Slice(r.Rows, func(i, j int) bool {
		return rowLessThan(r.Rows[i], r.Rows[j])
	})
	r.SerialToIndex = make(map[list.Serial]int)
	for i, row := range r.Rows {
		r.SerialToIndex[row.Serial()] = i
	}
}
