package list

import "testing"

func TestSliceRemove(t *testing.T) {
	type testcase struct {
		name   string
		data   []Element
		index  int
		result []Element
	}
	for _, tc := range []testcase{
		{
			name:   "empty slice",
			data:   []Element{},
			index:  0,
			result: []Element{},
		},
		{
			name:   "nil slice",
			data:   nil,
			index:  0,
			result: nil,
		},
		{
			name: "index out of bounds",
			data: []Element{
				testElement{},
			},
			index: 5,
			result: []Element{
				testElement{},
			},
		},
		{
			name: "single element slice",
			data: []Element{
				testElement{},
			},
			index:  0,
			result: []Element{},
		},
		{
			name: "two element slice (remove first)",
			data: []Element{
				testElement{serial: "a"},
				testElement{serial: "b"},
			},
			index: 0,
			result: []Element{
				testElement{serial: "b"},
			},
		},
		{
			name: "two element slice (remove last)",
			data: []Element{
				testElement{serial: "a"},
				testElement{serial: "b"},
			},
			index: 1,
			result: []Element{
				testElement{serial: "a"},
			},
		},
		{
			name: "three element slice (remove first)",
			data: []Element{
				testElement{serial: "a"},
				testElement{serial: "b"},
				testElement{serial: "c"},
			},
			index: 0,
			result: []Element{
				testElement{serial: "c"},
				testElement{serial: "b"},
			},
		},
		{
			name: "three element slice (remove middle)",
			data: []Element{
				testElement{serial: "a"},
				testElement{serial: "b"},
				testElement{serial: "c"},
			},
			index: 1,
			result: []Element{
				testElement{serial: "a"},
				testElement{serial: "c"},
			},
		},
		{
			name: "three element slice (remove last)",
			data: []Element{
				testElement{serial: "a"},
				testElement{serial: "b"},
				testElement{serial: "c"},
			},
			index: 2,
			result: []Element{
				testElement{serial: "a"},
				testElement{serial: "b"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			SliceRemove(&tc.data, tc.index)
			if !elementsEqual(tc.data, tc.result) {
				t.Errorf("expected %v, got %v", tc.result, tc.data)
			}
		})
	}
}

func TestSliceFilter(t *testing.T) {
	type testcase struct {
		name      string
		data      []Element
		predicate func(Element) bool
		result    []Element
	}
	for _, tc := range []testcase{
		{
			name:      "empty slice",
			data:      []Element{},
			predicate: func(_ Element) bool { return true },
			result:    []Element{},
		},
		{
			name:      "nil slice",
			data:      nil,
			predicate: func(_ Element) bool { return true },
			result:    nil,
		},
		{
			name: "nil predicate",
			data: []Element{
				testElement{},
			},
			predicate: nil,
			result: []Element{
				testElement{},
			},
		},
		{
			name: "single element slice remove none",
			data: []Element{
				testElement{},
			},
			predicate: func(_ Element) bool { return true },
			result: []Element{
				testElement{},
			},
		},
		{
			name: "single element slice remove all",
			data: []Element{
				testElement{},
			},
			predicate: func(_ Element) bool { return false },
			result:    []Element{},
		},
		{
			name: "two element slice (remove first)",
			data: []Element{
				testElement{serial: "a"},
				testElement{serial: "b"},
			},
			predicate: func(e Element) bool { return e.Serial() != "a" },
			result: []Element{
				testElement{serial: "b"},
			},
		},
		{
			name: "two element slice (remove last)",
			data: []Element{
				testElement{serial: "a"},
				testElement{serial: "b"},
			},
			predicate: func(e Element) bool { return e.Serial() != "b" },
			result: []Element{
				testElement{serial: "a"},
			},
		},
		{
			name: "three element slice (remove first)",
			data: []Element{
				testElement{serial: "a"},
				testElement{serial: "b"},
				testElement{serial: "c"},
			},
			predicate: func(e Element) bool { return e.Serial() != "a" },
			result: []Element{
				testElement{serial: "c"},
				testElement{serial: "b"},
			},
		},
		{
			name: "three element slice (remove middle)",
			data: []Element{
				testElement{serial: "a"},
				testElement{serial: "b"},
				testElement{serial: "c"},
			},
			predicate: func(e Element) bool { return e.Serial() != "b" },
			result: []Element{
				testElement{serial: "a"},
				testElement{serial: "c"},
			},
		},
		{
			name: "three element slice (remove last)",
			data: []Element{
				testElement{serial: "a"},
				testElement{serial: "b"},
				testElement{serial: "c"},
			},
			predicate: func(e Element) bool { return e.Serial() != "c" },
			result: []Element{
				testElement{serial: "a"},
				testElement{serial: "b"},
			},
		},
		{
			name: "three element slice (remove first two)",
			data: []Element{
				testElement{serial: "a"},
				testElement{serial: "b"},
				testElement{serial: "c"},
			},
			predicate: func(e Element) bool { return e.Serial() != "a" && e.Serial() != "b" },
			result: []Element{
				testElement{serial: "c"},
			},
		},
		{
			name: "three element slice (remove last two)",
			data: []Element{
				testElement{serial: "a"},
				testElement{serial: "b"},
				testElement{serial: "c"},
			},
			predicate: func(e Element) bool { return e.Serial() != "b" && e.Serial() != "c" },
			result: []Element{
				testElement{serial: "a"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			SliceFilter(&tc.data, tc.predicate)
			if !elementsEqual(tc.data, tc.result) {
				t.Errorf("expected %v, got %v", tc.result, tc.data)
			}
		})
	}
}
