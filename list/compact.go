package list

import "sort"

// Compact is a list of sorted Elements with a specific maximum size.
// It supports insertion, updating elements in place, and removing
// elements.
type Compact struct {
	elements []Element
	Size     int
	Comparator
}

// NewCompact returns a Compact with the given maximum size and
// using the given Comparator to sort its contents.
func NewCompact(size int, comp Comparator) *Compact {
	return &Compact{
		Size:       size,
		Comparator: comp,
	}
}

func (c *Compact) mapping() map[Serial]int {
	serialToRaw := make(map[Serial]int)
	for i, elem := range c.elements {
		serialToRaw[elem.Serial()] = i
	}
	return serialToRaw
}

// Apply inserts, updates, and removes elements from within the contents
// of the compact.
func (c *Compact) Apply(insertOrUpdate []Element, updateOnly []Element, remove []Serial) {
	serialToRaw := c.mapping()
	// Search newElems for elements that already exist within the Raw slice.
	SliceFilter(&insertOrUpdate, func(elem Element) bool {
		rawIndex, exists := serialToRaw[elem.Serial()]
		if exists {
			// Update the stored existing element.
			c.elements[rawIndex] = elem
			return false
		}
		return true
	})

	// Update elements if and only if they are present.
	for _, elem := range updateOnly {
		index, isPresent := serialToRaw[elem.Serial()]
		if !isPresent {
			continue
		}
		c.elements[index] = elem
	}

	// Find the index of each element needing removal.
	var targetIndicies []int
	for _, serial := range remove {
		idx, ok := serialToRaw[serial]
		if !ok {
			continue
		}
		targetIndicies = append(targetIndicies, idx)
	}
	// Remove them by swapping and re-slicing, starting from the highest
	// index to ensure that we do not move a removed element into the
	// middle of the list as part of the swap.
	sort.Sort(sort.Reverse(sort.IntSlice(targetIndicies)))
	for _, target := range targetIndicies {
		SliceRemove(&c.elements, target)
	}

	c.elements = append(c.elements, insertOrUpdate...)
	// Re-sort elements.
	sort.SliceStable(c.elements, func(i, j int) bool {
		return c.Comparator(c.elements[i], c.elements[j])
	})
}

// Compact returns a compacted slice of the elements managed by the Compact.
// The resulting elements are garanteed to be sorted using the
// Compact's Comparator and there will usually be no more than c.Size elements.
// The exception is when c.Size is smaller than 3 times the distance between
// keepStart and keepEnd. In that case, Compact will attempt to return a slice
// containing the region described by [keepStart,keepEnd] with the same number
// of elements on either side.
func (c *Compact) Compact(keepStart, keepEnd Serial) (contents []Element, compacted []Serial) {
	if len(c.elements) < 1 {
		return nil, nil
	}
	serialToRaw := c.mapping()
	keepStartIdx, ok := serialToRaw[keepStart]
	if !ok || keepStart == NoSerial {
		keepStartIdx = 0
	}
	keepEndIdx, ok := serialToRaw[keepEnd]
	if !ok || keepEnd == NoSerial {
		keepEndIdx = len(c.elements) - 1
	}
	visible := (1 + keepEndIdx - keepStartIdx)
	size := max(c.Size, 3*visible)
	additional := size - visible
	if additional > 0 {
		// cut the additional size in half, ensuring that no element is
		// lost to integer truncation.
		half := additional / 2
		secondHalf := additional - half
		if keepStartIdx < half {
			// Donate any unused quota at the beginning of the list to
			// the end.
			secondHalf += (half - keepStartIdx)
		}
		if newEnd := keepEndIdx + secondHalf; newEnd >= len(c.elements) {
			// Donate any unused quota at the end of the list to
			// the beginning.
			half += newEnd - (len(c.elements) - 1)
		}
		keepStartIdx = max(keepStartIdx-half, 0)
		keepEndIdx = min(keepEndIdx+secondHalf, len(c.elements)-1)
	}

	// Collect the serials of elements that are being deallocated by compaction.
	for i := 0; i < keepStartIdx; i++ {
		compacted = append(compacted, c.elements[i].Serial())
	}
	for i := keepEndIdx + 1; i < len(c.elements); i++ {
		compacted = append(compacted, c.elements[i].Serial())
	}

	// Allocate a new Raw slice to house the data, allowing the older,
	// longer slice to be garbage collected.
	newLength := keepEndIdx - keepStartIdx + 1
	newRaw := make([]Element, newLength)
	copy(newRaw, c.elements[keepStartIdx:keepEndIdx+1])
	c.elements = newRaw

	return c.elements, compacted
}
