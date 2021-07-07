package ninepatch

import "image"

// Grid describes the stretchable regions of a 9-Patch as 3x3 grid divided
// by 4 lines.
type Grid struct {
	// Size specifies the total dimensions including static and stretch regions.
	Size image.Point
	// X1 is the distance in pixels before the stretchable region along the X axis.
	// X2 is the distance in pixels after the stretchable region along the X axis.
	X1, X2 int
	// Y1 is the distance in pixels before the stretchable region along the Y axis.
	// Y2 is the distance in pixels after the stretchable region along the Y axis.
	Y1, Y2 int
}

// Static returns the statically known dimensions (the corners).
func (g Grid) Static() image.Point {
	return image.Point{
		X: g.X1 + g.X2,
		Y: g.Y1 + g.Y2,
	}
}

// Stretch returns the stretch dimensions (the space between the corners).
func (g Grid) Stretch() image.Point {
	stretch := g.Size.Sub(g.Static())
	if stretch.X < 0 {
		stretch.X = 0
	}
	if stretch.Y < 0 {
		stretch.Y = 0
	}
	return stretch
}
