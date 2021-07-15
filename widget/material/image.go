package material

import (
	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"
)

// Image lays out an image with optionally rounded corners.
type Image struct {
	widget.Image
	widget.Clickable
	// Radii specifies the amount of rounding.
	Radii unit.Value
	// Width and Height specify respective dimensions.
	// If left empty, dimensions will be unconstrained.
	Width, Height unit.Value
}

// Layout the image.
func (img Image) Layout(gtx layout.Context) layout.Dimensions {
	if img.Width.V > 0 {
		gtx.Constraints.Max.X = gtx.Px(img.Width)
	}
	if img.Height.V > 0 {
		gtx.Constraints.Max.Y = gtx.Px(img.Height)
	}
	defer op.Save(gtx.Ops).Load()
	macro := op.Record(gtx.Ops)
	dims := img.Image.Layout(gtx)
	call := macro.Stop()
	r := float32(gtx.Px(img.Radii))
	clip.RRect{
		Rect: f32.Rectangle{Max: layout.FPt(dims.Size)},
		NE:   r, NW: r, SE: r, SW: r,
	}.Add(gtx.Ops)
	call.Add(gtx.Ops)
	return dims
}
