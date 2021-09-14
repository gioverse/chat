/*
Package debug provides tools for debugging Gio layout code.
*/
package debug

import (
	"bytes"
	"encoding/json"
	"image/color"
	"io"
	"os"

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

// Dump logs the input as formatting JSON on stderr.
func Dump(v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	b = append(b, []byte("\n")...)
	io.Copy(os.Stderr, bytes.NewBuffer(b))
}
