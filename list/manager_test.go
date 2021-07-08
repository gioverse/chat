package list

import (
	"image"
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
		Synthesizer: func(a, b Element) []Element { return nil },
	})
	// Shut down the existing background processing for this manager.
	close(m.requests)

	// Replace the background processing channels with channels we can control
	// from within the test.
	requests := make(chan loadRequest, 1)
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
			case request = <-requests:
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

func requestsEqual(a, b loadRequest) bool {
	if a.Direction != b.Direction {
		return false
	}
	if a.viewport.First != b.viewport.First {
		return false
	}
	if a.viewport.Count != b.viewport.Count {
		return false
	}
	return true
}
