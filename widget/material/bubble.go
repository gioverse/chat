package material

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// BubbleStyle defines a colored surface with (optionally) rounded corners.
type BubbleStyle struct {
	// The radius of the corners of the surface.
	// Non-rounded rectangles can just provide a zero.
	CornerRadius unit.Dp
	Color        color.NRGBA
}

// Bubble creates a Bubble style for the provided theme with the theme
// background color and rounded corners.
func Bubble(th *material.Theme) BubbleStyle {
	return BubbleStyle{
		CornerRadius: unit.Dp(12),
		Color:        th.Bg,
	}
}

// Layout renders the BubbleStyle, beneath the provided widget.
func (c BubbleStyle) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			surface := clip.UniformRRect(image.Rectangle{
				Max: gtx.Constraints.Min,
			}, gtx.Dp(c.CornerRadius))
			paint.FillShape(gtx.Ops, c.Color, surface.Op(gtx.Ops))
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(w),
	)
}
