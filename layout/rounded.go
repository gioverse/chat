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
	defer op.Save(gtx.Ops).Load()
	radii := float32(gtx.Px(unit.Value(r)))
	clip.RRect{
		Rect: layout.FRect(image.Rectangle{Max: dims.Size}),
		NE:   radii,
		NW:   radii,
		SW:   radii,
		SE:   radii,
	}.Add(gtx.Ops)
	call.Add(gtx.Ops)
	return dims
}
