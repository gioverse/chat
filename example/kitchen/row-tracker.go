package main

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"sync"
	"time"

	"git.sr.ht/~gioverse/chat/example/kitchen/model"
	"git.sr.ht/~gioverse/chat/list"
	matchat "git.sr.ht/~gioverse/chat/widget/material"
)

// RowTracker is a stand-in for an application's data access logic.
// It stores a set of chat messages and can load them on request.
// It simulates network latency during the load operations for
// realism.
type RowTracker struct {
	sync.Mutex
	Rows          []list.Element
	SerialToIndex map[list.Serial]int
}

// NewExampleData constructs a RowTracker populated with the provided
// quantity of messages.
func NewExampleData(size int) *RowTracker {
	rt := &RowTracker{
		SerialToIndex: make(map[list.Serial]int),
	}
	go func() {
		for i := 0; i < size; i++ {
			rt.Lock()
			r := newRow(size - i)
			rt.Rows = append(rt.Rows, r)
			sort.Slice(rt.Rows, func(i, j int) bool {
				return rowLessThan(rt.Rows[i], rt.Rows[j])
			})
			for index, element := range rt.Rows {
				rt.SerialToIndex[element.Serial()] = index
			}
			rt.Unlock()
		}
	}()
	return rt
}

// SendMessage adds the message to the data model.
// This is analogous to interacting with the backend api.
//
// NOTE(jfm): should probably make the "this is the mock business api" more
// clear.
//
// TODO(jfm): roll client-side data into "message body", and server-side
// data can then fill out the rest of the `model.Message`.
// For example, this method needs to optionally accept images, and that
// might make the params list grow arbitrarily large, depending on the
// types of client-side data that need to be supported.
func (r *RowTracker) Send(sender, content string) model.Message {
	msg := model.Message{
		Sender:  sender,
		Content: content,
		// Backend controls content ID, thus we unconditionally override it,
		// simulating some "unique ID" algorithm.
		SerialID: fmt.Sprintf("%05d", time.Now().UnixNano()),
		// Simulate network failure.
		Status: func() string {
			if rand.Int()%10 == 0 {
				return matchat.FailedToSend
			}
			return ""
		}(),
		// Well, "we" sent it!
		Read:   true,
		Local:  true,
		SentAt: time.Now(),
	}
	r.Lock()
	r.Rows = append(r.Rows, list.Element(msg))
	sort.Slice(r.Rows, func(i, j int) bool {
		return rowLessThan(r.Rows[i], r.Rows[j])
	})
	for index, element := range r.Rows {
		r.SerialToIndex[element.Serial()] = index
	}
	r.Unlock()
	return msg
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

func (r *RowTracker) NewRow() list.Element {
	r.Lock()
	defer r.Unlock()
	index := len(r.Rows)
	element := newRow(index)
	r.Rows = append(r.Rows, element)
	r.SerialToIndex[element.Serial()] = index
	return element
}

// Load simulates loading chat history from a database or API. It
// sleeps for a random number of milliseconds and then returns
// some messages.
func (r *RowTracker) Load(dir list.Direction, relativeTo list.Serial) (loaded []list.Element) {
	duration := time.Millisecond * time.Duration(rand.Intn(1000))
	log.Println("sleeping", duration)
	time.Sleep(duration)
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
	r.SerialToIndex = make(map[list.Serial]int)
	for i, row := range r.Rows {
		r.SerialToIndex[row.Serial()] = i
	}
}
