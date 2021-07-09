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

func testSynthesizer(previous, current Element) []Element {
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
			p.Update(tc.input...)
			processed := p.Synthesize()
			if !elementsEqual(processed, tc.output) {
				t.Errorf("Expected %v, got %v", tc.output, processed)
			}
		})
	}
}

// TestProcessorUpdates ensures that the processor correctly updates
// elements with that share serial values with elements that are already in
// the managed list.
func TestProcessorUpdates(t *testing.T) {
	p := newProcessor(testSynthesizer, testComparator)
	type testcase struct {
		name   string
		input  []Element
		output []Element
	}
	// Run all testcases on a single processor instance to check its behavior over time.
	for _, tc := range []testcase{
		{
			name: "empty produces empty",
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
			name: "one updated produces two",
			input: []Element{
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
			input: []Element{
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
			name: "update every element",
			input: []Element{
				testElement{
					serial:     "a",
					synthCount: 2,
				},
				testElement{
					serial:     "b",
					synthCount: 0,
				},
				testElement{
					serial:     "c",
					synthCount: 2,
				},
				testElement{
					serial:     "d",
					synthCount: 0,
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
					serial:     "c",
					synthCount: 2,
				},
				testElement{
					serial:     "c",
					synthCount: 2,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p.Update(tc.input...)
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
			p.Update(tc.input...)
			_ = p.Synthesize()
			compacted := p.Compact(tc.req.MaxSize, tc.req.Viewport)
			if !serialsEqual(compacted, tc.compacted) {
				t.Errorf("Expected %v, got %v", tc.compacted, compacted)
			}
		})
	}
}
