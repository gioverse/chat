// SPDX-License-Identifier: Unlicense OR MIT

package main

/*
WARNING

This example exists to display the default behavior of an
unconfigured list.Manager. All real applications will need
to supply a value for each of the hooks in list.Hooks, so
this is not a good example for how to correctly use the
list.Manager type. For actually good examples look at the
other programs in the example folder.
*/

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"git.sr.ht/~gioverse/chat/list"

	"gioui.org/font/gofont"
)

func main() {
	go func() {
		w := app.NewWindow()
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func loop(w *app.Window) error {
	th := material.NewTheme(gofont.Collection())
	l := widget.List{List: layout.List{Axis: layout.Vertical}}
	m := list.NewManager(100, list.DefaultHooks(w, th))
	var ops op.Ops
	for {
		e := <-w.Events()
		switch e := e.(type) {
		case system.DestroyEvent:
			return e.Err
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)
			material.List(th, &l).Layout(gtx, m.UpdatedLen(&l.List), m.Layout)
			e.Frame(gtx.Ops)
		}
	}
}
