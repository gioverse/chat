package list

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"
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
	var more bool
	var loadInvoked bool
	hooks := Hooks{
		Invalidator: func() {},
		Comparator:  testComparator,
		Synthesizer: testSynthesizer,
		Loader: func(dir Direction, rt Serial) ([]Element, bool) {
			loadInvoked = true
			return nextLoad, more
		},
	}
	size := 6
	reqs, _, updates := asyncProcess(size, hooks)

	type testcase struct {
		// description of what this test case is checking
		name string
		// a request for data
		input loadRequest
		// the data that will be returned by the data request (if the loader is executed)
		load []Element
		// whether the async logic should expect additional content in the direction of
		// the load.
		loadMore bool
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
				viewport: viewport{
					Start: "",
					End:   "",
				},
				Direction: Before,
			},
			load:     testElements[7:],
			loadMore: true,
			expected: stateUpdate{
				Synthesis: Synthesis{
					Elements: testElements[7:],
					SerialToIndex: map[Serial]int{
						testElements[7].Serial(): 0,
						testElements[8].Serial(): 1,
						testElements[9].Serial(): 2,
					},
				},
			},
		},
		{
			name: "fetch content after (cannot succeed)",
			input: loadRequest{
				viewport: viewport{
					Start: "000",
					End:   "003",
				},
				Direction: After,
			},
			load: nil,
			expected: stateUpdate{
				Synthesis: Synthesis{
					Elements: testElements[7:],
					SerialToIndex: map[Serial]int{
						testElements[7].Serial(): 0,
						testElements[8].Serial(): 1,
						testElements[9].Serial(): 2,
					},
				},
			},
		},
		{
			name: "fetch content after again (should not attempt load)",
			input: loadRequest{
				viewport: viewport{
					Start: "000",
					End:   "003",
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
				viewport: viewport{
					Start: "000",
					End:   "003",
				},
				Direction: Before,
			},
			load:     testElements[4:7],
			loadMore: true,
			expected: stateUpdate{
				Synthesis: Synthesis{
					Elements: testElements[4:],
					SerialToIndex: map[Serial]int{
						testElements[4].Serial(): 0,
						testElements[5].Serial(): 1,
						testElements[6].Serial(): 2,
						testElements[7].Serial(): 3,
						testElements[8].Serial(): 4,
						testElements[9].Serial(): 5,
					},
				},
			},
		},
		{
			name: "fetch content after (cannot succeed, but should try anyway now that a different load succeeded)",
			input: loadRequest{
				viewport: viewport{
					Start: "000",
					End:   "006",
				},
				Direction: After,
			},
			expected: stateUpdate{
				Synthesis: Synthesis{
					Elements: testElements[4:],
					SerialToIndex: map[Serial]int{
						testElements[4].Serial(): 0,
						testElements[5].Serial(): 1,
						testElements[6].Serial(): 2,
						testElements[7].Serial(): 3,
						testElements[8].Serial(): 4,
						testElements[9].Serial(): 5,
					},
				},
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
				viewport: viewport{
					Start: "005",
					End:   "006",
				},
				Direction: Before,
			},
			load:     testElements[1:4],
			loadMore: true,
			expected: stateUpdate{
				Synthesis: Synthesis{
					Elements: testElements[3:9],
					SerialToIndex: map[Serial]int{
						testElements[3].Serial(): 0,
						testElements[4].Serial(): 1,
						testElements[5].Serial(): 2,
						testElements[6].Serial(): 3,
						testElements[7].Serial(): 4,
						testElements[8].Serial(): 5,
					},
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
				viewport: viewport{
					Start: "004",
					End:   "005",
				},
				Direction: Before,
			},
			load:     testElements[:3],
			loadMore: true,
			expected: stateUpdate{
				Synthesis: Synthesis{
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
				viewport: viewport{
					Start: "002",
					End:   "003",
				},
				Direction: Before,
			},
			load: nil,
			expected: stateUpdate{
				Synthesis: Synthesis{
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
		},
		{
			name: "fetch content before (should not attempt load)",
			input: loadRequest{
				viewport: viewport{
					Start: "000",
					End:   "006",
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
			more = tc.loadMore

			// request a load
			reqs <- tc.input

			// examine the update we get back
			if !tc.skipUpdate {
				update := <-updates
				if len(update) != 1 {
					t.Errorf("Expected 1 pending update, got %d", len(update))
				}
				if !updatesEqual(update[0], tc.expected) {
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

// TestCanModifyWhenIdle ensures that updates are queued if the reading
// side is idle (e.g. list manager is not currently being laid out).
func TestCanModifyWhenIdle(t *testing.T) {
	hooks := testHooks
	hooks.Comparator = func(i, j Element) bool {
		// For the purposes of this test, the sort order doesn't matter,
		// and we do not want to trigger the logic that filters insertions
		// at the beginning or end of the list.
		return false
	}
	requests, viewports, updates := asyncProcess(4, hooks)

	viewports <- viewport{
		Start: "0",
		End:   "5",
	}

	// Update with some number of elements.
	// The manager is not being laid out so
	// we expect these to queue up.
	requests <- modificationRequest{
		NewOrUpdate: []Element{testElement{serial: "1", synthCount: 0}},
		UpdateOnly:  []Element{},
		Remove:      []Serial{},
	}
	requests <- modificationRequest{
		NewOrUpdate: []Element{testElement{serial: "2", synthCount: 0}},
		UpdateOnly:  []Element{},
		Remove:      []Serial{},
	}
	requests <- modificationRequest{
		NewOrUpdate: []Element{testElement{serial: "3", synthCount: 0}},
		UpdateOnly:  []Element{},
		Remove:      []Serial{},
	}
	requests <- modificationRequest{
		NewOrUpdate: []Element{testElement{serial: "4", synthCount: 0}},
		UpdateOnly:  []Element{},
		Remove:      []Serial{},
	}

	close(requests)

	// Give time for async to shutdown.
	time.Sleep(time.Millisecond)

	// updates should have a queued value.
	if len(updates) != 1 {
		t.Fatalf("updates channel: expected 1 queued value, got %d", len(updates))
	}

	// We should recieve update elements 1, 2, 3, 4.
	total := 0
	var want []Element
	for pending := range updates {
		t.Log(pending)
		for ii := range pending {
			su := pending[ii]
			total++
			if su.Type != push {
				t.Errorf("expected push update, got %v", su.Type)
			}
			got := su.Synthesis.Source
			want = append(want, testElement{
				serial:     strconv.Itoa(total),
				synthCount: 0,
			})
			if !reflect.DeepEqual(got, want) {
				t.Errorf("state update: want %+v, got %+v", want, got)
			}
		}
	}
	if total != 4 {
		t.Fatalf("expected 4 pending updates, got %d", total)
	}
}
