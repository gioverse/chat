// SPDX-License-Identifier: Unlicense OR MIT

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"git.sr.ht/~gioverse/chat/async"
	"git.sr.ht/~gioverse/chat/profile"
	chatwidget "git.sr.ht/~gioverse/chat/widget"

	_ "image/jpeg"
	_ "image/png"
)

var (
	th = material.NewTheme(gofont.Collection())
	// noIO specifies to avoid downloading images or touching the disk.
	// Reduces blocking IO to let any blocking UI bubble up in the block profile.
	noIO bool
	// cache specifies whether to cache on disk or just hold images in memory.
	cache bool
	// purge the cache before layout.
	// Useful when you want to cache, but you want the cache to start empty.
	purge bool
	// profileOpt specifies what to profile.
	profileOpt string
	// cacheDir is the path to the cache.
	cacheDir = filepath.Join(os.TempDir(), "chat", "resources")
)

func init() {
	flag.BoolVar(&cache, "cache", true, "cache images to disk")
	flag.BoolVar(&purge, "purge", false, "purge the cache (deletes all entries)")
	flag.BoolVar(&noIO, "no-io", false, "run the loader without any IO")
	flag.StringVar(&profileOpt, "profile", "none", "create the provided kind of profile. Use one of [none, cpu, mem, block, goroutine, mutex, trace, gio]")
	flag.Parse()
	if purge {
		if err := os.RemoveAll(cacheDir); err != nil {
			log.Fatalf("purging cache: %v\n", err)
		}
	}
}

func main() {
	ui := NewUI()
	go func() {
		w := app.NewWindow(
			app.Title("Loader"),
		)
		if err := ui.Run(w); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}

// UI holds state for, and lays out, the UI.
type UI struct {
	// Loader api is designed to be useful as a zero value. So declaring it on
	// the UI is sufficient to start using it.
	async.Loader
	// reels executes the demonstration, laying out as many async widgets as
	// the viewport will allow.
	reels Reels
}

// NewUI allocates a UI with some number of reels.
func NewUI() UI {
	return UI{
		// MaxLoaded of '0' indicates to only keep state that is currently in
		// view.
		Loader: async.Loader{MaxLoaded: 0},
	}
}

// Run handles window events and renders the application.
func (ui *UI) Run(w *app.Window) error {
	profiler := profile.Opt(profileOpt).NewProfiler()
	profiler.Start()
	var ops op.Ops
	for {
		select {
		case <-ui.Loader.Updated():
			fmt.Printf("%#v\n", ui.Loader.Stats())
			w.Invalidate()
		case e := <-w.Events():
			switch e := e.(type) {
			case system.DestroyEvent:
				profiler.Stop()
				return e.Err
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)
				profiler.Record(gtx)
				ui.Layout(gtx)
				e.Frame(&ops)
			}
		}
	}
}

type (
	C = layout.Context
	D = layout.Dimensions
)

// Layout the UI. Notice the async.Loader wraps each frame. It does this in
// order to count frames and in-turn, detect stale data.
func (ui *UI) Layout(gtx C) D {
	return ui.Loader.Frame(gtx, func(gtx C) D {
		return ui.reels.Layout(gtx, &ui.Loader)
	})
}

// Reels lays out an infinitely growing list of Reel, thus forming a 2D grid of
// square widgets that fill the entire viewport. Reels are grown as the view
// expands.
type Reels struct {
	// items is a list of Reel too layout.
	items []Reel
	// list state.
	list widget.List
}

// Layout the reels vertically, performing async operations on the provided
// loader.
func (reels *Reels) Layout(gtx C, loader *async.Loader) D {
	reels.list.Axis = layout.Vertical
	return material.List(th, &reels.list).Layout(gtx, reels.Len(), func(gtx C, ii int) D {
		if ii == len(reels.items) {
			reels.Grow()
		}
		return reels.items[ii].Layout(gtx, loader)
	})
}

// Len reports the number of reels. Ensures at least one reel is available.
func (reels *Reels) Len() int {
	if len(reels.items) == 0 {
		reels.Grow()
	}
	return len(reels.items) + 1
}

// Grow a reel.
func (reels *Reels) Grow() {
	reels.items = append(reels.items, Reel{index: len(reels.items)})
}

// Reel lays out an infinitely growing number of square widgets in a scrollable
// list for demonstration. Reel grows as the view expands.
//
// index and count together form the ID of the Reel for `async.Loader` lookups,
// since the logical Reel is stateless (function of it's position in the grid).
type Reel struct {
	// index of this reel in the Reels list, for ID purposes.
	index int
	// count is the number of widgets in the Reel.
	count int
	// list state.
	list layout.List
	// cache the image ops mapped to reel item IDs.
	cache map[string]*chatwidget.CachedImage
}

// Len reports the number of widgets in the real. Ensures at least one widget
// is available.
func (reel *Reel) Len() int {
	if reel.count == 0 {
		reel.count++
	}
	return reel.count + 1
}

// Layout a Reel. Reel is represented by a colored square. Green when Queued,
// Red when Loading, and the downloaded image when Loaded.
func (reel *Reel) Layout(gtx C, loader *async.Loader) D {
	if reel.cache == nil {
		reel.cache = make(map[string]*chatwidget.CachedImage)
	}
	return reel.list.Layout(gtx, reel.Len(),
		func(gtx C, ii int) D {
			if ii == reel.count {
				reel.count++
			}
			return layout.UniformInset(unit.Dp(8)).Layout(gtx,
				func(gtx C) D {
					px := gtx.Px(unit.Dp(64))
					size := image.Point{X: px, Y: px}
					gtx.Constraints = layout.Exact(size)

					id := strconv.Itoa(reel.index) + ":" + strconv.Itoa(ii)

					// Schedule the resource.
					//
					// This returns an promise-like structure that can be in
					// various states: queued, loading and loaded.
					// If loaded, the contained value can be inspected and used.
					//
					// Calling schedule like this keeps the resource warm.
					// If this part of the layout falls out of view, the resource
					// becomes cold and gets garbage collected.
					//
					// You can leak the resource by taking a pointer to it or its
					// value to ensure it doesn't get garbage collected.
					r := loader.Schedule(id, func(ctx context.Context) interface{} {
						img, err := fetch(id, fmt.Sprintf(unsplash, 64, 64))
						if err != nil {
							log.Printf("error fetching image: %v", err)
							return nil
						}
						return img
					})

					// Switch on the various possible states of the resource.
					// Here we choose to layout colored squares, green for queued,
					// red for loading. When loaded we type assert the value for
					// and image, cache it and finally present it.
					switch r.State {
					case async.Queued:
						col := color.NRGBA{R: 0xFF, G: 0xC0, B: 0xC0, A: 0xFF}
						paint.FillShape(gtx.Ops, col, clip.Rect{Max: size}.Op())
						layout.Center.Layout(gtx, material.Body1(th, id).Layout)

					case async.Loading:
						col := color.NRGBA{R: 0xC0, G: 0xFF, B: 0xC0, A: 0xFF}
						paint.FillShape(gtx.Ops, col, clip.Rect{Max: size}.Op())
						layout.Center.Layout(gtx, material.Body1(th, id).Layout)

					case async.Loaded:
						col := color.NRGBA{R: 0xF0, G: 0xF0, B: 0xF0, A: 0xFF}
						paint.FillShape(gtx.Ops, col, clip.Rect{Max: size}.Op())

						m, ok := r.Value.(image.Image)
						if ok && m != nil {
							im, ok := reel.cache[id]
							if !ok {
								im = &chatwidget.CachedImage{}
								reel.cache[id] = im
							}
							im.Cache(m)
							layout.Center.Layout(gtx, func(gtx C) D {
								return widget.Image{
									Src: im.Op(),
									Fit: widget.Contain,
								}.Layout(gtx)
							})
						} else {
							layout.Center.Layout(gtx, material.Body1(th, id).Layout)
						}
					}

					return D{Size: size}
				})
		})
}

// unsplash endpoint that returns random nature images for the given dimensions.
const unsplash = "https://source.unsplash.com/random/%dx%d?nature"

// fetch image for the given id.
// Image is initially downloaded from the provided url and stored on disk.
// If flag `cache` is false, downloads will not be stored on disk.
func fetch(id, u string) (image.Image, error) {
	if noIO {
		return nil, nil
	}
	path := filepath.Join(cacheDir, id)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("preparing resource directory: %w", err)
	}
	var (
		src io.Reader
	)
	if cache {
		if info, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) || info.Size() == 0 {
			if info != nil && info.Size() == 0 {
				if err := os.Remove(path); err != nil {
					return nil, fmt.Errorf("removing corrupt image file: %w", err)
				}
			}
			if err := func() error {
				f, err := os.Create(path)
				if err != nil {
					return fmt.Errorf("creating resource file: %w", err)
				}
				defer f.Close()
				r, err := http.Get(u)
				if err != nil {
					return fmt.Errorf("GET: %w", err)
				}
				defer r.Body.Close()
				if r.StatusCode != http.StatusOK {
					return fmt.Errorf("GET: %s", r.Status)
				}
				if _, err := io.Copy(f, r.Body); err != nil {
					return fmt.Errorf("downloading resource to disk: %w", err)
				}
				return nil
			}(); err != nil {
				return nil, err
			}
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("opening resource file: %w", err)
		}
		defer f.Close()
		src = f
	} else {
		r, err := http.Get(u)
		if err != nil {
			return nil, fmt.Errorf("GET: %w", err)
		}
		defer r.Body.Close()
		if r.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GET: %s", r.Status)
		}
		src = r.Body
	}
	img, _, err := image.Decode(src)
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}
	return img, nil
}
