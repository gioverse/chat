/*
Package debug provides tools for debugging Gio layout code.
*/
package debug

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"os"
	"runtime"

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

// Caller returns the function nFrames above it on the call stack.
// Passing 3 as nFrames will return the details of the function
// invoking the function in which caller was invoked. This can help
// determine which of several code paths were taken to reach a
// particular place in the code.
func Caller(nFrames int) string {
	fpcs := make([]uintptr, 1)
	n := runtime.Callers(nFrames, fpcs)
	if n == 0 {
		return "NO CALLER"
	}

	caller := runtime.FuncForPC(fpcs[0] - 1)
	if caller == nil {
		return "MSG CALLER WAS NIL"
	}

	// Print the file name and line number
	file, line := caller.FileLine(fpcs[0] - 1)
	return fmt.Sprintf("%s:%d", file, line)
}
