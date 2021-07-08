package list

import (
	"sort"

	"gioui.org/layout"
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

// Hooks provides the lifecycle hooks necessary for a ListManager
// to orchestrate the state of all its managed elements. See the documentation
// of each function type for details.
type Hooks struct {
	Synthesizer
	Comparator
	// Invalidator triggers a new frame in the window displaying the managed
	// list.
	Invalidator func()
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

// Processed returns a slice of elements that are ready for display. This
// method appends newElems, sorts the slice, uses a Synthesizer hook to
// generate any synthetic elements, and then returns the result.
func (r *processor) Process(newElems ...Element) []Element {
	r.Raw = append(r.Raw, newElems...)
	// Truncate the processed element slice.
	processed := make([]Element, 0, len(r.Raw))
	r.ProcessedToRaw = r.ProcessedToRaw[:0]
	// Re-sort elements.
	sort.Slice(r.Raw, func(i, j int) bool {
		return r.Comparator(r.Raw[i], r.Raw[j])
	})
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
