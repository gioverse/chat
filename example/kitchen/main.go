// Package kitchen demonstrates the various chat components and features.
package main

import (
	"fmt"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"git.sr.ht/~gioverse/chat"
	"git.sr.ht/~gioverse/chat/example/kitchen/appwidget"
	"git.sr.ht/~gioverse/chat/example/kitchen/appwidget/apptheme"
	"git.sr.ht/~gioverse/chat/example/kitchen/model"
	"git.sr.ht/~gioverse/chat/ninepatch"
	lorem "github.com/drhodes/golorem"
)

func main() {
	var (
		// Instantiate the chat window.
		w = app.NewWindow(
			app.Title("Chat"),
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
	RowsList layout.List
	*chat.RowManager
	Bg color.NRGBA
}

// NewUI constructs a UI and populates it with dummy data.
func NewUI() *UI {
	var ui UI

	ui.RowsList.ScrollToEnd = true

	var (
		cookie = func() ninepatch.NinePatch {
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
		}()
		hotdog = func() ninepatch.NinePatch {
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
		}()
	)

	ui.RowManager = chat.NewManager(
		// Define an allocator function that can instaniate the appropriate
		// state type for each kind of row data in our list.
		func(data chat.Row) interface{} {
			switch data.(type) {
			case model.Message:
				return &appwidget.Message{}
			default:
				return nil
			}
		},
		// Define a presenter that can transform each kind of row data
		// and state into a widget.
		func(data chat.Row, state interface{}) layout.Widget {
			switch data := data.(type) {
			case model.Message:
				msg := apptheme.NewMessage(th, state.(*appwidget.Message), data)
				switch data.Theme {
				case "hotdog":
					msg = msg.WithNinePatch(th, hotdog)
				case "cookie":
					msg = msg.WithNinePatch(th, cookie)
				}
				return msg.Layout
			case model.DateBoundary:
				return apptheme.DateSeparator(th.Theme, data).Layout
			case model.UnreadBoundary:
				return apptheme.UnreadSeparator(th.Theme, data).Layout
			default:
				return func(gtx C) D { return D{} }
			}
		})

	// Configure a pleasing light gray background color.
	ui.Bg = color.NRGBA{220, 220, 220, 255}

	// Populate the UI with dummy random messages.
	max := 100
	for i := 0; i < max; i++ {
		var rowData chat.Row
		if i%10 == 0 {
			rowData = model.DateBoundary{Date: time.Now().Add(time.Hour * 24 * time.Duration(-(100 - i)))}
		} else if i == max-3 {
			rowData = model.UnreadBoundary{}
		} else {
			rowData = model.Message{
				Serial:  fmt.Sprintf("%d", i),
				Content: lorem.Paragraph(1, 5),
				SentAt:  time.Now().Add(time.Minute * time.Duration(-(100 - i))),
				Sender:  lorem.Word(3, 10),
				Local:   i%2 == 0,
				Status: func() string {
					if rand.Int()%10 == 0 {
						return apptheme.FailedToSend
					}
					return ""
				}(),
				Theme: func() string {
					switch val := rand.Intn(10); val {
					case 0:
						return "cookie"
					case 1:
						return "hotdog"
					default:
						return ""
					}
				}(),
			}
		}
		ui.RowManager.Rows = append(ui.RowManager.Rows, rowData)
	}

	return &ui
}

// Layout the application UI.
func (ui *UI) Layout(gtx C) D {
	paint.Fill(gtx.Ops, ui.Bg)
	ui.RowsList.Axis = layout.Vertical
	return ui.RowsList.Layout(gtx, ui.RowManager.Len(), ui.RowManager.Layout)
}
