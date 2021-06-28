// Package kitchen demonstrates the various chat components and features.
package main

import (
	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

func main() {
	go func() {
		var (
			w   = app.NewWindow(app.Size(unit.Dp(800), unit.Dp(600)))
			ops op.Ops
		)
		// Event loop executes indefinitely, until the app is signalled to quit.
		// Integrate external services here.
		for event := range w.Events() {
			switch event := event.(type) {
			case system.FrameEvent:
				layoutUI(layout.NewContext(&ops, event))
				event.Frame(&ops)
			}
		}
	}()
	// Surrender main thread to OS.
	// Necessary for certain platforms.
	app.Main()
}

// Type alias common layout types for legibility.
type (
	C = layout.Context
	D = layout.Dimensions
)

// th is the active theme object.
var th = material.NewTheme(gofont.Collection())

// layoutUI renders the user interface.
func layoutUI(gtx C) D {
	return layout.Center.Layout(gtx, func(gtx C) D {
		return material.H1(th, "Hello Chat!").Layout(gtx)
	})
}
