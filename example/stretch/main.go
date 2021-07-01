package main

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
	"strings"
	"sync"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"git.sr.ht/~gioverse/chat/res"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// NinePatch can lay out a 9patch image as the background for another widget.
//
// TODO: create a constructor that automatically parses the Inset, X1, X2, Y1, and Y2
// from the source image.
type NinePatch struct {
	// Image is the backing image of the 9patch.
	image.Image
	// Inset encodes the mandatory content insets defined by the black lines on the
	// bottom and right of the 9patch image.
	layout.Inset
	// X1 is the distance in pixels before the stretchable region along the X axis
	// X2 is the distance in pixels after the stretchable region along the X axis
	X1, X2 int
	// Y1 is the distance in pixels before the stretchable region along the Y axis
	// Y2 is the distance in pixels after the stretchable region along the Y axis
	Y1, Y2 int
}

// NinePatchRegion describes how to lay out a particular region of a 9patch image.
// It defines an offset and size within the source image, and an offset and size
// within the layout. It provides a layout method that will handle converting
// between the provided offsets and sizes.
type NinePatchRegion struct {
	Size, Offset       image.Point
	SrcSize, SrcOffset image.Point
}

// Layout the region of the provided ImageOp described by the NinePatchRegion.
func (n NinePatchRegion) Layout(gtx C, src paint.ImageOp) D {
	defer op.Save(gtx.Ops).Load()
	// Shift layout to the origin of the region that we are covering, but compensate
	// for the fact that we're going to be reaching to an arbitrary point in the
	// source image. This logic aligns the origin of the important region of the
	// source image with the origin of the region that we're laying out.
	op.Offset(layout.FPt(n.Offset.Sub(n.SrcOffset))).Add(gtx.Ops)

	// Set the paint material to our source texture.
	src.Add(gtx.Ops)
	// If we need to scale the source image to cover the content area, do so:
	if n.Size != n.SrcSize {
		op.Affine(f32.Affine2D{}.Scale(layout.FPt(n.Offset), f32.Point{
			X: float32(n.Size.X) / float32(n.SrcSize.X),
			Y: float32(n.Size.Y) / float32(n.SrcSize.Y),
		})).Add(gtx.Ops)
	}
	// Clip the scaled image to the bounds of the area we need to cover.
	clip.Rect(image.Rectangle{
		Min: n.SrcOffset,
		Max: n.SrcSize.Add(n.SrcOffset),
	}).Add(gtx.Ops)
	// Paint the scaled, clipped image.
	paint.PaintOp{}.Add(gtx.Ops)

	return D{Size: n.Size}
}

// Layout the provided widget with the NinePatch as a background.
//
// TODO: eliminate the 1-pixel border that configures the insets
// and stretchable dimensions.
func (n NinePatch) Layout(gtx C, w layout.Widget) D {
	macro := op.Record(gtx.Ops)
	dims := n.Inset.Layout(gtx, w)
	call := macro.Stop()
	middleSrcWidth := n.Image.Bounds().Dx() - (n.X1 + n.X2)
	middleSrcHeight := n.Image.Bounds().Dy() - (n.Y1 + n.Y2)
	middleWidth := dims.Size.X - (n.X1 + n.X2)
	middleHeight := dims.Size.Y - (n.Y1 + n.Y2)

	imageOp := paint.NewImageOp(n.Image)

	upperLeft := NinePatchRegion{
		Size: image.Point{
			X: n.X1,
			Y: n.Y1,
		},
		SrcSize: image.Point{
			X: n.X1,
			Y: n.Y1,
		},
	}
	upperMiddle := NinePatchRegion{
		Offset: image.Point{
			X: n.X1,
		},
		Size: image.Point{
			X: middleWidth,
			Y: n.Y1,
		},
		SrcOffset: image.Point{
			X: n.X1,
		},
		SrcSize: image.Point{
			X: middleSrcWidth,
			Y: n.Y1,
		},
	}
	upperRight := NinePatchRegion{
		Offset: image.Point{
			X: n.X1 + middleWidth,
		},
		Size: image.Point{
			X: n.X2,
			Y: n.Y1,
		},
		SrcOffset: image.Point{
			X: n.X1 + middleSrcWidth,
		},
		SrcSize: image.Point{
			X: n.X2,
			Y: n.Y1,
		},
	}

	middleLeft := NinePatchRegion{
		Offset: image.Point{
			Y: n.Y1,
		},
		Size: image.Point{
			Y: middleHeight,
			X: n.X1,
		},
		SrcOffset: image.Point{
			Y: n.Y1,
		},
		SrcSize: image.Point{
			Y: middleSrcHeight,
			X: n.X1,
		},
	}
	middleMiddle := NinePatchRegion{
		Offset: image.Point{
			Y: n.Y1,
			X: n.X1,
		},
		Size: image.Point{
			Y: middleHeight,
			X: middleWidth,
		},
		SrcOffset: image.Point{
			Y: n.Y1,
			X: n.X1,
		},
		SrcSize: image.Point{
			Y: middleSrcHeight,
			X: middleSrcWidth,
		},
	}
	middleRight := NinePatchRegion{
		Offset: image.Point{
			Y: n.Y1,
			X: n.X1 + middleWidth,
		},
		Size: image.Point{
			Y: middleHeight,
			X: n.X2,
		},
		SrcOffset: image.Point{
			Y: n.Y1,
			X: n.X1 + middleSrcWidth,
		},
		SrcSize: image.Point{
			Y: middleSrcHeight,
			X: n.X2,
		},
	}

	bottomLeft := NinePatchRegion{
		Offset: image.Point{
			Y: n.Y1 + middleHeight,
		},
		Size: image.Point{
			Y: n.Y2,
			X: n.X1,
		},
		SrcOffset: image.Point{
			Y: n.Y1 + middleSrcHeight,
		},
		SrcSize: image.Point{
			Y: n.Y2,
			X: n.X1,
		},
	}
	bottomMiddle := NinePatchRegion{
		Offset: image.Point{
			Y: n.Y1 + middleHeight,
			X: n.X1,
		},
		Size: image.Point{
			Y: n.Y2,
			X: middleWidth,
		},
		SrcOffset: image.Point{
			Y: n.Y1 + middleSrcHeight,
			X: n.X1,
		},
		SrcSize: image.Point{
			Y: n.Y2,
			X: middleSrcWidth,
		},
	}
	bottomRight := NinePatchRegion{
		Offset: image.Point{
			Y: n.Y1 + middleHeight,
			X: n.X1 + middleWidth,
		},
		Size: image.Point{
			Y: n.Y2,
			X: n.X2,
		},
		SrcOffset: image.Point{
			Y: n.Y1 + middleSrcHeight,
			X: n.X1 + middleSrcWidth,
		},
		SrcSize: image.Point{
			Y: n.Y2,
			X: n.X2,
		},
	}

	upperLeft.Layout(gtx, imageOp)
	upperMiddle.Layout(gtx, imageOp)
	upperRight.Layout(gtx, imageOp)
	middleLeft.Layout(gtx, imageOp)
	middleMiddle.Layout(gtx, imageOp)
	middleRight.Layout(gtx, imageOp)
	bottomLeft.Layout(gtx, imageOp)
	bottomMiddle.Layout(gtx, imageOp)
	bottomRight.Layout(gtx, imageOp)

	call.Add(gtx.Ops)

	return dims
}

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

// th is the active theme object.
var th = material.NewTheme(gofont.Collection())

var (
	input component.TextField
	once  sync.Once
	// Hard-code the correct configuration for a specific 9patch image.
	ninep = NinePatch{
		Inset: layout.Inset{
			Top:    unit.Px(31),
			Bottom: unit.Px(28),
			Left:   unit.Px(70),
			Right:  unit.Px(71),
		},
		X1: 86,
		X2: 62,
		Y1: 55,
		Y2: 48,
		Image: func() image.Image {
			data, _ := res.Resources.Open("9-Patch/iap_hotdog_asset.png")
			defer data.Close()
			image, _, _ := image.Decode(data)
			return image
		}(),
	}
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
