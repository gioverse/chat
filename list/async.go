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
}

// populateWith sets s.Elements to the provided slice of elements
// and populates the SerialToIndex map.
func (s *stateUpdate) populateWith(elems []Element) {
	s.Elements = elems
	s.SerialToIndex = make(map[Serial]int)
	for index, elem := range s.Elements {
		s.SerialToIndex[elem.Serial()] = index
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
				su        stateUpdate
				newElems  []Element
				rmSerials []Serial
			)
			select {
			case req, more := <-reqChan:
				switch req := req.(type) {
				case modificationRequest:
					newElems = req.Update
					rmSerials = req.Remove
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
			processor.Update(newElems, rmSerials)
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
