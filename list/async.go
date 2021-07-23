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
	processor := newProcessor(hooks.Synthesizer, hooks.Comparator)
	reqChan := make(chan interface{})
	updateChan := make(chan stateUpdate, 1)
	go func() {
		defer close(updateChan)
		var (
			viewport        layout.Position
			ignoreDirection Direction
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
				switch req := req.(type) {
				case modificationRequest:
					newElems = req.NewOrUpdate
					rmSerials = req.Remove
					updateOnly = req.UpdateOnly
					su.PreserveListEnd = true

					/*
						In order to preserve the invariant that the Raw list contains a
						contiguous slice of elements, we need to remove any elements
						from the update that sort to the beginning or end of the Raw list
						unless we are at the beginning or end of the underlying data.
						This is because we cannot tell how far away new elements are from the
						beginning or end of the list, and therefore how many elements
						might exist between them and the current boundaries of the list.
						The loader hook will serve the new elements to us at their appropriate
						position, so we should rely upon it to do so.
					*/
					sliceFilter(&newElems, func(elem Element) bool {
						if len(processor.Raw) == 0 {
							return true
						}
						beginning := processor.Comparator(elem, processor.Raw[0])
						end := processor.Comparator(processor.Raw[len(processor.Raw)-1], elem)
						// If this element sorts before the beginning of the list or after
						// the end of the list, it should not be inserted unless we are at
						// the appropriate end of the list.
						switch {
						case beginning && ignoreDirection == Before:
							return true
						case end && ignoreDirection == After:
							return true
						case beginning || end:
							return false
						}
						return true
					})
					ignoreDirection = noDirection
				case loadRequest:
					if !more {
						return
					}
					if req.Direction == ignoreDirection {
						continue
					}
					viewport = req.viewport

					// Find the serial of the element at either end of the list.
					var loadSerial Serial
					switch req.Direction {
					case Before:
						loadSerial = processor.SerialForProcessedIndex(0)
					case After:
						loadSerial = processor.SerialForProcessedIndex(len(processor.ProcessedToRaw) - 1)
					}
					// Load new elements.
					newElems = hooks.Loader(req.Direction, loadSerial)
					// Track whether all new elements in a given direction have been
					// exhausted.
					if len(newElems) == 0 {
						ignoreDirection = req.Direction
					} else {
						ignoreDirection = noDirection
					}
				}
			}
			// Process any new elements.
			processor.Update(newElems, updateOnly, rmSerials)
			su.populateWith(processor.Synthesize())

			// Always try to compact after a state update.
			if len(processor.Raw) > maxSize {
				su.CompactedSerials = processor.Compact(maxSize, viewport)
				// Reprocess elements if we compacted any.
				if len(su.CompactedSerials) > 0 {
					su.populateWith(processor.Synthesize())
				}
			}
			updateChan <- su
			hooks.Invalidator()
		}
	}()
	return reqChan, updateChan
}
