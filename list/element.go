package list

import (
	"fmt"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/widget/material"
)

// Serial uniquely identifies a list element.
type Serial string

// NoSerial is a special serial that can be used by Elements that do not require
// a unique identifier. Only stateless elements may go without a unique
// identifier.
const NoSerial = Serial("")

// Element is a type that can be presented by a Manager.
type Element interface {
	// Serial returns a unique identifier for the Element, if it has one.
	// In order for an Element to be stateful, it _must_ return a unique
	// Serial. Elements that are not stateful may return the special Serial
	// NoSerial to indicate that they do not need any state allocated
	// for them.
	Serial() Serial
}

// Start is a psuedo Element that indicates the beginning of the list view,
// that is, the beginning of the elements currently loaded in memory.
// Type assert inside Synthesizer to check for list boundary.
type Start struct{}

func (Start) Serial() Serial {
	return Serial("START")
}

// End is a psuedo Element that indicates the end of the list view, that is,
// the end of the elements currently loaded in memory.
// Type assert inside Synthesizer to check for list boundary.
type End struct{}

func (End) Serial() Serial {
	return Serial("END")
}

// Synthesizer is a function that can insert synthetic elements into
// a list of elements. The most common use case for this is to insert
// separators between elements indicating the passage of time or
// some other logical transition between them. previous may be nil
// if the Synthesizer is invoked at the beginning of the list. This
// function may choose to return nil to prevent current from
// being shown.
type Synthesizer func(previous, current, next Element) []Element

// Comparator returns whether element a sorts before element b in the
// list.
type Comparator func(a, b Element) bool

// Loader is a function that can fulfill load requests. If it returns
// a response with no elements in a given direction, the manager will not
// invoke the loader in that direction again until the manager loads
// data from the other end of the list or another manger state update
// occurs.
//
// Loader implements pull modifications. When the manager wants more data it
// will invoke the Loader hook to get more.
type Loader func(direction Direction, relativeTo Serial) []Element

// Presenter is a function that can transform the data for an Element
// into a widget to be laid out in the user interface. It must not return
// nil. The state parameter may be nil if the Element either has no
// Serial or if the Allocator function returned nil for the element.
type Presenter func(current Element, state interface{}) layout.Widget

// Allocator is a function that can allocate the appropriate state
// type for a given Element. It will only be invoked for Elements that
// return a serial from their Serial() method. It may return nil,
// indicating that the element in question does not need any persistent
// state.
type Allocator func(current Element) (state interface{})

// Hooks provides the lifecycle hooks necessary for a Manager
// to orchestrate the state of all its managed elements. See the documentation
// of each function type for details.
type Hooks struct {
	Synthesizer
	Comparator
	Loader
	Presenter
	Allocator
	// Invalidator triggers a new frame in the window displaying the managed
	// list.
	Invalidator func()
}

type defaultElement struct {
	serial Serial
}

func (d defaultElement) Serial() Serial {
	return d.serial
}

func newDefaultElements() (out []Element) {
	for i := 0; i < 100; i++ {
		out = append(out, defaultElement{
			serial: Serial(fmt.Sprintf("%05d", i)),
		})
	}
	return out
}

// DefaultHooks returns a Hooks instance with most fields defined as no-ops.
// It does populate the Invalidator field with w.Invalidate.
func DefaultHooks(w *app.Window, th *material.Theme) Hooks {
	return Hooks{
		Synthesizer: func(prev, curr, next Element) []Element {
			return []Element{curr}
		},
		Comparator: func(a, b Element) bool {
			return string(a.Serial()) < string(b.Serial())
		},
		Loader: func(dir Direction, relativeTo Serial) []Element {
			if relativeTo == NoSerial {
				return newDefaultElements()
			}
			return nil
		},
		Presenter: func(elem Element, state interface{}) layout.Widget {
			return material.H4(th, "Implement list.Hooks to change me.").Layout
		},
		Allocator: func(elem Element) interface{} {
			return nil
		},
		Invalidator: w.Invalidate,
	}
}

func min(ints ...int) int {
	lowest := ints[0]
	for i := 1; i < len(ints); i++ {
		if ints[i] < lowest {
			lowest = ints[i]
		}
	}
	return lowest
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Direction indicates the direction of a load request with respect to the list.
type Direction uint8

// Add combines the receiving direction with the parameter.
func (d *Direction) Add(other Direction) {
	switch *d {
	case noDirection:
		*d = other
	case After:
		if other == Before {
			*d = both
		}
	case Before:
		if other == After {
			*d = both
		}
	}
}

// Contains returns whether the receiver direction logically includes the
// provided direction.
func (d Direction) Contains(other Direction) bool {
	switch d {
	case noDirection:
		return false
	case both:
		return true
	case After:
		return other == After
	case Before:
		return other == Before
	default:
		return false
	}
}

const (
	noDirection Direction = iota
	// Before loads serial values earlier than a reference value.
	Before
	// After loads serial values after a reference value.
	After
	// both indicates that both directions are ignored relative to a
	// reference value.
	both
)

// String converts a direction into a printable representation.
func (d Direction) String() string {
	switch d {
	case noDirection:
		return "no direction"
	case Before:
		return "Before"
	case After:
		return "After"
	case both:
		return "both"
	default:
		return "unknown direction"
	}
}

// loadRequest represents a request to load more elements on one end of the list.
type loadRequest struct {
	Direction Direction
	viewport
}

// modificationRequest represents a request to insert or update some elements
// within the managed list.
type modificationRequest struct {
	NewOrUpdate []Element
	UpdateOnly  []Element
	Remove      []Serial
}

// SliceRemove takes the given index of a slice and swaps it with the final
// index in the slice, then shortens the slice by one element. This hides
// the element at index from the slice, though it does not erase its data.
func SliceRemove(s *[]Element, index int) {
	if s == nil || len(*s) < 1 || index >= len(*s) {
		return
	}
	lastIndex := len(*s) - 1
	(*s)[index], (*s)[lastIndex] = (*s)[lastIndex], (*s)[index]
	*s = (*s)[:lastIndex]
}

// SliceFilter removes elements for which the predicate returns false
// from the slice.
func SliceFilter(s *[]Element, predicate func(elem Element) bool) {
	if predicate == nil {
		return
	}
	// Avoids using a range loop because we modify the slice as we iterate.
	for i := 0; i < len(*s); i++ {
		elem := (*s)[i]
		if predicate(elem) {
			continue
		}
		// Remove this element from the new slice.
		SliceRemove(s, i)
		// Check the element at this index again next iteration.
		i--
	}
}

// MakeIndexValid forces the given index to be in bounds for given slice.
func MakeIndexValid(slice []Element, index int) int {
	if index > len(slice) {
		index = len(slice) - 1
	} else if index < 0 {
		index = 0
	}
	return index
}

// SerialAtOrBefore returns the serial of the element at the given index
// if it is not NoSerial. If it is NoSerial, this method iterates backwards
// towards the beginning of the list, searching for the nearest element with
// a serial. If no serial is found before the beginning of the list, NoSerial
// is returned.
func SerialAtOrBefore(list []Element, index int) Serial {
	for i := MakeIndexValid(list, index); i >= 0; i-- {
		if s := list[index].Serial(); s != NoSerial {
			return s
		}
	}
	return NoSerial
}

// SerialAtOrAfter returns the serial of the element at the given index
// if it is not NoSerial. If it is NoSerial, this method iterates forwards
// towards the end of the list, searching for the nearest element with
// a serial. If no serial is found before the end of the list, NoSerial
// is returned.
func SerialAtOrAfter(list []Element, index int) Serial {
	for i := MakeIndexValid(list, index); i < len(list); i++ {
		if s := list[index].Serial(); s != NoSerial {
			return s
		}
	}
	return NoSerial
}
