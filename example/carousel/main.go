// SPDX-License-Identifier: Unlicense OR MIT

package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"git.sr.ht/~gioverse/chat/list"

	"gioui.org/font/gofont"
)

const defaultSize = 800

func main() {
	go func() {
		w := app.NewWindow(
			app.Size(unit.Dp(defaultSize), unit.Dp(defaultSize+10)),
			app.Title("Carousel"),
		)
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

// imageElement is a list element that displays a single image.
type imageElement struct {
	serial string
	img    image.Image
}

func (i imageElement) Serial() list.Serial {
	return list.Serial(i.serial)
}

// loader creates new elements in a given direction by pulling a new image from unsplash.com.
func loader(dir list.Direction, relativeTo list.Serial) []list.Element {
	var newSerial int
	if relativeTo == list.NoSerial {
		newSerial = 0
	} else {
		newSerial, _ = strconv.Atoi(string(relativeTo))
		switch dir {
		case list.Before:
			newSerial--
		case list.After:
			newSerial++
		}
	}
	r, err := http.Get(fmt.Sprintf("https://source.unsplash.com/random/%dx%d?nature", defaultSize, defaultSize))
	if err != nil {
		log.Printf("fetching image data: %v", err)
		return nil
	}
	defer r.Body.Close()
	img, _, err := image.Decode(r.Body)
	if err != nil {
		log.Printf("decoding image: %v", err)
		return nil
	}
	return []list.Element{
		imageElement{
			serial: fmt.Sprintf("%d", newSerial),
			img:    img,
		},
	}
}

// comparator returns whether a sorts before b.
func comparator(a, b list.Element) bool {
	asIntA, _ := strconv.Atoi(string(a.Serial()))
	asIntB, _ := strconv.Atoi(string(b.Serial()))
	return asIntA < asIntB
}

// allocator creates paint.PaintOps to hold image state for an element.
func allocator(e list.Element) interface{} {
	element := e.(imageElement)
	return paint.NewImageOp(element.img)
}

// present lays out the image contained within an element.
func present(e list.Element, state interface{}) layout.Widget {
	paintOp := state.(paint.ImageOp)
	return widget.Image{
		Fit: widget.Contain,
		Src: paintOp,
	}.Layout
}

// synthesizer transforms elements, though no transformation is necessary for this use-case.
func synthesizer(previous, current, next list.Element) []list.Element {
	return []list.Element{current}
}

func loop(w *app.Window) error {
	th := material.NewTheme(gofont.Collection())
	l := widget.List{}
	hooks := list.Hooks{
		Loader:      loader,
		Comparator:  comparator,
		Synthesizer: synthesizer,
		Allocator:   allocator,
		Presenter:   present,
		Invalidator: w.Invalidate,
	}
	m := list.NewManager(10, hooks)
	var ops op.Ops
	t := time.NewTicker(time.Second)
	var stats runtime.MemStats
	for {
		select {
		case e := <-w.Events():
			switch e := e.(type) {
			case system.DestroyEvent:
				return e.Err
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)
				material.List(th, &l).Layout(gtx, m.UpdatedLen(&l.List), m.Layout)
				e.Frame(gtx.Ops)
			}
		case <-t.C:
			runtime.ReadMemStats(&stats)
			log.Println("Bytes heap-allocated:", stats.Alloc)
		}
	}
}
