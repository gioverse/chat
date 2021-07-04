// Package example is a playground for toying with and showcasing 9-Patch.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"git.sr.ht/~gioverse/chat/example/kitchen/appwidget/apptheme"
	"git.sr.ht/~gioverse/chat/ninepatch"
	lorem "github.com/drhodes/golorem"
)

func main() {
	var (
		// Instantiate the chat window.
		w = app.NewWindow(
			app.Title("9-Patch"),
			app.Size(unit.Dp(800), unit.Dp(600)),
		)
		// Define an operation list for gio.
		ops op.Ops
		// Instantiate our UI state.
		ui = NewUI()
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
				ui.Layout(layout.NewContext(&ops, event))
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
var (
	fonts = gofont.Collection()
	th    = apptheme.NewTheme(fonts)
)

// UI manages the state for the entire application's UI.
type UI struct {
	// Toggles patch visibility.
	Toggles []struct {
		Name string
		widget.Bool
	}
	// Patches available to the UI.
	Patches map[string]MessageStyle
	// Visible patches to render (by name).
	Visible []string
	// Width controls content width.
	Width widget.Float
	// Height controls content height.
	Height widget.Float
	// Real captures the real pixel dimensions of the content.
	Real image.Point
}

// NewUI constructs a UI and populates it with dummy data.
func NewUI() *UI {
	return &UI{
		Patches: map[string]MessageStyle{
			"platocookie": {
				Content: material.Body1(th.Theme, lorem.Sentence(5, 20)),
				Surface: func() ninepatch.NinePatch {
					imgf, err := os.Open("res/9-Patch/iap_platocookie_asset_2.png")
					if err != nil {
						panic(fmt.Errorf("opening image: %w", err))
					}
					defer imgf.Close()
					img, err := png.Decode(imgf)
					if err != nil {
						panic(fmt.Errorf("decoding png: %w", err))
					}
					return ninepatch.DecodeNinePatch(img)
				}(),
			},
			"hotdog": {
				Content: material.Body1(th.Theme, lorem.Sentence(5, 20)),
				Surface: func() ninepatch.NinePatch {
					imgf, err := os.Open("res/9-Patch/iap_hotdog_asset.png")
					if err != nil {
						panic(fmt.Errorf("opening image: %w", err))
					}
					defer imgf.Close()
					img, err := png.Decode(imgf)
					if err != nil {
						panic(fmt.Errorf("decoding png: %w", err))
					}
					return ninepatch.DecodeNinePatch(img)
				}(),
			},
		},
		Toggles: []struct {
			Name string
			widget.Bool
		}{
			{Name: "platocookie", Bool: widget.Bool{Value: true}},
			{Name: "hotdog", Bool: widget.Bool{Value: true}},
		},
		Visible: make([]string, 2),
	}
}

// Layout the application UI.
func (ui *UI) Layout(gtx C) D {
	for ii := range ui.Toggles {
		t := ui.Toggles[ii]
		if t.Bool.Value {
			ui.Visible[ii] = t.Name
		} else {
			ui.Visible[ii] = ""
		}
	}
	return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx C) D {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Middle,
		}.Layout(
			gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Center.Layout(gtx, func(gtx C) D {
					return material.H3(th.Theme, "9-Patch Demo").Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx C) D {
				var items []layout.FlexChild
				for ii := range ui.Toggles {
					toggle := &ui.Toggles[ii]
					items = append(items, layout.Rigid(func(gtx C) D {
						return material.CheckBox(th.Theme, &toggle.Bool, toggle.Name).Layout(gtx)
					}))
				}
				return layout.Flex{
					Axis:      layout.Horizontal,
					Alignment: layout.Middle,
					Spacing:   layout.SpaceSides,
				}.Layout(gtx, items...)
			}),
			layout.Rigid(func(gtx C) D {
				return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical}.Layout(
						gtx,
						layout.Rigid(func(gtx C) D {
							px := unit.Px(float32(ui.Real.X))
							dp := unit.Dp(px.V / gtx.Metric.PxPerDp)
							return LabeledSliderStyle{
								Label:  material.Body1(th.Theme, fmt.Sprintf("Content Width: %s (%s)", px, dp)),
								Slider: material.Slider(th.Theme, &ui.Width, 0, 1),
							}.Layout(gtx)
						}),
						layout.Rigid(func(gtx C) D {
							px := unit.Px(float32(ui.Real.Y))
							dp := unit.Dp(px.V / gtx.Metric.PxPerDp)
							return LabeledSliderStyle{
								Label:  material.Body1(th.Theme, fmt.Sprintf("Content Height: %s (%s)", px, dp)),
								Slider: material.Slider(th.Theme, &ui.Height, 0, 1),
							}.Layout(gtx)
						}),
					)
				})
			}),
			layout.Flexed(1, func(gtx C) D {
				return layout.Center.Layout(gtx, func(gtx C) D {
					var items []layout.FlexChild
					for ii := range ui.Visible {
						patch, ok := ui.Patches[ui.Visible[ii]]
						if !ok {
							continue
						}
						items = append(items, layout.Rigid(func(gtx C) D {
							gtx.Constraints.Min.X = int(ui.Width.Value * float32(gtx.Constraints.Max.X))
							gtx.Constraints.Min.Y = int(ui.Height.Value * float32(gtx.Constraints.Max.Y))
							ui.Real = gtx.Constraints.Min
							return patch.Layout(gtx)
						}))
					}
					return layout.Flex{
						Axis:      layout.Vertical,
						Alignment: layout.Middle,
					}.Layout(gtx, items...)
				})
			}),
		)
	})
}

// MessageStyle draws a message atop a ninepatch surface.
type MessageStyle struct {
	Content material.LabelStyle
	Surface ninepatch.NinePatch
}

func (msg MessageStyle) Layout(gtx C) D {
	cs := &gtx.Constraints
	return msg.Surface.Layout(gtx, func(gtx C) D {
		return RectStyle{
			Size:  cs.Min,
			Color: color.NRGBA{G: 200, A: 200},
		}.Layout(gtx)
	})
	// return msg.Surface.Layout(gtx, func(gtx C) D {
	// 	return msg.Content.Layout(gtx)
	// })
}

// LabeledSliderStyle draws a slider with a label.
type LabeledSliderStyle struct {
	Label  material.LabelStyle
	Slider material.SliderStyle
}

func (slider LabeledSliderStyle) Layout(gtx C) D {
	return layout.Flex{Axis: layout.Vertical}.Layout(
		gtx,
		layout.Rigid(slider.Label.Layout),
		layout.Rigid(slider.Slider.Layout),
	)
}

// RectStyle draws a colored rectangle.
type RectStyle struct {
	Color color.NRGBA
	Size  image.Point
	Radii float32
}

func (r RectStyle) Layout(gtx C) D {
	paint.FillShape(
		gtx.Ops,
		r.Color,
		clip.UniformRRect(
			f32.Rectangle{
				Max: layout.FPt(r.Size),
			},
			r.Radii,
		).Op(gtx.Ops))
	return layout.Dimensions{Size: image.Pt(r.Size.X, r.Size.Y)}
}
