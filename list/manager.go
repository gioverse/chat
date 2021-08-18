package list

import (
	"fmt"
	"math"

	"gioui.org/layout"
)

// Manager presents heterogenous Element data. Each element could represent
// any element of an interface with a list.
//
// State is updated with two strategies, push and pull:
//
// Pull updates occur when the list has scrolled to the end of it's current data
// and needs to ask for more. In this case, the Loader hook will be invoked
// concurrently to get the data, if any.
//
// Push updates occur when the data source changes outside of the list. The
// application can push those changes into the list with a call to `Modify`.
//
// Any changes that fall outside the bounds of the data will be ignored and
// expected to be Loaded appropriately when scrolled into view.
type Manager struct {
	// Prefetch specifies a minimum threshold at which to start prefetching more
	// data (in either direction), as a percentage in the range [0,1] of the
	// total number of elements present in the list.
	//
	// In other words, a prefetch of 0.15 ensures load will be invoked if the
	// viewport is laying out the first or final 15% of elements.
	//
	// Defaults to '0.15' (15%), clamped to '1.0' (100%).
	Prefetch float32

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
	requests chan<- interface{}

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

	return rm
}

// DefaultPrefetch is the default prefetching threshold.
const DefaultPrefetch = 0.15

// Modify is a thread-safe means of atomically pushing modifications to the Manager:
// inserting elements into, updating elements within, or removing elements from
// the managed list state.
//
// Elements in the newOrUpdated parameter will be inserted into the managed state,
// and any pre-existing element with the same serial will be removed.
// Elements in the updateOnly parameter will replace any elements in the
// managed list with the same serial, but otherwise will not be inserted.
// Elements with a serial in the remove parameter will be removed from
// the managed list.
//
// Elements that sort outside of the view will be ignored. In that case the
// loader hook should load it when scrolled into view.
//
// This method may block, and should not be called from the goroutine that
// is performing layout.
//
// Use this method to push modifications from the data source.
//
// For "pull" modifications, see the Loader hook.
func (m *Manager) Modify(newOrUpdated []Element, updateOnly []Element, remove []Serial) {
	m.requests <- modificationRequest{
		NewOrUpdate: newOrUpdated,
		UpdateOnly:  updateOnly,
		Remove:      remove,
	}
}

// Update atomically modifies the Manager to insert or update from the provided
// elements.
//
// Elements provided that exist in the Manager will be updated in-place, and those
// that do not will be inserted as new elements.
func (m *Manager) Update(newOrUpdated []Element) {
	m.requests <- modificationRequest{
		NewOrUpdate: newOrUpdated,
		UpdateOnly:  nil,
		Remove:      nil,
	}
}

// InPlace atomically modifies the Manager to update from the provided elements.
//
// Elements provided that exist in the Manager will be updated in-place, and those
// that do not  will be ignored.
func (m *Manager) InPlace(updateOnly []Element) {
	m.requests <- modificationRequest{
		NewOrUpdate: nil,
		UpdateOnly:  updateOnly,
		Remove:      nil,
	}
}

// Remove atomically modifies the Manager to remove elements based on a Serial.
//
// Elements in the Manager that are specified in the remove list will be deleted.
// Serials that map to non-existant elements will be ignored.
func (m *Manager) Remove(remove []Serial) {
	m.requests <- modificationRequest{
		NewOrUpdate: nil,
		UpdateOnly:  nil,
		Remove:      remove,
	}
}

// Layout the element at the given index.
func (m *Manager) Layout(gtx layout.Context, index int) layout.Dimensions {
	if index < 0 {
		index = 0
	}
	if m.Prefetch <= 0.0 {
		m.Prefetch = DefaultPrefetch
	}
	if m.Prefetch > 1.0 {
		m.Prefetch = 1.0
	}
	// indexf is the precentage of the total list of elements that
	// the index represents.
	indexf := float32(index) / float32(max(len(m.elements), 1))
	// If the beginning of the list is visible, try to load prior history.
	if indexf < m.Prefetch && len(m.elements) > 0 {
		m.tryRequest(Before)
	}
	// If the end of the list is visible, try to load history afterwards.
	if indexf > 1.0-m.Prefetch && len(m.elements) > 0 {
		m.tryRequest(After)
	}
	// If there are too few elements such that the prefetch zone is never entered,
	// try to load history afterwards.
	//
	// For example, if prefetch is 0.15, indexf needs to be > 0.75 to trigger a
	// load. If there are only 2 elements present, indexf will not exceed 0.50,
	// which means the load request gets ignored despite the end of the list
	// being visible.
	//
	// The minium number of elements required to overcome this check is equal to
	// the granularity of the prefetch. Thus with a prefetch of 0.15, the list
	// needs to contain at least 7 elements to ignore this load request.
	if fewElements := len(m.elements) < int(math.Ceil(float64(1.0/m.Prefetch))); fewElements {
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
			newStartIndex, ok := su.SerialToIndex[startSerial]
			if !ok {
				// The element that was previously at the top of the viewport
				// is no longer within the list. Walk backwards towards the
				// beginning of the list, searching for an element that is
				// both in the old state list and in the updated one.
				// If this fails to find a matching element, just set the
				// viewport to start on the first element.
				for ii := listStart - 1; (startSerial == NoSerial || !ok) && ii >= 0; ii-- {
					startSerial = m.elements[ii].Serial()
					newStartIndex, ok = su.SerialToIndex[startSerial]
				}
			}
			list.Position.First = newStartIndex
			if !su.PreserveListEnd {
				// Ensure that the list considers the possibility that new content
				// has changed the end of the list.
				list.Position.BeforeEnd = true
			}
		}
		m.elements = su.Elements
		// Delete the persistent widget state for any compacted element.
		for _, serial := range su.CompactedSerials {
			delete(m.elementState, serial)
		}
	default:
	}
	if len(m.elements) == 0 {
		// Push an initial request to populate the first few messages.
		m.tryRequest(After)
	}

	// Update the cached copy of the list position to the latest value.
	m.viewportCache = list.Position

	return len(m.elements)
}

// ManagedElements returns the slice of elements managed by the manager
// during the current frame. This MUST be called from the layout goroutine,
// and callers must not insert, remove, or reorder elements.
//
// This method is useful for checking the relative positions of managed
// elements during layout. Many applications will never need this functionality.
func (m *Manager) ManagedElements(gtx layout.Context) []Element {
	return m.elements
}

// ManagedState returns the map of widget state managed by the manager
// during the current frame. This MUST be called from the layout goroutine,
// and callers must not insert or remove elements from the returned map.
//
// This method is useful for checking for events on all managed widgets in
// a single loop ahead of laying each element out, rather than checking
// each element during layout. Many applications will never need this
// functionality.
func (m *Manager) ManagedState(gtx layout.Context) map[Serial]interface{} {
	return m.elementState
}
