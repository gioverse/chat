package list

import (
	"fmt"
	"sort"

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

// Synthesizer is a function that can insert synthetic elements into
// a list of elements. The most common use case for this is to insert
// separators between elements indicating the passage of time or
// some other logical transition between them. previous may be nil
// if the Synthesizer is invoked at the beginning of the list. This
// function may choose to return nil to prevent current from
// being shown.
type Synthesizer func(previous, current Element) []Element

// Comparator returns whether element a sorts before element b in the
// list.
type Comparator func(a, b Element) bool

// Loader is a function that can fulfill load requests. If it returns
// a response with no elements in a given direction, the manager will not
// invoke the loader in that direction again until the manager loads
// data from the other end of the list or another manger state update
// occurs.
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
		Synthesizer: func(prev, curr Element) []Element {
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

const (
	noDirection Direction = iota
	// Before loads serial values earlier than a reference value.
	Before
	// After loads serial values after a reference value.
	After
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
	default:
		return "unknown direction"
	}
}

// loadRequest represents a request to load more elements on one end of the list.
type loadRequest struct {
	Direction Direction
	viewport  layout.Position
}

// modificationRequest represents a request to insert or update some elements
// within the managed list.
type modificationRequest struct {
	NewOrUpdate []Element
	UpdateOnly  []Element
	Remove      []Serial
}

// processor transforms a list of unsorted elements into sorted,
// presentable elements using the Comparator and Synthesizer it is
// provided.
type processor struct {
	Raw []Element
	// ProcessedToRaw tracks the raw element responsible for generating
	// eached processed element. For any given index into Process'
	// last return value, Raw[ProcessedToRaw[index]] is the corresponding
	// raw element.
	ProcessedToRaw []int
	Synthesizer
	Comparator
}

// newProcessor constructs a processor using the provided hook functions.
func newProcessor(synth Synthesizer, compare Comparator) *processor {
	return &processor{
		Synthesizer: synth,
		Comparator:  compare,
	}
}

// sliceRemove takes the given index of a slice and swaps it with the final
// index in the slice, then shortens the slice by one element. This hides
// the element at index from the slice, though it does not erase its data.
func sliceRemove(s *[]Element, index int) {
	lastIndex := len(*s) - 1
	(*s)[index], (*s)[lastIndex] = (*s)[lastIndex], (*s)[index]
	*s = (*s)[:lastIndex]
}

// Update updates the internal Raw element slice with new elements.
// This method updates the value of any existing element in Raw with
// the new value for that serial provided in newElems, appends
// all totally new elements to the end of Raw, and sorts Raw.
func (r *processor) Update(newElems []Element, updateOnly []Element, removed []Serial) {
	serialToRaw := make(map[Serial]int)
	for i, elem := range r.Raw {
		serialToRaw[elem.Serial()] = i
	}
	// Search newElems for elements that already exist within the Raw slice.
	// Avoids using a range loop because we modify the slice as we iterate.
	for i := 0; i < len(newElems); i++ {
		elem := newElems[i]
		rawIndex, exists := serialToRaw[elem.Serial()]
		if !exists {
			continue
		}
		// Update the stored existing element.
		r.Raw[rawIndex] = elem
		// Remove this element from the new slice.
		sliceRemove(&newElems, i)
		// Check the element at this index again next iteration.
		i--
	}

	// Update elements if and only if they are present.
	for _, elem := range updateOnly {
		index, isPresent := serialToRaw[elem.Serial()]
		if !isPresent {
			continue
		}
		r.Raw[index] = elem
	}

	// Find the index of each element needing removal.
	var targetIndicies []int
	for _, serial := range removed {
		idx, ok := serialToRaw[serial]
		if !ok {
			continue
		}
		targetIndicies = append(targetIndicies, idx)
	}
	// Remove them by swapping and re-slicing, starting from the highest
	// index to ensure that we do not move a removed element into the
	// middle of the list as part of the swap.
	sort.Sort(sort.Reverse(sort.IntSlice(targetIndicies)))
	for _, target := range targetIndicies {
		sliceRemove(&r.Raw, target)
	}

	r.Raw = append(r.Raw, newElems...)
	// Re-sort elements.
	sort.Slice(r.Raw, func(i, j int) bool {
		return r.Comparator(r.Raw[i], r.Raw[j])
	})
}

// Synthesize returns a slice of elements that are ready for display. This
// method uses the Synthesizer hook to generate any synthetic elements
// and returns the result.
func (r *processor) Synthesize() []Element {
	// Truncate the processed element slice.
	processed := make([]Element, 0, len(r.Raw))
	r.ProcessedToRaw = r.ProcessedToRaw[:0]
	// Synthesize any elements that need to be created between existing
	// ones.
	for i, elem := range r.Raw {
		var previous Element
		if i > 0 {
			previous = r.Raw[i-1]
		}
		synthesized := r.Synthesizer(previous, elem)
		// Mark that each of these synthesized elements came from the
		// raw element at index i.
		for range synthesized {
			r.ProcessedToRaw = append(r.ProcessedToRaw, i)
		}
		processed = append(processed, synthesized...)
	}
	return processed
}

// Compact attempts to deallocate list elements that are not in view.
// maxElem is the maximum number of elements allowed in the Raw slice
// after compaction. viewport is the current scrolling viewport of the
// list. If the provided maxElem is zero, 3*viewport.Count will be used.
// The serials of all elements removed from the Raw element slice are
// returned.
func (r *processor) Compact(maxElem int, viewport layout.Position) []Serial {
	if len(r.Raw) < 1 {
		return nil
	}
	var compactedSerials []Serial
	if maxElem == 0 {
		maxElem = viewport.Count * 3
	}
	// Resolve the viewport within the raw element slice.
	maxProcessedIndex := len(r.ProcessedToRaw) - 1
	keepStart := r.ProcessedToRaw[min(viewport.First, maxProcessedIndex)]
	initialEndOffset := min(viewport.Count, maxElem, maxProcessedIndex-keepStart)
	keepEnd := r.ProcessedToRaw[keepStart+initialEndOffset]

	additional := maxElem - (keepEnd - keepStart)
	if additional > 0 {
		// cut the additional size in half, ensuring that no element is
		// lost to integer truncation.
		half := additional / 2
		secondHalf := additional - half
		if keepStart < half {
			// Donate any unused quota at the beginning of the list to
			// the end.
			secondHalf += (half - keepStart)
		}
		if newEnd := keepEnd + secondHalf; newEnd > len(r.Raw) {
			// Donate any unused quota at the end of the list to
			// the beginning.
			half += newEnd - len(r.Raw)
		}
		keepStart = max(keepStart-half, 0)
		keepEnd = min(keepEnd+secondHalf, len(r.Raw))
	}

	// Collect the serials of elements that are being deallocated by compaction.
	for i := 0; i < keepStart; i++ {
		compactedSerials = append(compactedSerials, r.Raw[i].Serial())
	}
	for i := keepEnd; i < len(r.Raw); i++ {
		compactedSerials = append(compactedSerials, r.Raw[i].Serial())
	}

	// Allocate a new Raw slice to house the data, allowing the older,
	// longer slice to be garbage collected.
	newLength := keepEnd - keepStart
	newRaw := make([]Element, newLength)
	copy(newRaw, r.Raw[keepStart:keepEnd])
	r.Raw = newRaw

	return compactedSerials
}

// SerialForProcessedIndex returns the serial identifier for the element
// at the given processed index (or the element responsible for synthesizing
// it).
func (p *processor) SerialForProcessedIndex(index int) Serial {
	if index < 0 || index >= len(p.ProcessedToRaw) {
		return NoSerial
	}
	return p.Raw[p.ProcessedToRaw[index]].Serial()
}
