// Package example is a playground for toying with and showcasing 9-Patch.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"git.sr.ht/~gioverse/chat/example/kitchen/appwidget/apptheme"
	"git.sr.ht/~gioverse/chat/ninepatch"
	"git.sr.ht/~gioverse/chat/res"
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
	// Messages available to the UI.
	Messages map[string]*FauxMessage
	// Visible messages to render (by name).
	Visible []string
	// Content controls the content dimensions.
	Content struct {
		Width  widget.Float
		Height widget.Float
	}
	// Constraints controls the constraints to render the 9-Patch with.
	Constraints struct {
		X widget.Float
		Y widget.Float
	}
	// TextContent controls whether to simulate text content.
	TextContent widget.Bool
	// TextAmount controls the amount of text to display.
	TextAmount widget.Float
	// PxPerDp controls the px-dp ratio to simulate different screen densities.
	PxPerDp widget.Float
	// ControlContainer adds scrolling to the controls.
	ControlContainer widget.List
	// DemoContainer adds scrolling to the demo.
	DemoContainer widget.List
}

// NewUI constructs a UI and populates it with dummy data.
func NewUI() *UI {
	return &UI{
		Messages: map[string]*FauxMessage{
			"platocookie": {
				Text: lorem.Sentence(1, 5),
				Surface: func() ninepatch.NinePatch {
					imgf, err := res.Resources.Open("9-Patch/iap_platocookie_asset_2.png")
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
				Text: lorem.Sentence(1, 5),
				Surface: func() ninepatch.NinePatch {
					imgf, err := res.Resources.Open("9-Patch/iap_hotdog_asset.png")
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
			{Name: "hotdog", Bool: widget.Bool{Value: false}},
		},
		Visible: make([]string, 2),
		Constraints: struct {
			X widget.Float
			Y widget.Float
		}{
			X: widget.Float{Value: 250},
			Y: widget.Float{Value: 250},
		},
		TextContent: widget.Bool{Value: true},
		PxPerDp:     widget.Float{Value: 1.0},
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
	if ui.TextAmount.Changed() {
		for _, msg := range ui.Messages {
			words := int(ui.TextAmount.Value)
			msg.Text = lorem.Sentence(words, words)
		}
	}
	if ui.Content.Width.Value > ui.Constraints.X.Value {
		ui.Content.Width.Value = ui.Constraints.X.Value
	}
	if ui.Content.Height.Value > ui.Constraints.Y.Value {
		ui.Content.Height.Value = ui.Constraints.Y.Value
	}
	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(ui.layoutTitle),
		layout.Rigid(ui.layoutContent),
	)
}

func (ui *UI) layoutTitle(gtx C) D {
	return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx C) D {
		return layout.Center.Layout(gtx, func(gtx C) D {
			return material.H4(th.Theme, "9-Patch Demo").Layout(gtx)
		})
	})
}

// breakpoint specifies dp at which to switch layout from horizontal to vertical.
// 450 is hopefully sane.
const breakpoint = 450

// layoutContent lays out the controls and demo area.
//
// It uses a horizontal layout for large constraints and a vertical layout
// for small constraints.
func (ui *UI) layoutContent(gtx C) D {
	var (
		axis = layout.Vertical
	)
	if gtx.Constraints.Max.X > gtx.Px(unit.Dp(breakpoint)) {
		axis = layout.Horizontal
	}
	return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx C) D {
		return layout.Flex{
			Axis: axis,
		}.Layout(gtx,
			layout.Flexed(1, func(gtx C) D {
				gtx.Constraints.Max.X = gtx.Px(unit.Dp(400))
				ui.ControlContainer.Axis = layout.Vertical
				return material.List(th.Theme, &ui.ControlContainer).Layout(gtx, 1, func(gtx C, _ int) D {
					return layout.E.Layout(gtx, func(gtx C) D {
						return layout.UniformInset(unit.Dp(20)).Layout(gtx, ui.layoutControls)
					})
				})
			}),
			layout.Rigid(func(gtx C) D {
				if axis == layout.Horizontal {
					return D{}
				}
				return layout.Inset{
					Top:    unit.Dp(10),
					Bottom: unit.Dp(10),
				}.Layout(gtx, func(gtx C) D {
					return component.Divider(th.Theme).Layout(gtx)
				})
			}),
			layout.Flexed(1, func(gtx C) D {
				ui.DemoContainer.Axis = layout.Vertical
				return material.List(th.Theme, &ui.DemoContainer).Layout(gtx, 1, func(gtx C, _ int) D {
					return layout.Center.Layout(gtx, func(gtx C) D {
						return layout.UniformInset(unit.Dp(20)).Layout(gtx, ui.layoutDemo)
					})
				})
			}),
		)
	})
}

func (ui *UI) layoutControls(gtx C) D {
	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(
		gtx,
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
		// Layout constraint sliders.
		layout.Rigid(func(gtx C) D {
			px, dp := DP(gtx.Metric.PxPerDp, ui.Constraints.X.Value)
			return LabeledSliderStyle{
				Label:  material.Body1(th.Theme, fmt.Sprintf("X Constraint: %s (%s)", px, dp)),
				Slider: material.Slider(th.Theme, &ui.Constraints.X, 0, 700),
			}.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			px, dp := DP(gtx.Metric.PxPerDp, ui.Constraints.Y.Value)
			return LabeledSliderStyle{
				Label:  material.Body1(th.Theme, fmt.Sprintf("Y Constraint: %s (%s)", px, dp)),
				Slider: material.Slider(th.Theme, &ui.Constraints.Y, 0, 700),
			}.Layout(gtx)
		}),
		// Layout content sliders.
		layout.Rigid(func(gtx C) D {
			px, dp := DP(gtx.Metric.PxPerDp, ui.Content.Width.Value)
			return LabeledSliderStyle{
				Label:  material.Body1(th.Theme, fmt.Sprintf("Content Width: %s (%s)", px, dp)),
				Slider: material.Slider(th.Theme, &ui.Content.Width, 0, 700),
			}.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			px, dp := DP(gtx.Metric.PxPerDp, ui.Content.Height.Value)
			return LabeledSliderStyle{
				Label:  material.Body1(th.Theme, fmt.Sprintf("Content Height: %s (%s)", px, dp)),
				Slider: material.Slider(th.Theme, &ui.Content.Height, 0, 700),
			}.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			return LabeledSliderStyle{
				Label:  material.Body1(th.Theme, fmt.Sprintf("PxPerDp: %.2f (default: %.2f)", ui.PxPerDp.Value, gtx.Metric.PxPerDp)),
				Slider: material.Slider(th.Theme, &ui.PxPerDp, 0.3, 20),
			}.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			return LabeledSliderStyle{
				Label:  material.Body1(th.Theme, "Text Amount"),
				Slider: material.Slider(th.Theme, &ui.TextAmount, 0, 700),
			}.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
			}.Layout(
				gtx,
				layout.Rigid(func(gtx C) D {
					return material.Body1(th.Theme, "Show Text").Layout(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					return D{Size: image.Point{X: gtx.Px(unit.Dp(10))}}
				}),
				layout.Rigid(func(gtx C) D {
					return material.Switch(th.Theme, &ui.TextContent, "Show Text").Layout(gtx)
				}),
			)
		}),
	)
}

func (ui *UI) layoutDemo(gtx C) D {
	var items []layout.FlexChild
	for ii := range ui.Visible {
		msg, ok := ui.Messages[ui.Visible[ii]]
		if !ok {
			continue
		}
		items = append(items, layout.Flexed(1, func(gtx C) D {
			cs := &gtx.Constraints
			cs.Max.X = int(ui.Constraints.X.Value)
			cs.Max.Y = int(ui.Constraints.Y.Value)
			return widget.Border{
				Color: color.NRGBA{A: 200},
				Width: unit.Dp(1),
			}.Layout(gtx, func(gtx C) D {
				return layout.Stack{}.Layout(
					gtx,
					layout.Stacked(func(gtx C) D {
						return D{Size: gtx.Constraints.Max}
					}),
					layout.Expanded(func(gtx C) D {
						gtx.Constraints.Min.X = int(ui.Content.Width.Value)
						gtx.Constraints.Min.Y = int(ui.Content.Height.Value)
						gtx.Metric.PxPerDp = ui.PxPerDp.Value
						return NewMessage(th.Theme, msg, &ui.TextContent).Layout(gtx)
					}),
				)
			})
		}))
	}
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Middle,
	}.Layout(gtx, items...)
}

// FauxMessage contains state needed to layout a fake message.
type FauxMessage struct {
	Text    string
	Surface ninepatch.NinePatch
}

// MessageStyle lays a message content atop a ninepatch surface.
type MessageStyle struct {
	*FauxMessage
	Content layout.Widget
}

// NewMessage constructs a MessageStyle.
func NewMessage(th *material.Theme, msg *FauxMessage, showText *widget.Bool) MessageStyle {
	content := func(gtx C) D {
		lb := material.Body1(th, msg.Text)
		lb.Color = th.ContrastFg
		return lb.Layout(gtx)
	}
	if !showText.Value {
		content = func(gtx C) D {
			return component.Rect{
				Color: color.NRGBA{G: 200, A: 200},
				Size:  gtx.Constraints.Min,
			}.Layout(gtx)
		}
	}
	return MessageStyle{
		FauxMessage: msg,
		Content:     content,
	}
}

func (msg MessageStyle) Layout(gtx C) D {
	return msg.Surface.Layout(gtx, func(gtx C) D {
		return msg.Content(gtx)
	})
}

// LabeledSliderStyle draws a slider with a label.
type LabeledSliderStyle struct {
	Label  material.LabelStyle
	Slider material.SliderStyle
}

func (slider LabeledSliderStyle) Layout(gtx C) D {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(
		gtx,
		layout.Rigid(slider.Label.Layout),
		layout.Rigid(slider.Slider.Layout),
	)
}

// DP helper computes the dp given some pixels and the ratio of pixels per dp.
//
// Note: This helper is for display purposes. Pixels are rounded for clarity,
// therefore do not use results as "real" units in layout.
func DP(pixelperdp float32, pixels float32) (px unit.Value, dp unit.Value) {
	pixels = float32(math.Round(float64(pixels)))
	return unit.Px(pixels), unit.Dp(pixels / pixelperdp)
}
