package list

import (
	"fmt"
)

type updateType uint8

const (
	// pull indicates the results from a load request. The update pulled
	// data from the data store.
	pull updateType = iota
	// push indicates the results from an asynchronous insertion of
	// data. The application pushed data to the list.
	push
)

func (u updateType) String() string {
	switch u {
	case pull:
		return "pull"
	case push:
		return "push"
	default:
		return "unknown"
	}
}

// stateUpdate contains a new slice of element data and a mapping from all of
// the element serials to their respective indicies. This data structure is designed
// to allow the UI code to quickly find and update any offsets and locations
// within the new data.
type stateUpdate struct {
	Synthesis
	// CompactedSerials is a slice of Serials that were compacted within this update.
	CompactedSerials []Serial
	// Ignore reports which directions (if any) the async backend currently
	// believes to have no new content.
	Ignore Direction
	Type   updateType
}

func (s stateUpdate) String() string {
	return fmt.Sprintf("Synthesis: %v, Compacted: %v, Ignore: %v", s.Synthesis, s.CompactedSerials, s.Ignore)
}

// viewport represents a range of elements visible within a list.
type viewport struct {
	Start, End Serial
}

// asyncProcess runs a list.processor concurrently.
// New elements are processed and compacted according to maxSize
// on each loadRequest. Close the loadRequest channel to terminate
// processing.
func asyncProcess(maxSize int, hooks Hooks) (chan<- interface{}, chan viewport, <-chan []stateUpdate) {
	compact := NewCompact(maxSize, hooks.Comparator)
	var synthesis Synthesis
	reqChan := make(chan interface{})
	updateChan := make(chan []stateUpdate, 1)
	viewports := make(chan viewport, 1)
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
					su.Type = push
					newElems = req.NewOrUpdate
					rmSerials = req.Remove
					updateOnly = req.UpdateOnly

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
					ignore = NoDirection
				case loadRequest:
					su.Type = pull
					viewport = req.viewport
					if ignore.Contains(req.Direction) {
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
					var more bool
					newElems, more = hooks.Loader(req.Direction, loadSerial)
					// Track whether all new elements in a given direction have been
					// exhausted.
					if len(newElems) == 0 || !more {
						ignore.Add(req.Direction)
					} else {
						ignore = NoDirection
					}
				}
			}
			// Apply state updates.
			compact.Apply(newElems, updateOnly, rmSerials)

			// Update the viewport if there is a new one available.
			select {
			case viewport = <-viewports:
			default:
			}

			// Fetch new contents and list of compacted content.
			contents, compacted := compact.Compact(viewport.Start, viewport.End)
			su.CompactedSerials = compacted
			// Synthesize elements based on new contents.
			synthesis = Synthesize(contents, hooks.Synthesizer)
			su.Synthesis = synthesis
			su.Ignore = ignore

			// Try send update. If the widget is not being actively laid out,
			// we don't want to block.
			select {
			case updateChan <- []stateUpdate{su}:
			default:
				fmt.Printf("update: %+v\n", su.Synthesis.Source)

				// Append latest update to the list.
				su := su
				pending := <-updateChan
				pending = append(pending, su)
				updateChan <- pending
			}

			hooks.Invalidator()
		}
	}()
	return reqChan, viewports, updateChan
}
