package list

import "fmt"

// stateUpdate contains a new slice of element data and a mapping from all of
// the element serials to their respective indicies. This data structure is designed
// to allow the UI code to quickly find and update any offsets and locations
// within the new data.
type stateUpdate struct {
	Synthesis
	// CompactedSerials is a slice of Serials that were compacted within this update.
	CompactedSerials []Serial
	// PreserveListEnd indicates whether or not the list.Position.BeforeEnd field
	// should be reset when applying this state update.
	PreserveListEnd bool
}

func (s stateUpdate) String() string {
	return fmt.Sprintf("Synthesis: %v, Compacted: %v, Preserve: %v", s.Synthesis, s.CompactedSerials, s.PreserveListEnd)
}

// viewport represents a range of elements visible within a list.
type viewport struct {
	Start, End Serial
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
			viewport viewport
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
			// Apply state updates.
			compact.Apply(newElems, updateOnly, rmSerials)
			// Fetch new contents and list of compacted content.
			contents, compacted := compact.Compact(viewport.Start, viewport.End)
			su.CompactedSerials = compacted
			// Synthesize elements based on new contents.
			synthesis = Synthesize(contents, hooks.Synthesizer)
			su.Synthesis = synthesis

			updateChan <- su
			hooks.Invalidator()
		}
	}()
	return reqChan, updateChan
}
