// Package kitchen demonstrates the various chat components and features.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "image/jpeg"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"git.sr.ht/~gioverse/chat"
	"git.sr.ht/~gioverse/chat/example/kitchen/appwidget"
	"git.sr.ht/~gioverse/chat/example/kitchen/appwidget/apptheme"
	"git.sr.ht/~gioverse/chat/example/kitchen/model"
	"git.sr.ht/~gioverse/chat/ninepatch"
	"git.sr.ht/~gioverse/chat/res"
	lorem "github.com/drhodes/golorem"
)

var (
	// max images to generate.
	max int
)

func init() {
	rand.Seed(time.Now().UnixNano())
	flag.IntVar(&max, "max", 100, "max images to generate (default 100)")
	flag.Parse()
}

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
	RowsList widget.List
	*chat.RowManager
	Bg color.NRGBA
	// Modal is layed above the content area with a scrim.
	Modal layout.Widget
	// Scrim clicks to dismiss the modal.
	Scrim widget.Clickable
}

// NewUI constructs a UI and populates it with dummy data.
func NewUI() *UI {
	var ui UI

	ui.RowsList.ScrollToEnd = true

	var (
		cookie = func() ninepatch.NinePatch {
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
		}()
		hotdog = func() ninepatch.NinePatch {
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
				state, ok := state.(*appwidget.Message)
				if !ok {
					return func(C) D { return D{} }
				}
				if state.Clicked() {
					ui.Modal = func(gtx C) D {
						return widget.Image{
							Src:      state.Image,
							Fit:      widget.ScaleDown,
							Position: layout.Center,
						}.Layout(gtx)
					}
				}
				msg := apptheme.NewMessage(th, state, data)
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
				Image: func() image.Image {
					if rand.Float32() < 0.7 {
						return nil
					}
					sizes := []image.Point{
						image.Pt(1792, 828),
						image.Pt(828, 1792),
						image.Pt(600, 600),
						image.Pt(300, 300),
					}
					img, err := randomImage(sizes[rand.Intn(len(sizes))])
					if err != nil {
						log.Print(err)
						return nil
					}
					return img
				}(),
				Avatar: func() image.Image {
					sizes := []image.Point{
						image.Pt(64, 64),
						image.Pt(32, 32),
						image.Pt(24, 24),
					}
					img, err := randomImage(sizes[rand.Intn(len(sizes))])
					if err != nil {
						log.Print(err)
						return nil
					}
					return img
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
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx C) D {
			gtx.Constraints.Min = gtx.Constraints.Max
			return material.List(th.Theme, &ui.RowsList).Layout(gtx, ui.RowManager.Len(), ui.RowManager.Layout)
		}),
		layout.Expanded(func(gtx C) D {
			return ui.layoutModal(gtx)
		}),
	)
}

func (ui *UI) layoutModal(gtx C) D {
	if ui.Scrim.Clicked() {
		ui.Modal = nil
	}
	if ui.Modal == nil {
		return D{}
	}
	return layout.Stack{}.Layout(
		gtx,
		layout.Stacked(func(gtx C) D {
			return material.Clickable(gtx, &ui.Scrim, func(gtx C) D {
				return component.Rect{
					Size:  gtx.Constraints.Max,
					Color: color.NRGBA{A: 250},
				}.Layout(gtx)
			})
		}),
		layout.Expanded(func(gtx C) D {
			return layout.UniformInset(unit.Dp(25)).Layout(gtx, func(gtx C) D {
				return layout.Center.Layout(gtx, func(gtx C) D {
					return ui.Modal(gtx)
				})
			})
		}),
	)
}

// randomImage returns a random image at the given size.
// Downloads some number of random images from unplash and caches them on disk.
//
// TODO(jfm) [performance]: download images concurrently (parallel downloads,
// async to the gui event loop).
func randomImage(sz image.Point) (image.Image, error) {
	cache := filepath.Join(os.TempDir(), "chat", fmt.Sprintf("%dx%d", sz.X, sz.Y))
	if err := os.MkdirAll(cache, 0755); err != nil {
		return nil, fmt.Errorf("preparing cache directory: %w", err)
	}
	entries, err := ioutil.ReadDir(cache)
	if err != nil {
		return nil, fmt.Errorf("reading cache entries: %w", err)
	}
	entries = filter(entries, isFile)
	if len(entries) == 0 {
		for ii := 0; ii < 10; ii++ {
			ii := ii
			if err := func() error {
				r, err := http.Get(fmt.Sprintf("https://source.unsplash.com/random/%dx%d", sz.X, sz.Y))
				if err != nil {
					return fmt.Errorf("fetching image data: %w", err)
				}
				defer r.Body.Close()
				imgf, err := os.Create(filepath.Join(cache, strconv.Itoa(ii)))
				if err != nil {
					return fmt.Errorf("creating image file on disk: %w", err)
				}
				defer imgf.Close()
				if _, err := io.Copy(imgf, r.Body); err != nil {
					return fmt.Errorf("downloading image: %w", err)
				}
				return nil
			}(); err != nil {
				return nil, fmt.Errorf("populating image cache: %w", err)
			}
		}
		return randomImage(sz)
	}
	selection := entries[rand.Intn(len(entries))]
	imgf, err := os.Open(filepath.Join(cache, selection.Name()))
	if err != nil {
		return nil, fmt.Errorf("opening image file: %w", err)
	}
	defer imgf.Close()
	img, _, err := image.Decode(imgf)
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}
	return img, nil
}

// isFile filters out non-file entries.
func isFile(info fs.FileInfo) bool {
	return !info.IsDir()
}

func filter(list []fs.FileInfo, predicate func(fs.FileInfo) bool) (filtered []fs.FileInfo) {
	for _, item := range list {
		if predicate(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
