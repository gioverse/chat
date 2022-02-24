// Package kitchen demonstrates the various chat components and features.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	_ "image/jpeg"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"git.sr.ht/~gioverse/chat/example/kitchen/ui"
	"git.sr.ht/~gioverse/chat/profile"
)

var (
	config ui.Config
	// profileOpt specifies what to profile.
	profileOpt string
)

func init() {
	rand.Seed(time.Now().UnixNano())
	flag.StringVar(&config.Theme, "theme", "light", "theme to use {light,dark}")
	flag.StringVar(&profileOpt, "profile", "none", "create the provided kind of profile. Use one of [none, cpu, mem, block, goroutine, mutex, trace, gio]")
	flag.BoolVar(&config.UsePlato, "plato", false, "use Plato Team Inc themed widgets")
	flag.IntVar(&config.Latency, "latency", 1000, "maximum latency (in millis) to simulate")
	flag.IntVar(&config.LoadSize, "load-size", 30, "number of items to load at a time")
	flag.IntVar(&config.BufferSize, "buffer-size", 30, "number of elements to hold in memory before compacting")

	flag.Parse()
}

type (
	C = layout.Context
	D = layout.Dimensions
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
		ui = ui.NewUI(w.Invalidate, config)
	)
	go func() {
		profiler := profile.Opt(profileOpt).NewProfiler()
		profiler.Start()
		// Event loop executes indefinitely, until the app is signalled to quit.
		// Integrate external services here.
		for {
			select {
			case <-ui.Loader.Updated():
				w.Invalidate()
			case event := <-w.Events():
				switch event := event.(type) {
				case system.DestroyEvent:
					profiler.Stop()
					if err := event.Err; err != nil {
						fmt.Printf("error: premature window close: %v\n", err)
						os.Exit(1)
					}
					os.Exit(0)
				case system.FrameEvent:
					gtx := layout.NewContext(&ops, event)
					profiler.Record(gtx)
					ui.Layout(gtx)
					event.Frame(&ops)
				}
			}
		}
	}()
	// Surrender main thread to OS.
	// Necessary for certain platforms.
	app.Main()
}
