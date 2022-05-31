package material

import (
	"image"

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
	Radii unit.Dp
	// Width and Height specify respective dimensions.
	// If left empty, dimensions will be unconstrained.
	Width, Height unit.Dp
}

// Layout the image.
func (img Image) Layout(gtx layout.Context) layout.Dimensions {
	if img.Width > 0 {
		gtx.Constraints.Max.X = gtx.Constraints.Constrain(image.Pt(gtx.Dp(img.Width), 0)).X
	}
	if img.Height > 0 {
		gtx.Constraints.Max.Y = gtx.Constraints.Constrain(image.Pt(0, gtx.Dp(img.Height))).Y
	}
	if img.Image.Src == (paint.ImageOp{}) {
		return D{Size: gtx.Constraints.Max}
	}
	macro := op.Record(gtx.Ops)
	dims := img.Image.Layout(gtx)
	call := macro.Stop()
	r := gtx.Dp(img.Radii)
	defer clip.RRect{
		Rect: image.Rectangle{Max: dims.Size},
		NE:   r, NW: r, SE: r, SW: r,
	}.Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return dims
}
