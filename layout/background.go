package layout

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/x/component"
)

// Background lays out a widget over a colored background.
type Background color.NRGBA

func (bg Background) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := w(gtx)
	call := macro.Stop()
	return layout.Stack{}.Layout(
		gtx,
		layout.Expanded(component.Rect{
			Size:  dims.Size,
			Color: color.NRGBA(bg),
		}.Layout),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			call.Add(gtx.Ops)
			return dims
		}),
	)
}
