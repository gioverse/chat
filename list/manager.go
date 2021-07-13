package list

import (
	"fmt"

	"gioui.org/layout"
)

// Manager presents heterogenous Element data. Each element could represent
// any element of an interface with a list.
type Manager struct {
	// elements is the list of data to present.
	elements []Element

	// viewportCache holds the last known viewport position of the managed list.
	viewportCache layout.Position

	// presenter is a function that can transform a single Element into
	// a presentable widget.
	presenter Presenter

	// allocator is a function that can instantiate the state for a particular
	// Element.
	allocator Allocator

	// elementState is a map storing the state for the elements managed
	// by the manager.
	elementState map[Serial]interface{}

	// requests is a blocking channel of LoadRequests. Requests sent on this
	// channel will be picked up by the state management goroutine, and
	// the results will be available as data on the stateUpdates channel.
	requests chan<- loadRequest

	// stateUpdates is a buffered channel that receives changes in the managed
	// elements from the state management goroutine.
	stateUpdates <-chan stateUpdate
}

// tryRequest will send the loadRequest if and only if the background processing
// goroutine is immediately able to start working on it. Otherwise it will
// discard the request.
func (m *Manager) tryRequest(dir Direction) {
	select {
	case m.requests <- loadRequest{
		Direction: dir,
		viewport:  m.viewportCache,
	}:
	default:
	}
	return
}

// NewManager constructs a manager. maxSize defines the number of raw elements
// that the list will manage simultaneously. If the list grows beyond this, it
// will automatically discard some elements to stay beneath this limit. The
// provided hooks implement application-specific logic necessary for the
// Manager to do its job. This constructor will panic if any hooks are
// not defined.
func NewManager(maxSize int, hooks Hooks) *Manager {
	switch {
	case hooks.Allocator == nil:
		panic(fmt.Errorf("must provide an implementation of Allocator"))
	case hooks.Presenter == nil:
		panic(fmt.Errorf("must provide an implementation of Presenter"))
	case hooks.Comparator == nil:
		panic(fmt.Errorf("must provide an implementation of Comparator"))
	case hooks.Synthesizer == nil:
		panic(fmt.Errorf("must provide an implementation of Synthesizer"))
	case hooks.Loader == nil:
		panic(fmt.Errorf("must provide an implementation of Loader"))
	case hooks.Invalidator == nil:
		panic(fmt.Errorf("must provide an implementation of Invalidator"))
	}
	rm := &Manager{
		presenter:    hooks.Presenter,
		allocator:    hooks.Allocator,
		elementState: make(map[Serial]interface{}),
	}

	rm.requests, rm.stateUpdates = asyncProcess(maxSize, hooks)

	// Push an initial request to populate the first few messages.
	rm.requests <- loadRequest{Direction: After}

	return rm
}

// Layout the element at the given index.
func (m *Manager) Layout(gtx layout.Context, index int) layout.Dimensions {
	// If the beginning of the list is visible, try to load prior history.
	if index == 0 && len(m.elements) > 0 {
		m.tryRequest(Before)
	}
	// If the end of the list is visible, try to load history afterwards.
	if index == len(m.elements)-1 && len(m.elements) > 0 {
		m.tryRequest(After)
	}
	// Lay out the element for the current index.
	data := m.elements[index]
	id := data.Serial()
	state, ok := m.elementState[id]
	if !ok && id != NoSerial {
		state = m.allocator(data)
		m.elementState[id] = state
	}
	widget := m.presenter(data, state)
	return widget(gtx)
}

// UpdatedLen returns the number of elements managed by this manager, and also updates
// the state of the ListManager and List prior to layout. This method should
// be called to provide a layout.List with the length of the underlying list,
// and the layout.List should be passed in as a parameter.
func (m *Manager) UpdatedLen(list *layout.List) int {
	// Update the state of the manager in response to any loads.
	select {
	case su := <-m.stateUpdates:
		if len(m.elements) > 0 {
			listStart := min(list.Position.First, len(m.elements)-1)
			startSerial := m.elements[listStart].Serial()
			newStartIndex := su.SerialToIndex[startSerial]
			list.Position.First = newStartIndex
			// Ensure that the list considers the possibility that new content
			// has changed the end of the list.
			list.Position.BeforeEnd = true
		}
		m.elements = su.Elements
		// Delete the persistent widget state for any compacted element.
		for _, serial := range su.CompactedSerials {
			delete(m.elementState, serial)
		}
	default:
	}

	// Update the cached copy of the list position to the latest value.
	m.viewportCache = list.Position

	return len(m.elements)
}
