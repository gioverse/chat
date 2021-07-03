// Package kitchen demonstrates the various chat components and features.
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"
	"sync"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"git.sr.ht/~gioverse/chat/ninepatch"
	"git.sr.ht/~gioverse/chat/res"
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

type (
	C = layout.Context
	D = layout.Dimensions
)

// th is the active theme object.
var th = material.NewTheme(gofont.Collection())

var (
	input component.TextField
	once  sync.Once
	ninep = ninepatch.DecodeNinePatch(func() image.Image {
		data, err := res.Resources.Open("9-Patch/iap_hotdog_asset.png")
		if err != nil {
			panic(fmt.Errorf("opening 9-Patch image: %w", err))
		}
		defer data.Close()
		img, err := png.Decode(data)
		if err != nil {
			panic(fmt.Errorf("decoding 9-Patch image: %w", err))
		}
		return img
	}())
)

// layoutUI renders the user interface.
func layoutUI(gtx C) D {
	once.Do(func() {
		input.SetText(strings.TrimSpace(`
This is a 9-Patch layout.

As the message wraps, the patches expand appropriately.

Naturally, the corners are static, the horizontal gutters expand vertically, and the vertical gutters expand horizontally.

Now all we need to do is stretch images across the patches to create a 9-Patch themed message. 
		`))
	})
	return layout.UniformInset(unit.Dp(30)).Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(
			gtx,
			layout.Rigid(func(gtx C) D {
				return material.H3(th, "9-Patch").Layout(gtx)
			}),
			layout.Rigid(func(gtx C) D {
				return ninep.Layout(gtx, material.Body1(th, input.Text()).Layout)
			}),
			layout.Rigid(func(gtx C) D {
				return layout.Spacer{Height: unit.Dp(10)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx C) D {
				return input.Layout(gtx, th, "type a thing")
			}),
		)
	})
}
