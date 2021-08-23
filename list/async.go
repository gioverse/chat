package list

import "gioui.org/layout"

// stateUpdate contains a new slice of element data and a mapping from all of
// the element serials to their respective indicies. This data structure is designed
// to allow the UI code to quickly find and update any offsets and locations
// within the new data.
type stateUpdate struct {
	Elements      []Element
	SerialToIndex map[Serial]int
	// CompactedSerials is a slice of Serials that were compacted within this update.
	CompactedSerials []Serial
	// PreserveListEnd indicates whether or not the list.Position.BeforeEnd field
	// should be reset when applying this state update.
	PreserveListEnd bool
}

// populateWith sets s.Elements to the provided slice of elements
// and populates the SerialToIndex map.
func (s *stateUpdate) populateWith(elems []Element) {
	s.Elements = elems
	s.SerialToIndex = make(map[Serial]int)
	for index, elem := range s.Elements {
		if elem.Serial() != NoSerial {
			s.SerialToIndex[elem.Serial()] = index
		}
	}
}

// asyncProcess runs a list.processor concurrently.
// New elements are processed and compacted according to maxSize
// on each loadRequest. Close the loadRequest channel to terminate
// processing.
func asyncProcess(maxSize int, hooks Hooks) (chan<- interface{}, <-chan stateUpdate) {
	compact := NewCompact(maxSize, hooks.Comparator)
	var synthesis Synthesis
	reqChan := make(chan interface{})
	updateChan := make(chan stateUpdate, 1)
	go func() {
		defer close(updateChan)
		var (
			viewport layout.Position
			ignore   Direction
		)
		for {
			var (
				su         stateUpdate
				newElems   []Element
				updateOnly []Element
				rmSerials  []Serial
			)
			select {
			case req, more := <-reqChan:
				if !more {
					return
				}
				switch req := req.(type) {
				case modificationRequest:
					newElems = req.NewOrUpdate
					rmSerials = req.Remove
					updateOnly = req.UpdateOnly
					su.PreserveListEnd = true

					/*
						Remove any elements that sort outside the boundaries of the
						current list.
					*/
					SliceFilter(&newElems, func(elem Element) bool {
						if len(synthesis.Source) == 0 {
							return true
						}
						sortsBefore := compact.Comparator(elem, synthesis.Source[0])
						sortsAfter := compact.Comparator(synthesis.Source[len(synthesis.Source)-1], elem)
						// If this element sorts before the beginning of the list or after
						// the end of the list, it should not be inserted unless we are at
						// the appropriate end of the list.
						switch {
						case sortsBefore && ignore == Before:
							return true
						case sortsAfter && ignore == After:
							return true
						case sortsBefore || sortsAfter:
							return false
						default:
							return true
						}
					})
					ignore = noDirection
				case loadRequest:
					viewport = req.viewport
					if req.Direction == ignore {
						continue
					}

					// Find the serial of the element at either end of the list.
					var loadSerial Serial
					switch req.Direction {
					case Before:
						loadSerial = synthesis.SerialAt(0)
					case After:
						loadSerial = synthesis.SerialAt(len(synthesis.Source) - 1)
					}
					// Load new elements.
					newElems = append(newElems, hooks.Loader(req.Direction, loadSerial)...)
					// Track whether all new elements in a given direction have been
					// exhausted.
					if len(newElems) == 0 {
						ignore = req.Direction
					} else {
						ignore = noDirection
					}
				}
			}
			// Define the elements within the viewport before any modification
			// to the underlying slice of elements.
			vpStart, vpEnd := synthesis.ViewportToSerials(viewport)
			// Apply state updates.
			compact.Apply(newElems, updateOnly, rmSerials)
			// Fetch new contents and list of compacted content.
			contents, compacted := compact.Compact(vpStart, vpEnd)
			su.CompactedSerials = compacted
			// Synthesize elements based on new contents.
			synthesis = Synthesize(contents, hooks.Synthesizer)
			su.populateWith(synthesis.Elements)

			updateChan <- su
			hooks.Invalidator()
		}
	}()
	return reqChan, updateChan
}
