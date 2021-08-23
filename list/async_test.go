package list

import (
	"fmt"
	"reflect"
	"testing"

	"gioui.org/layout"
)

// define a set of elements that can be used across tests.
var testElements = func() []Element {
	testElements := []Element{}
	for i := 0; i < 10; i++ {
		testElements = append(testElements, testElement{
			serial:     fmt.Sprintf("%03d", i),
			synthCount: 1,
		})
	}
	return testElements
}()

func TestAsyncProcess(t *testing.T) {
	var nextLoad []Element
	var loadInvoked bool
	hooks := Hooks{
		Invalidator: func() {},
		Comparator:  testComparator,
		Synthesizer: testSynthesizer,
		Loader: func(dir Direction, rt Serial) []Element {
			loadInvoked = true
			return nextLoad
		},
	}
	size := 6
	reqs, updates := asyncProcess(size, hooks)

	type testcase struct {
		// description of what this test case is checking
		name string
		// a request for data
		input loadRequest
		// the data that will be returned by the data request (if the loader is executed)
		load []Element
		// should the testcase block waiting for an update on the update channel
		skipUpdate bool
		// the update to expect on the update channel
		expected stateUpdate
		// any conditions that should be checked after the rest of the logic
		extraChecks func() error
	}

	// Run each testcase sequentially on the same processor to check different
	// points during the lifecycle of its data.
	for _, tc := range []testcase{
		{
			name: "initial load of data",
			input: loadRequest{
				viewport: layout.Position{
					First: 0,
					Count: 0,
				},
				Direction: Before,
			},
			load: testElements[7:],
			expected: stateUpdate{
				Elements: testElements[7:],
				SerialToIndex: map[Serial]int{
					testElements[7].Serial(): 0,
					testElements[8].Serial(): 1,
					testElements[9].Serial(): 2,
				},
				CompactedSerials: nil,
			},
		},
		{
			name: "fetch content after (cannot succeed)",
			input: loadRequest{
				viewport: layout.Position{
					First: 0,
					Count: 3,
				},
				Direction: After,
			},
			load: nil,
			expected: stateUpdate{
				Elements: testElements[7:],
				SerialToIndex: map[Serial]int{
					testElements[7].Serial(): 0,
					testElements[8].Serial(): 1,
					testElements[9].Serial(): 2,
				},
				CompactedSerials: nil,
			},
		},
		{
			name: "fetch content after again (should not attempt load)",
			input: loadRequest{
				viewport: layout.Position{
					First: 0,
					Count: 3,
				},
				Direction: After,
			},
			skipUpdate: true,
			extraChecks: func() error {
				if loadInvoked {
					return fmt.Errorf("should not have invoked load after a load in the same direction returned nothing")
				}
				return nil
			},
		},
		{
			name: "fetch content before",
			input: loadRequest{
				viewport: layout.Position{
					First: 0,
					Count: 3,
				},
				Direction: Before,
			},
			load: testElements[4:7],
			expected: stateUpdate{
				Elements: testElements[4:],
				SerialToIndex: map[Serial]int{
					testElements[4].Serial(): 0,
					testElements[5].Serial(): 1,
					testElements[6].Serial(): 2,
					testElements[7].Serial(): 3,
					testElements[8].Serial(): 4,
					testElements[9].Serial(): 5,
				},
				CompactedSerials: nil,
			},
		},
		{
			name: "fetch content after (cannot succeed, but should try anyway now that a different load succeeded)",
			input: loadRequest{
				viewport: layout.Position{
					First: 0,
					Count: 6,
				},
				Direction: After,
			},
			expected: stateUpdate{
				Elements: testElements[4:],
				SerialToIndex: map[Serial]int{
					testElements[4].Serial(): 0,
					testElements[5].Serial(): 1,
					testElements[6].Serial(): 2,
					testElements[7].Serial(): 3,
					testElements[8].Serial(): 4,
					testElements[9].Serial(): 5,
				},
				CompactedSerials: nil,
			},
			extraChecks: func() error {
				if !loadInvoked {
					return fmt.Errorf("should have invoked load")
				}
				return nil
			},
		},
		{
			name: "fetch content before (should compact the end)",
			input: loadRequest{
				viewport: layout.Position{
					First: 1,
					Count: 2,
				},
				Direction: Before,
			},
			load: testElements[1:4],
			expected: stateUpdate{
				Elements: testElements[3:9],
				SerialToIndex: map[Serial]int{
					testElements[3].Serial(): 0,
					testElements[4].Serial(): 1,
					testElements[5].Serial(): 2,
					testElements[6].Serial(): 3,
					testElements[7].Serial(): 4,
					testElements[8].Serial(): 5,
				},
				CompactedSerials: []Serial{
					testElements[1].Serial(),
					testElements[2].Serial(),
					testElements[9].Serial(),
				},
			},
		},
		{
			name: "fetch content before (should compact the end a little more)",
			input: loadRequest{
				viewport: layout.Position{
					First: 1,
					Count: 2,
				},
				Direction: Before,
			},
			load: testElements[:3],
			expected: stateUpdate{
				Elements: testElements[2:8],
				SerialToIndex: map[Serial]int{
					testElements[2].Serial(): 0,
					testElements[3].Serial(): 1,
					testElements[4].Serial(): 2,
					testElements[5].Serial(): 3,
					testElements[6].Serial(): 4,
					testElements[7].Serial(): 5,
				},
				CompactedSerials: []Serial{
					testElements[0].Serial(),
					testElements[1].Serial(),
					testElements[8].Serial(),
				},
			},
		},
		{
			name: "fetch content before (no more content)",
			input: loadRequest{
				viewport: layout.Position{
					First: 0,
					Count: 1,
				},
				Direction: Before,
			},
			load: nil,
			expected: stateUpdate{
				Elements: testElements[2:8],
				SerialToIndex: map[Serial]int{
					testElements[2].Serial(): 0,
					testElements[3].Serial(): 1,
					testElements[4].Serial(): 2,
					testElements[5].Serial(): 3,
					testElements[6].Serial(): 4,
					testElements[7].Serial(): 5,
				},
			},
		},
		{
			name: "fetch content before (should not attempt load)",
			input: loadRequest{
				viewport: layout.Position{
					First: 0,
					Count: 6,
				},
				Direction: Before,
			},
			skipUpdate: true,
			extraChecks: func() error {
				if loadInvoked {
					return fmt.Errorf("should not have invoked load after a load in the same direction returned nothing")
				}
				return nil
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// ensure that the next invocation of the loader will load this
			// testcase's data payload.
			nextLoad = tc.load

			// request a load
			reqs <- tc.input

			// examine the update we get back
			if !tc.skipUpdate {
				update := <-updates
				if !updatesEqual(update, tc.expected) {
					t.Errorf("Expected %v, got %v", tc.expected, update)
				}
			}
			if tc.extraChecks != nil {
				if err := tc.extraChecks(); err != nil {
					t.Error(err)
				}
			}
			loadInvoked = false
		})
	}
}

func updatesEqual(a, b stateUpdate) bool {
	if !elementsEqual(a.Elements, b.Elements) {
		return false
	}
	if !serialsEqual(a.CompactedSerials, b.CompactedSerials) {
		return false
	}
	return reflect.DeepEqual(a.SerialToIndex, b.SerialToIndex)
}
