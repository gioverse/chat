package list

import (
	"reflect"
	"strings"
	"testing"

	"gioui.org/layout"
)

type testElement struct {
	serial     string
	synthCount int
}

func (t testElement) Serial() Serial {
	return Serial(t.serial)
}

func testSynthesizer(previous, current, next Element) []Element {
	out := []Element{}
	for i := 0; i < current.(testElement).synthCount; i++ {
		out = append(out, current)
	}
	return out
}

func testComparator(a, b Element) bool {
	return strings.Compare(string(a.Serial()), string(b.Serial())) < 0
}

func elementsEqual(a, b []Element) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func serialsEqual(a, b []Serial) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

// TestProcessorSynthesize ensures that processing elements sorts them and
// synthesizes new elements using the hooks provided to the processor.
func TestProcessorSynthesize(t *testing.T) {
	type testcase struct {
		name   string
		input  []Element
		output []Element
	}

	for _, tc := range []testcase{
		{
			name: "empty produces empty",
		},
		{
			name: "one produces none",
			input: []Element{
				testElement{
					serial:     "a",
					synthCount: 0,
				},
			},
		},
		{
			name: "one produces one",
			input: []Element{
				testElement{
					serial:     "a",
					synthCount: 1,
				},
			},
			output: []Element{
				testElement{
					serial:     "a",
					synthCount: 1,
				},
			},
		},
		{
			name: "one produces many",
			input: []Element{
				testElement{
					serial:     "a",
					synthCount: 3,
				},
			},
			output: []Element{
				testElement{
					serial:     "a",
					synthCount: 3,
				},
				testElement{
					serial:     "a",
					synthCount: 3,
				},
				testElement{
					serial:     "a",
					synthCount: 3,
				},
			},
		},
		{
			name: "many produces many",
			input: []Element{
				testElement{
					serial:     "a",
					synthCount: 1,
				},
				testElement{
					serial:     "b",
					synthCount: 2,
				},
				testElement{
					serial:     "c",
					synthCount: 1,
				},
			},
			output: []Element{
				testElement{
					serial:     "a",
					synthCount: 1,
				},
				testElement{
					serial:     "b",
					synthCount: 2,
				},
				testElement{
					serial:     "b",
					synthCount: 2,
				},
				testElement{
					serial:     "c",
					synthCount: 1,
				},
			},
		},
		{
			name: "many unsorted produces many sorted",
			input: []Element{
				testElement{
					serial:     "c",
					synthCount: 1,
				},
				testElement{
					serial:     "b",
					synthCount: 2,
				},
				testElement{
					serial:     "a",
					synthCount: 1,
				},
			},
			output: []Element{
				testElement{
					serial:     "a",
					synthCount: 1,
				},
				testElement{
					serial:     "b",
					synthCount: 2,
				},
				testElement{
					serial:     "b",
					synthCount: 2,
				},
				testElement{
					serial:     "c",
					synthCount: 1,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newProcessor(testSynthesizer, testComparator)
			p.Update(tc.input, nil, nil)
			processed := p.Synthesize()
			if !elementsEqual(processed, tc.output) {
				t.Errorf("Expected %v, got %v", tc.output, processed)
			}
		})
	}
}

// TestProcessorModify ensures that the processor correctly updates
// elements with that share serial values with elements that are already in
// the managed list.
func TestProcessorModify(t *testing.T) {
	p := newProcessor(testSynthesizer, testComparator)
	type testcase struct {
		name         string
		newOrUpdates []Element
		updates      []Element
		removals     []Serial
		output       []Element
	}
	// Run all testcases on a single processor instance to check its behavior over time.
	for _, tc := range []testcase{
		{
			name: "empty produces empty",
		},
		{
			name: "one produces one",
			newOrUpdates: []Element{
				testElement{
					serial:     "a",
					synthCount: 1,
				},
			},
			output: []Element{
				testElement{
					serial:     "a",
					synthCount: 1,
				},
			},
		},
		{
			name: "updating nonexistent changes nothing",
			updates: []Element{
				testElement{
					serial:     "b",
					synthCount: 1,
				},
			},
			output: []Element{
				testElement{
					serial:     "a",
					synthCount: 1,
				},
			},
		},
		{
			name: "updating existing take effect",
			updates: []Element{
				testElement{
					serial:     "a",
					synthCount: 0,
				},
			},
		},
		{
			name: "one updated produces two",
			newOrUpdates: []Element{
				testElement{
					serial:     "a",
					synthCount: 2,
				},
			},
			output: []Element{
				testElement{
					serial:     "a",
					synthCount: 2,
				},
				testElement{
					serial:     "a",
					synthCount: 2,
				},
			},
		},
		{
			name: "insert some more data",
			newOrUpdates: []Element{
				testElement{
					serial:     "a",
					synthCount: 1,
				},
				testElement{
					serial:     "b",
					synthCount: 1,
				},
				testElement{
					serial:     "c",
					synthCount: 1,
				},
				testElement{
					serial:     "d",
					synthCount: 1,
				},
			},
			output: []Element{
				testElement{
					serial:     "a",
					synthCount: 1,
				},
				testElement{
					serial:     "b",
					synthCount: 1,
				},
				testElement{
					serial:     "c",
					synthCount: 1,
				},
				testElement{
					serial:     "d",
					synthCount: 1,
				},
			},
		},
		{
			name: "update every other element",
			newOrUpdates: []Element{
				testElement{
					serial:     "a",
					synthCount: 2,
				},
				testElement{
					serial:     "b",
					synthCount: 1,
				},
				testElement{
					serial:     "c",
					synthCount: 2,
				},
				testElement{
					serial:     "d",
					synthCount: 1,
				},
			},
			output: []Element{
				testElement{
					serial:     "a",
					synthCount: 2,
				},
				testElement{
					serial:     "a",
					synthCount: 2,
				},
				testElement{
					serial:     "b",
					synthCount: 1,
				},
				testElement{
					serial:     "c",
					synthCount: 2,
				},
				testElement{
					serial:     "c",
					synthCount: 2,
				},
				testElement{
					serial:     "d",
					synthCount: 1,
				},
			},
		},
		{
			name: "remove first element",
			removals: []Serial{
				"a",
			},
			output: []Element{
				testElement{
					serial:     "b",
					synthCount: 1,
				},
				testElement{
					serial:     "c",
					synthCount: 2,
				},
				testElement{
					serial:     "c",
					synthCount: 2,
				},
				testElement{
					serial:     "d",
					synthCount: 1,
				},
			},
		},
		{
			name: "remove last element",
			removals: []Serial{
				"d",
			},
			output: []Element{
				testElement{
					serial:     "b",
					synthCount: 1,
				},
				testElement{
					serial:     "c",
					synthCount: 2,
				},
				testElement{
					serial:     "c",
					synthCount: 2,
				},
			},
		},
		{
			name: "remove all elements",
			removals: []Serial{
				"b", "c",
			},
			output: []Element{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p.Update(tc.newOrUpdates, tc.updates, tc.removals)
			processed := p.Synthesize()
			if !elementsEqual(processed, tc.output) {
				t.Errorf("Expected %v, got %v", tc.output, processed)
			}
		})
	}
}

var compactionList = []Element{
	testElement{
		serial:     "a",
		synthCount: 1,
	},
	testElement{
		serial:     "b",
		synthCount: 1,
	},
	testElement{
		serial:     "c",
		synthCount: 1,
	},
	testElement{
		serial:     "d",
		synthCount: 1,
	},
	testElement{
		serial:     "e",
		synthCount: 1,
	},
	testElement{
		serial:     "f",
		synthCount: 1,
	},
	testElement{
		serial:     "g",
		synthCount: 1,
	},
}

// TestProcessorCompact ensures that the compaction algorithm deallocates the
// expected list elements.
func TestProcessorCompact(t *testing.T) {
	type compactionRequest struct {
		MaxSize  int
		Viewport layout.Position
	}

	type testcase struct {
		name      string
		input     []Element
		req       compactionRequest
		compacted []Serial
	}

	for _, tc := range []testcase{
		{
			name: "empty list does not compact",
			req: compactionRequest{
				MaxSize: len(compactionList),
			},
		},
		{
			name:  "compact to size larger than list",
			input: compactionList,
			req: compactionRequest{
				MaxSize: len(compactionList),
			},
			compacted: nil,
		},
		{
			name:  "compact at beginning",
			input: compactionList,
			req: compactionRequest{
				MaxSize: len(compactionList) - 2,
			},
			compacted: []Serial{"f", "g"},
		},
		{
			name:  "compact at end",
			input: compactionList,
			req: compactionRequest{
				MaxSize: len(compactionList) - 2,
				Viewport: layout.Position{
					First: 2,
					Count: len(compactionList) - 2,
				},
			},
			compacted: []Serial{"a", "b"},
		},
		{
			name:  "compact in middle",
			input: compactionList,
			req: compactionRequest{
				MaxSize: len(compactionList) - 4,
				Viewport: layout.Position{
					First: 2,
					Count: len(compactionList) - 4,
				},
			},
			compacted: []Serial{"a", "b", "f", "g"},
		},
		{
			name:  "compact to size smaller than viewport at beginning",
			input: compactionList,
			req: compactionRequest{
				MaxSize: len(compactionList) / 2,
				Viewport: layout.Position{
					First: 0,
					Count: len(compactionList),
				},
			},
			compacted: []Serial{"d", "e", "f", "g"},
		},
		{
			name:  "compact to size smaller than viewport at end",
			input: compactionList,
			req: compactionRequest{
				MaxSize: len(compactionList) / 2,
				Viewport: layout.Position{
					First: 4,
					Count: len(compactionList) - 4,
				},
			},
			compacted: []Serial{"a", "b", "c", "d"},
		},
		{
			name:  "compact to default size (3x viewport)",
			input: compactionList,
			req: compactionRequest{
				Viewport: layout.Position{
					First: 4,
					Count: 2,
				},
			},
			compacted: []Serial{"a"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newProcessor(testSynthesizer, testComparator)
			p.Update(tc.input, nil, nil)
			_ = p.Synthesize()
			compacted := p.Compact(tc.req.MaxSize, tc.req.Viewport)
			if !serialsEqual(compacted, tc.compacted) {
				t.Errorf("Expected %v, got %v", tc.compacted, compacted)
			}
		})
	}
}

// TestSynthesizerListBoundaries ensures that the Synthesizer is given special
// sentinel values to indicate the absolutes start and absolute end of the
// managed list.
//
// The test uses a synth implementation that inserts elements when it sees the
// sentinel values.
// The test then checks for the presence of those inserted elements to verify
// that the sentinel values were sent correctly.
func TestSynthesizerListBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name   string
		input  []Element
		output []Element
	}{
		{
			name:   "empty: synth never called; expect empty slice",
			output: []Element{},
		},
		{
			name: "single element: expect both pseudo elements",
			input: []Element{
				testElement{serial: "1", synthCount: 1},
			},
			output: []Element{
				Start{},
				testElement{serial: "1", synthCount: 1},
				End{},
			},
		},
		{
			name: "two element: expect both pseudo elements",
			input: []Element{
				testElement{serial: "1", synthCount: 1},
				testElement{serial: "2", synthCount: 1},
			},
			output: []Element{
				Start{},
				testElement{serial: "1", synthCount: 1},
				testElement{serial: "2", synthCount: 1},
				End{},
			},
		},
		{
			name: "three element: expect both pseudo elements",
			input: []Element{
				testElement{serial: "1", synthCount: 1},
				testElement{serial: "2", synthCount: 1},
				testElement{serial: "3", synthCount: 1},
			},
			output: []Element{
				Start{},
				testElement{serial: "1", synthCount: 1},
				testElement{serial: "2", synthCount: 1},
				testElement{serial: "3", synthCount: 1},
				End{},
			},
		},
		{
			name: "many element: expect both pseudo elements",
			input: []Element{
				testElement{serial: "1", synthCount: 1},
				testElement{serial: "2", synthCount: 1},
				testElement{serial: "3", synthCount: 1},
				testElement{serial: "4", synthCount: 1},
				testElement{serial: "5", synthCount: 1},
				testElement{serial: "6", synthCount: 1},
				testElement{serial: "7", synthCount: 1},
				testElement{serial: "8", synthCount: 1},
				testElement{serial: "9", synthCount: 1},
			},
			output: []Element{
				Start{},
				testElement{serial: "1", synthCount: 1},
				testElement{serial: "2", synthCount: 1},
				testElement{serial: "3", synthCount: 1},
				testElement{serial: "4", synthCount: 1},
				testElement{serial: "5", synthCount: 1},
				testElement{serial: "6", synthCount: 1},
				testElement{serial: "7", synthCount: 1},
				testElement{serial: "8", synthCount: 1},
				testElement{serial: "9", synthCount: 1},
				End{},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			proc := newProcessor(func(previous, current, next Element) []Element {
				var (
					atStart bool
					atEnd   bool
					out     []Element
				)
				if _, ok := previous.(Start); ok {
					atStart = true
				}
				if _, ok := next.(End); ok {
					atEnd = true
				}
				if atStart {
					out = append(out, previous)
				}
				out = append(out, current)
				if atEnd {
					out = append(out, next)
				}
				return out
			}, testComparator)
			proc.Update(tc.input, nil, nil)
			got := proc.Synthesize()
			want := tc.output
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got != want\n\t got = %#v\n\twant = %#v\n", got, want)
			}
		})
	}
}
