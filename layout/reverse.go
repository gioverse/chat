package layout

import (
	"gioui.org/layout"
)

// Reverse the order of the provided flex children if the boolean is true.
func Reverse(shouldReverse bool, items ...layout.FlexChild) []layout.FlexChild {
	if len(items) == 0 {
		return items
	}
	if shouldReverse {
		for ii := 0; ii < len(items)/2; ii++ {
			var (
				head = ii
				tail = len(items) - 1 - ii
			)
			if head == tail {
				break
			}
			items[head], items[tail] = items[tail], items[head]
		}
		return items
	}
	return items
}
