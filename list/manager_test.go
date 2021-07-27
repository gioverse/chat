package list

import (
	"image"
	"strconv"
	"testing"
	"time"

	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

type actuallyStatefulElement struct {
	serial string
}

func (a actuallyStatefulElement) Serial() Serial {
	return Serial(a.serial)
}

func TestStateUpdate(t *testing.T) {
	elems := []Element{
		actuallyStatefulElement{serial: "a"},
		actuallyStatefulElement{serial: "b"},
		actuallyStatefulElement{serial: ""},
		actuallyStatefulElement{serial: ""},
		actuallyStatefulElement{serial: "c"},
		actuallyStatefulElement{serial: "d"},
	}

	var su stateUpdate
	su.populateWith(elems)

	for i := range su.Elements {
		if elems[i].Serial() != su.Elements[i].Serial() {
			t.Errorf("expected state update at index %d to match input", i)
		}
	}

	for s, i := range su.SerialToIndex {
		if elems[i].Serial() != s {
			t.Errorf("state update mapping for serial %s is wrong", s)
		}
	}

	if _, ok := su.SerialToIndex[NoSerial]; ok {
		t.Errorf("state update should not map the lack of a serial to an index")
	}
}

func TestManager(t *testing.T) {
	// create a fake rendering context
	var ops op.Ops
	gtx := layout.NewContext(&ops, system.FrameEvent{
		Now: time.Now(),
		Metric: unit.Metric{
			PxPerDp: 1,
			PxPerSp: 1,
		},
		Size: image.Pt(1000, 1000),
	})
	var list layout.List

	allocationCounter := 0
	presenterCounter := 0

	m := NewManager(6, Hooks{
		Allocator: func(e Element) interface{} {
			allocationCounter++
			switch e.(type) {
			case actuallyStatefulElement:
				// Just allocate something, doesn't matter what.
				return actuallyStatefulElement{}
			}
			return nil
		},
		Presenter: func(e Element, state interface{}) layout.Widget {
			presenterCounter++
			switch e.(type) {
			case actuallyStatefulElement:
				// Trigger a panic if the wrong state type was provided.
				_ = state.(actuallyStatefulElement)
			}
			return layout.Spacer{
				Width:  unit.Dp(5),
				Height: unit.Dp(5),
			}.Layout
		},
		Loader: func(dir Direction, relativeTo Serial) []Element {
			return nil
		},
		Invalidator: func() {},
		Comparator:  func(a, b Element) bool { return true },
		Synthesizer: func(a, b, c Element) []Element { return nil },
	})
	// Shut down the existing background processing for this manager.
	close(m.requests)

	// Replace the background processing channels with channels we can control
	// from within the test.
	requests := make(chan interface{}, 1)
	updates := make(chan stateUpdate, 1)
	m.requests = requests
	m.stateUpdates = updates

	var persistentElements []Element

	type testcase struct {
		name                  string
		expectingRequest      bool
		expectedRequest       loadRequest
		sendUpdate            bool
		update                stateUpdate
		expectedAllocations   int
		expectedPresentations int
		stateSize             int
	}
	for _, tc := range []testcase{
		{
			name:       "load inital elements",
			sendUpdate: true,
			update: func() stateUpdate {
				// Send an update to provide a few elements to work with.
				persistentElements = testElements[0:3]
				var su stateUpdate
				su.populateWith(persistentElements)
				return su
			}(),
			expectedAllocations:   3,
			expectedPresentations: 3,
			expectingRequest:      true,
			expectedRequest: loadRequest{
				Direction: Before,
			},
			stateSize: 3,
		},
		{
			name:       "load stateless elements (shouldn't allocate, should present)",
			sendUpdate: true,
			update: func() stateUpdate {
				// Send an update to provide a few elements to work with.
				persistentElements = append(persistentElements,
					testElement{
						synthCount: 1,
						serial:     string(NoSerial),
					},
					testElement{
						synthCount: 1,
						serial:     string(NoSerial),
					})
				var su stateUpdate
				su.populateWith(persistentElements)
				return su
			}(),
			expectedAllocations:   0,
			expectedPresentations: 5,
			expectingRequest:      true,
			expectedRequest: loadRequest{
				Direction: Before,
			},
			stateSize: 3,
		},
		{
			name:       "load a truly stateful element",
			sendUpdate: true,
			update: func() stateUpdate {
				// Send an update to provide a few elements to work with.
				persistentElements = append(persistentElements, actuallyStatefulElement{
					serial: "serial",
				})
				var su stateUpdate
				su.populateWith(persistentElements)
				return su
			}(),
			expectedAllocations:   1,
			expectedPresentations: 6,
			expectingRequest:      true,
			expectedRequest: loadRequest{
				Direction: Before,
			},
			stateSize: 4,
		},
		{
			name:       "compact the stateful element",
			sendUpdate: true,
			update: func() stateUpdate {
				// Send an update to provide a few elements to work with.
				persistentElements = persistentElements[:len(persistentElements)-1]
				var su stateUpdate
				su.populateWith(persistentElements)
				su.CompactedSerials = []Serial{"serial"}
				return su
			}(),
			expectedAllocations:   0,
			expectedPresentations: 5,
			expectingRequest:      true,
			expectedRequest: loadRequest{
				Direction: Before,
			},
			stateSize: 3,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Send a state update if configured.
			if tc.sendUpdate {
				updates <- tc.update
			}

			// Lay out the managed list.
			list.Layout(gtx, m.UpdatedLen(&list), m.Layout)

			// Ensure the hooks were invoked the expected number of times.
			if allocationCounter != tc.expectedAllocations {
				t.Errorf("expected allocator to be called %d times, was called %d", tc.expectedAllocations, allocationCounter)
			}
			if presenterCounter != tc.expectedPresentations {
				t.Errorf("expected presenter to be called %d times, was called %d", tc.expectedPresentations, presenterCounter)
			}
			presenterCounter = 0
			allocationCounter = 0

			// Check the loadRequest, if any.
			var request loadRequest
			select {
			case req := <-requests:
				request = req.(loadRequest)
				if !tc.expectingRequest {
					t.Errorf("did not expect load request %v", request)
				} else if tc.expectedRequest.Direction != request.Direction {
					t.Errorf("expected loadRequest %v, got %v", tc.expectedRequest, request)
				}
			default:
			}
			if tc.stateSize != len(m.elementState) {
				t.Errorf("expected %d states allocated, got %d", tc.stateSize, len(m.elementState))
			}
		})
	}
}

// TestManagerPreftch tests that load requests are only issued when the index
// falls into the prefetch zones, and that any requests are for the proper
// direction.
func TestManagerPrefetch(t *testing.T) {
	for _, tc := range []struct {
		// name of test case.
		name string
		// prefetch percentage.
		prefetch float32
		// elements to allocate in the list.
		elements int
		// index to scroll to.
		index int
		// expcted request, nil if we are not expecting anything.
		expect *loadRequest
	}{
		{
			name:     "index outside prefetch zone: no prefetch",
			prefetch: 0.2,
			elements: 10,
			index:    5,
			expect:   nil,
		},
		{
			name:     "index on prefetch boundary: no prefetch",
			prefetch: 0.2,
			elements: 10,
			index:    2,
			expect:   nil,
		},
		{
			name:     "index on prefetch boundary: no prefetch",
			prefetch: 0.2,
			elements: 10,
			index:    7,
			expect:   nil,
		},
		{
			name:     "index inside 'before' zone: prefetch before",
			prefetch: 0.2,
			elements: 10,
			index:    1,
			expect:   &loadRequest{Direction: Before},
		},
		{
			name:     "index inside 'before' zone: prefetch before",
			prefetch: 0.2,
			elements: 10,
			index:    0,
			expect:   &loadRequest{Direction: Before},
		},
		{
			name:     "index inside 'after' zone: prefetch after",
			prefetch: 0.2,
			elements: 10,
			index:    8,
			expect:   &loadRequest{Direction: After},
		},
		{
			name:     "index inside 'after' zone: prefetch after",
			prefetch: 0.2,
			elements: 10,
			index:    9,
			expect:   &loadRequest{Direction: After},
		},
		{
			name:     "index greater than bounds: load after",
			prefetch: 0.2,
			elements: 10,
			index:    10, // zero-index means the last index would be '9'
			expect:   &loadRequest{Direction: After},
		},
		{
			name:     "index less than bounds: load before",
			prefetch: 0.2,
			elements: 10,
			index:    -1,
			expect:   &loadRequest{Direction: Before},
		},
		{
			name:     "zeroed prefetch should default",
			prefetch: 0.0,
			elements: 10,
			index:    1,
			expect:   &loadRequest{Direction: Before},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var (
				ops op.Ops
				gtx = layout.NewContext(&ops, system.FrameEvent{
					Now: time.Now(),
					Metric: unit.Metric{
						PxPerDp: 1,
						PxPerSp: 1,
					},
					Size: image.Pt(1, 1),
				})
				list         layout.List
				requests     = make(chan interface{}, 1)
				stateUpdates = make(chan stateUpdate, 1)
			)

			// Manually allocate list manager.
			// Constructor doesn't help with testing.
			m := Manager{
				Prefetch:     tc.prefetch,
				presenter:    testHooks.Presenter,
				allocator:    testHooks.Allocator,
				requests:     requests,
				stateUpdates: stateUpdates,
				elementState: make(map[Serial]interface{}),
				elements: func() (elements []Element) {
					elements = make([]Element, tc.elements)
					for ii := 0; ii < tc.elements; ii++ {
						elements[ii] = &actuallyStatefulElement{
							serial: strconv.Itoa(ii),
						}
					}
					return elements
				}(),
			}

			// Set the index position to render the list at.
			list.Position = layout.Position{
				BeforeEnd: true,
				First:     tc.index,
				Count:     1,
			}

			// Lay out the managed list.
			list.Layout(gtx, m.UpdatedLen(&list), m.Layout)

			select {
			case req := <-requests:
				request := req.(loadRequest)
				if tc.expect == nil {
					t.Errorf("did not expect load request %v", request)
				}
				if tc.expect != nil && tc.expect.Direction != request.Direction {
					t.Errorf("got %v, want %v", request, tc.expect)
				}
			default:
			}

		})
	}
}

// dupSlice returns a slice composed of the same elements in the same order,
// but backed by a different array.
func dupSlice(in []Element) []Element {
	out := make([]Element, len(in))
	for i := range in {
		out[i] = in[i]
	}
	return out
}

func TestManagerViewportOnRemoval(t *testing.T) {
	// create a fake rendering context
	var ops op.Ops
	gtx := layout.NewContext(&ops, system.FrameEvent{
		Now: time.Now(),
		Metric: unit.Metric{
			PxPerDp: 1,
			PxPerSp: 1,
		},
		// Make the viewport small enough that only a few list elements
		// fit on screen at a time.
		Size: image.Pt(10, 10),
	})
	var list layout.List

	m := NewManager(6, Hooks{
		Allocator: func(e Element) interface{} { return nil },
		Presenter: func(e Element, state interface{}) layout.Widget {
			return layout.Spacer{
				Width:  unit.Dp(5),
				Height: unit.Dp(5),
			}.Layout
		},
		Loader:      func(dir Direction, relativeTo Serial) []Element { return nil },
		Invalidator: func() {},
		Comparator:  func(a, b Element) bool { return true },
		Synthesizer: func(a, b, c Element) []Element { return nil },
	})
	// Shut down the existing background processing for this manager.
	close(m.requests)

	// Replace the background processing channels with channels we can control
	// from within the test.
	updates := make(chan stateUpdate, 1)
	m.requests = nil
	m.stateUpdates = updates

	var persistentElements []Element

	type testcase struct {
		name                           string
		sendUpdate                     bool
		update                         stateUpdate
		startFirstIndex, endFirstIndex int
	}
	for _, tc := range []testcase{
		{
			name:       "load inital elements",
			sendUpdate: true,
			update: func() stateUpdate {
				// Send an update to provide a few elements to work with.
				persistentElements = dupSlice(testElements)
				var su stateUpdate
				su.populateWith(persistentElements)
				return su
			}(),
			startFirstIndex: 0,
			endFirstIndex:   0,
		},
		{
			name:       "remove first visible element",
			sendUpdate: true,
			update: func() stateUpdate {
				persistentElements = dupSlice(append(testElements[0:2], testElements[3:]...))
				var su stateUpdate
				su.populateWith(persistentElements)
				return su
			}(),
			startFirstIndex: 2,
			endFirstIndex:   1,
		},
		{
			name:       "replace all elements, inserting many non-serial elements",
			sendUpdate: true,
			update: func() stateUpdate {
				persistentElements = dupSlice([]Element{
					testElement{
						serial:     string(NoSerial),
						synthCount: 1,
					},
					testElement{
						serial:     string(NoSerial),
						synthCount: 1,
					},
					testElements[len(testElements)-1],
					testElement{
						serial:     string(NoSerial),
						synthCount: 1,
					},
					testElement{
						serial:     string(NoSerial),
						synthCount: 1,
					},
				})
				var su stateUpdate
				su.populateWith(persistentElements)
				return su
			}(),
			startFirstIndex: 0,
			endFirstIndex:   0,
		},
		{
			name:       "remove the one stateful element",
			sendUpdate: true,
			update: func() stateUpdate {
				persistentElements = dupSlice([]Element{
					testElement{
						serial:     string(NoSerial),
						synthCount: 1,
					},
					testElement{
						serial:     string(NoSerial),
						synthCount: 1,
					},
					testElement{
						serial:     string(NoSerial),
						synthCount: 1,
					},
					testElement{
						serial:     string(NoSerial),
						synthCount: 1,
					},
				})
				var su stateUpdate
				su.populateWith(persistentElements)
				return su
			}(),
			startFirstIndex: 2,
			endFirstIndex:   0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Send a state update if configured.
			if tc.sendUpdate {
				updates <- tc.update
			}

			// Lay out the managed list.
			list.Position.First = tc.startFirstIndex
			list.Layout(gtx, m.UpdatedLen(&list), m.Layout)

			// Ensure that the viewport is in the right place afterward.
			if list.Position.First != tc.endFirstIndex {
				t.Errorf("Expected list.Position.First to be %d after layout, got %d",
					tc.endFirstIndex, list.Position.First)
			}
		})
	}
}

// testHooks allocates default hooks for testing.
var testHooks = Hooks{
	Allocator: func(e Element) interface{} {
		switch e.(type) {
		case actuallyStatefulElement:
			// Just allocate something, doesn't matter what.
			return actuallyStatefulElement{}
		}
		return nil
	},
	Presenter: func(e Element, state interface{}) layout.Widget {
		switch e.(type) {
		case actuallyStatefulElement:
			// Trigger a panic if the wrong state type was provided.
			_ = state.(actuallyStatefulElement)
		}
		return layout.Spacer{
			Width:  unit.Dp(5),
			Height: unit.Dp(5),
		}.Layout
	},
	Loader: func(dir Direction, relativeTo Serial) []Element {
		return nil
	},
	Invalidator: func() {},
	Comparator:  func(a, b Element) bool { return true },
	Synthesizer: func(a, b, c Element) []Element { return nil },
}
