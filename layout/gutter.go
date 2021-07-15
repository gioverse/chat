package layout

import (
	"gioui.org/layout"
	"gioui.org/unit"
)

// GutterStyle configures a gutter on either side of a horizontal row of content.
// Both sides can optionally display a widget atop the gutter space.
type GutterStyle struct {
	LeftWidth  unit.Value
	RightWidth unit.Value
	layout.Alignment
}

// Gutter returns a GutterStyle with a narrow left gutter and a wide right gutter.
func Gutter() GutterStyle {
	return GutterStyle{
		LeftWidth:  unit.Dp(12),
		RightWidth: unit.Dp(60),
		Alignment:  layout.Middle,
	}
}

// Layout the gutter with the left and right widgets laid out atop the gutter areas
// and the center widget in the remaining space in between. Left or right may be
// provided as nil to indicate that nothing should be displayed in the gutter.
func (g GutterStyle) Layout(gtx layout.Context, left, center, right layout.Widget) layout.Dimensions {
	return layout.Flex{
		Alignment: g.Alignment,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layoutGutterSide(gtx, g.LeftWidth, left)
		}),
		layout.Flexed(1, center),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layoutGutterSide(gtx, g.RightWidth, right)
		}),
	)
}

// layoutGutterSide lays out a spacer with a given width, and stacks another
// widget on top.
func layoutGutterSide(gtx layout.Context, width unit.Value, widget layout.Widget) layout.Dimensions {
	spacer := layout.Spacer{
		Width: width,
	}
	if widget == nil {
		return spacer.Layout(gtx)
	}
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			d := layout.Spacer{
				Width: width,
			}.Layout(gtx)
			return d
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			d := widget(gtx)
			return d
		}),
	)
}
