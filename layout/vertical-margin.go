package layout

import (
	"gioui.org/layout"
	"gioui.org/unit"
)

// VerticalMarginStyle provides a simple API for insetting a widget equally
// on its top and bottom edges. Consistently wrapping chat elements
// in a single VerticalMarginStyle as their outermost layout type will
// ensure that they are spaced evenly and no part of their content
// crowds that of the message above and below.
type VerticalMarginStyle struct {
	Size unit.Dp
}

// VerticalMargin configures a vertical margin with a sensible default
// margin.
func VerticalMargin() VerticalMarginStyle {
	return VerticalMarginStyle{
		Size: unit.Dp(4),
	}
}

// Layout the provided widget within the margin and return their combined
// dimensions.
func (v VerticalMarginStyle) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	return layout.Inset{
		Top:    v.Size,
		Bottom: v.Size,
	}.Layout(gtx, w)
}
