/*
Package debug provides tools for debugging Gio layout code.
*/
package debug

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// Outline traces a small black outline around the provided widget.
func Outline(gtx C, w func(gtx C) D) D {
	return widget.Border{
		Color: color.NRGBA{A: 255},
		Width: unit.Dp(1),
	}.Layout(gtx, w)
}
