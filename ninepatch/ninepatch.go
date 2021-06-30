// Package ninepatch implements 9-Patch image rendering in Gio.
// https://developer.android.com/guide/topics/graphics/drawables#nine-patch
package ninepatch

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// TODO(jfm): how to capture scaling of an image dimensions?
// For example, creating an image at 100px square, then scaling by 2x, how to
// return the scaled dimensions.

type (
	C = layout.Context
	D = layout.Dimensions
)

// Rectangle is a 9-Patch themed rectangle container, that lays content in the
// content-area.
type Rectangle struct {
	Src paint.ImageOp
	// TODO(jfm): allow for arbitrary stretch areas.
	Areas []Area
}

type Area struct {
	image.Rectangle
	Stretch Stretch
}

type Stretch uint8

const (
	Static Stretch = iota
	Vertical
	Horizontal
	Full
)

// img is a generated image to test 9patch scaling.
var img paint.ImageOp = func() paint.ImageOp {
	var (
		sz  = image.Point{X: 9, Y: 9}
		img = image.NewNRGBA(image.Rectangle{Max: sz})
	)
	for xx := 0; xx < sz.X; xx++ {
		for yy := 0; yy < sz.Y; yy++ {
			if xx > 0 && xx%(sz.X/3) == 0 {
				img.Set(xx, yy, color.NRGBA{B: 200, A: 255})
			} else if yy > 0 && yy%(sz.Y/3) == 0 {
				img.Set(xx, yy, color.NRGBA{B: 200, A: 255})
			} else {
				img.Set(xx, yy, color.Black)
			}
		}
	}
	return paint.NewImageOp(img)
}()

// Layout content atop the 9-Patch themed rectangle.
//
// TODO(jfm):
// - decide how to specify stretch areas in code
// - layout stretch areas
// - decode stretch area data from 9-Patch image files (`9.png`)
func (r Rectangle) Layout(gtx C, w layout.Widget) D {
	/*
		1. stretch an arbitrary rectangle image over an arbitray widget. This forms
		the basis of the content area, which is the center area that stretches both
		axes as needed.
		2. place static rectangles at the corners of the content area
		3. place and stretch vertical rectangles
		4. place and stretch horizonta rectangles
	*/

	// Stretch the image across the dimensions of w.
	// This is how the content area of the 9-patch will stretch.
	m := op.Record(gtx.Ops)
	contentDims := w(gtx)
	content := m.Stop()
	defer content.Add(gtx.Ops)
	func() {
		defer op.Save(gtx.Ops).Load()
		// Given we can stretch an image across the content, we can likely
		// stretch a clip of an image all the same.
		//
		// So, this code now takes a 10x10 image, clips 1x1 pixel of it, then
		// stretches that 1x1 across then entire content area.
		//
		// This demonstrates how to stretch part of an image across an area.
		//
		// BUG: area is correct, but positing is not. Why?

		// Setup the scaling.
		op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Point{
			X: float32(contentDims.Size.X / 2),
			Y: float32(contentDims.Size.Y / 2),
		})).Add(gtx.Ops)

		// Select our clip to scale up.
		clip.Rect{
			Min: image.Point{X: 3, Y: 3},
			Max: image.Point{X: 5, Y: 5},
		}.Add(gtx.Ops)

		// Add the image data.
		img.Add(gtx.Ops)

		// Paint it all out.
		paint.PaintOp{}.Add(gtx.Ops)
	}()

	// Now place the static corners.
	// Then place and stretch the middles.

	// var (
	// 	sz = img.Size()
	// 	// third = sz.X / 3
	// 	// stretch = float32(2.0)
	// )
	// if len(r.Areas) == 0 {
	// 	paint.PaintOp{}.Add(gtx.Ops)
	// } else {
	// 	for _, a := range r.Areas {
	// 		func() {
	// 			defer op.Save(gtx.Ops).Load()
	// 			// clip.Rect(a.Rectangle).Add(gtx.Ops)
	// 			switch a.Stretch {
	// 			case Vertical:
	// 				widget.Image{Src: img, Fit: widget.Fill}.Layout(gtx)
	// 				op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Point{X: 1, Y: stretch})).Add(gtx.Ops)
	// 			case Horizontal:
	// 				op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Point{X: stretch, Y: 1})).Add(gtx.Ops)
	// 			case Full:
	// 				op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Point{X: stretch, Y: stretch})).Add(gtx.Ops)
	// 			}
	// 			paint.PaintOp{}.Add(gtx.Ops)
	// 		}()
	// 	}
	// }
	return D{Size: contentDims.Size}
}
