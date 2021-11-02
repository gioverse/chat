package layout

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
)

// Rounded lays out a widget with rounded corners.
type Rounded unit.Value

func (r Rounded) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := w(gtx)
	call := macro.Stop()
	radii := float32(gtx.Px(unit.Value(r)))
	defer clip.RRect{
		Rect: layout.FRect(image.Rectangle{Max: dims.Size}),
		NE:   radii,
		NW:   radii,
		SW:   radii,
		SE:   radii,
	}.Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return dims
}
