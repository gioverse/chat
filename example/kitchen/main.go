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
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"git.sr.ht/~gioverse/chat/example/kitchen/appwidget/apptheme"
	"git.sr.ht/~gioverse/chat/profile"
)

var (
	// theme to use {light,dark}.
	theme string
	// usePlato to use plato themed widgets.
	usePlato bool
	// latency specifies whether to simulate latency.
	latency bool
	// profileOpt specifies what to profile.
	profileOpt string
)

// th is the active theme object.
var (
	fonts = gofont.Collection()
	th    = apptheme.NewTheme(fonts)
)

func init() {
	rand.Seed(time.Now().UnixNano())
	flag.StringVar(&theme, "theme", "light", "theme to use {light,dark}")
	flag.StringVar(&profileOpt, "profile", "none", "create the provided kind of profile. Use one of [none, cpu, mem, block, goroutine, mutex, trace, gio]")
	flag.BoolVar(&usePlato, "plato", false, "use Plato Team Inc themed widgets")
	flag.BoolVar(&latency, "latency", true, "whether to simulate network latency")
	flag.Parse()
	switch theme {
	case "light":
		th.UsePalette(apptheme.Light)
	case "dark":
		th.UsePalette(apptheme.Dark)
	}
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
		ui = NewUI(w)
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
