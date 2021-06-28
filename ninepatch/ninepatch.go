// Package ninepatch implements 9-Patch image rendering in Gio.
// https://developer.android.com/guide/topics/graphics/drawables#nine-patch
package ninepatch

import (
	"gioui.org/layout"
	"gioui.org/op/paint"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// Rectangle is a 9-Patch themed rectangle container, that lays content in the
// content-area.
type Rectangle struct {
	Src paint.ImageOp
}

// Layout content atop the 9-Patch themed rectangle.
func (r Rectangle) Layout(gtx C, w layout.Widget) D {
	return layout.Stack{}.Layout(
		gtx,
		layout.Expanded(func(gtx C) D {
			r.Src.Add(gtx.Ops)
			return D{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(func(gtx C) D {
			return w(gtx)
		}),
	)
}
