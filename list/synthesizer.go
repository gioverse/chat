package list

import (
	"fmt"

	"gioui.org/layout"
)

// Synthesis holds the results of transforming a slice of Elements
// with a Synthesizer hook.
type Synthesis struct {
	// Elements holds the resulting elements.
	Elements []Element
	// SerialToIndex maps the serial of an element to the index it
	// occupies within the Elements slice.
	SerialToIndex map[Serial]int
	// ToSourceIndicies maps each index in Elements to the index of
	// the element that generated it when given to the Synthesizer.
	// It is always true that Elements[i] was synthesized from
	// Source[ToSourceIndicies[i]].
	ToSourceIndicies []int
	// The source elements.
	Source []Element
}

func (s Synthesis) String() string {
	return fmt.Sprintf("{Elements: %v}", s.Elements)
}

// SerialAt returns the serial at the given index within the Source slice,
// if there is one.
func (s Synthesis) SerialAt(index int) Serial {
	if index < 0 || index >= len(s.Source) {
		return NoSerial
	}
	return s.Source[index].Serial()
}

// ViewportToSerials converts the First and Count fields of the provided
// viewport into a pair of serials representing the range of elements
// visible within that viewport.
func (s Synthesis) ViewportToSerials(viewport layout.Position) (Serial, Serial) {
	if len(s.ToSourceIndicies) < 1 {
		return NoSerial, NoSerial
	}
	if viewport.First >= len(s.ToSourceIndicies) {
		viewport.First = len(s.ToSourceIndicies) - 1
	}
	startSrcIdx := s.ToSourceIndicies[viewport.First]
	startSerial := SerialAtOrBefore(s.Source, startSrcIdx)
	lastIndex := len(s.ToSourceIndicies) - 1
	vpLastIndex := max(0, viewport.First+viewport.Count-1)
	endSrcIdx := s.ToSourceIndicies[min(vpLastIndex, lastIndex)]
	endSerial := SerialAtOrAfter(s.Source, endSrcIdx)
	return startSerial, endSerial
}

// Synthesize applies a Synthesizer to a slice of elements, returning
// the resulting slice of elements as well as a mapping from the index
// of each resulting element to the input element that generated it.
func Synthesize(elements []Element, synth Synthesizer) Synthesis {
	var s Synthesis
	s.Source = elements
	for i, elem := range elements {
		var (
			previous Element
			next     Element
		)
		if i > 0 {
			previous = elements[i-1]
		} else {
			previous = Start{}
		}
		if i < len(elements)-1 {
			next = elements[i+1]
		} else {
			next = End{}
		}
		synthesized := synth(previous, elem, next)
		// Mark that each of these synthesized elements came from the
		// raw element at index i.
		for range synthesized {
			s.ToSourceIndicies = append(s.ToSourceIndicies, i)
		}
		s.Elements = append(s.Elements, synthesized...)
	}
	s.SerialToIndex = make(map[Serial]int)
	for i, e := range s.Elements {
		if e.Serial() != NoSerial {
			s.SerialToIndex[e.Serial()] = i
		}
	}
	return s
}
