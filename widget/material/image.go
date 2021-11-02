package material

import (
	"image"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
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
		gtx.Constraints.Max.X = gtx.Constraints.Constrain(image.Pt(gtx.Px(img.Width), 0)).X
	}
	if img.Height.V > 0 {
		gtx.Constraints.Max.Y = gtx.Constraints.Constrain(image.Pt(0, gtx.Px(img.Height))).Y
	}
	if img.Image.Src == (paint.ImageOp{}) {
		return D{Size: gtx.Constraints.Max}
	}
	macro := op.Record(gtx.Ops)
	dims := img.Image.Layout(gtx)
	call := macro.Stop()
	r := float32(gtx.Px(img.Radii))
	defer clip.RRect{
		Rect: f32.Rectangle{Max: layout.FPt(dims.Size)},
		NE:   r, NW: r, SE: r, SW: r,
	}.Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return dims
}
