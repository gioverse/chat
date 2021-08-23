package list

import (
	"reflect"
	"strings"
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
