// Package kitchen demonstrates the various chat components and features.
package main

import (
	"fmt"
	"image/png"
	"os"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"git.sr.ht/~jackmordaunt/chat/ninepatch"
)

func main() {
	var (
		w = app.NewWindow(
			app.Title("Chat"),
			app.Size(unit.Dp(800), unit.Dp(600)),
		)
		ops op.Ops
	)
	go func() {
		// Event loop executes indefinitely, until the app is signalled to quit.
		// Integrate external services here.
		for event := range w.Events() {
			switch event := event.(type) {
			case system.DestroyEvent:
				if err := event.Err; err != nil {
					fmt.Printf("error: premature window close: %v\n", err)
					os.Exit(1)
				}
				os.Exit(0)
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

// patch is an example 9-Patch png for demonstration.
var patch = (func() paint.ImageOp {
	f, err := os.Open("res/9-patches/iap_blaster_asset.9.png")
	if err != nil {
		panic(fmt.Errorf("opening patch image file: %w", err))
	}
	defer f.Close()
	m, err := png.Decode(f)
	if err != nil {
		panic(fmt.Errorf("decoding pgn: %w", err))
	}
	return paint.NewImageOp(m)
})()

// layoutUI renders the user interface.
func layoutUI(gtx C) D {
	return layout.Center.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, layout.Rigid(func(gtx C) D {
			return ninepatch.Rectangle{Src: patch}.Layout(gtx, func(gtx C) D {
				return material.Label(th, unit.Dp(24), "lorem ipsum").Layout(gtx)
			})
		}))
	})
}
