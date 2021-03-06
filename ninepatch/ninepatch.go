// Package ninepatch implements 9-Patch image rendering in Gio.
// https://developer.android.com/guide/topics/graphics/drawables#nine-patch
package ninepatch

import (
	"image"
	"math"
	"sync"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// NinePatch can lay out a 9-Patch image as the background for another widget.
//
// Note: create a new instance per 9-Patch image. Changing the image.Image
// after the first layout will have no effect because the paint.ImageOp is
// cached.
type NinePatch struct {
	// Image is the backing image of the 9-Patch.
	image.Image
	// Grid describes the stretchable regions of the 9-Patch.
	Grid Grid
	// Inset describes content insets defined by the black lines on the bottom
	// and right of the 9-Patch image.
	Content PxInset
	// Cache the image.
	cache paint.ImageOp
	once  sync.Once
}

// PxInset describes an inset in pixels.
type PxInset struct {
	Top, Bottom, Left, Right int
}

func (p PxInset) ToDp(m unit.Metric) layout.Inset {
	return layout.Inset{
		Top:    unit.Dp(float32(p.Top) * m.PxPerDp),
		Bottom: unit.Dp(float32(p.Bottom) * m.PxPerDp),
		Left:   unit.Dp(float32(p.Left) * m.PxPerDp),
		Right:  unit.Dp(float32(p.Right) * m.PxPerDp),
	}
}

// Patch describes the position and size of single patch in a 9-Patch image.
type Patch struct {
	Offset image.Point
	Size   image.Point
}

// Region describes how to lay out a particular patch of a 9-Patch image.
type Region struct {
	// Source is the patch relative to the source image.
	Source Patch
	// Stretched is the patch relative to the layout.
	Stretched Patch
}

// Layout the patch of the provided ImageOp described by the Region, scaling
// as needed.
func (r Region) Layout(gtx C, src paint.ImageOp) D {
	// Set the paint material to our source texture.
	src.Add(gtx.Ops)

	// If we need to scale the source image to cover the content area, do so:
	if r.Stretched.Size != r.Source.Size {
		defer op.Affine(f32.Affine2D{}.Scale(layout.FPt(r.Stretched.Offset), f32.Point{
			X: float32(r.Stretched.Size.X) / float32(r.Source.Size.X),
			Y: float32(r.Stretched.Size.Y) / float32(r.Source.Size.Y),
		})).Push(gtx.Ops).Pop()
	}

	// Shift layout to the origin of the region that we are covering, but compensate
	// for the fact that we're going to be reaching to an arbitrary point in the
	// source image. This logic aligns the origin of the important region of the
	// source image with the origin of the region that we're laying out.
	defer op.Offset(r.Stretched.Offset.Sub(r.Source.Offset)).Push(gtx.Ops).Pop()

	// Clip the scaled image to the bounds of the area we need to cover.
	defer clip.Rect(image.Rectangle{
		Min: r.Source.Offset,
		Max: r.Source.Size.Add(r.Source.Offset),
	}).Push(gtx.Ops).Pop()

	// Paint the scaled, clipped image.
	paint.PaintOp{}.Add(gtx.Ops)

	return D{Size: r.Stretched.Size}
}

// DefaultScale is a standard 72 DPI.
// Inverse of `widget.Image`, shrink as the screen becomes _less_ dense.
const DefaultScale = 1 / float32(160.0/72.0)

// Layout the provided widget with the NinePatch as a background.
func (n NinePatch) Layout(gtx C, w layout.Widget) D {
	n.once.Do(func() {
		n.cache = paint.NewImageOp(n.Image)
	})

	// TODO(jfm) [performance]: cache scaled grid instead of recomputing every
	// frame.

	// TODO(jfm): publicize scale factor in a way that is obvious to use and
	// tested.

	scale := DefaultScale

	// Handle screen density.
	scale *= gtx.Metric.PxPerDp

	var (
		src = n.Grid
		str = Grid{
			X1: int(math.Round(float64(src.X1) * float64(scale))),
			X2: int(math.Round(float64(src.X2) * float64(scale))),
			Y1: int(math.Round(float64(src.Y1) * float64(scale))),
			Y2: int(math.Round(float64(src.Y2) * float64(scale))),
		}
		inset = layout.Inset{
			Left:   unit.Dp(float32(n.Content.Left) * scale),
			Right:  unit.Dp(float32(n.Content.Right) * scale),
			Top:    unit.Dp(float32(n.Content.Top) * scale),
			Bottom: unit.Dp(float32(n.Content.Bottom) * scale),
		}
	)

	// Layout content in macro to compute it's dimensions.
	// These dimensions are needed to figure out how much stretch is needed.
	macro := op.Record(gtx.Ops)
	dims := inset.Layout(gtx, w)
	call := macro.Stop()

	str.Size = dims.Size

	// Handle tiny content: at least stretch by the amount that original does.
	if str.Stretch().Y <= src.Stretch().Y {
		dims.Size.Y = dims.Size.Y - str.Stretch().Y + src.Stretch().Y
		str.Size.Y = str.Size.Y - str.Stretch().Y + src.Stretch().Y
	}
	if str.Stretch().X <= src.Stretch().X {
		dims.Size.X = dims.Size.X - str.Stretch().X + src.Stretch().X
		str.Size.X = str.Size.X - str.Stretch().X + src.Stretch().X
	}

	// Layout each of the 9 patches.

	// upper left
	Region{
		Source: Patch{
			Size: image.Point{
				X: src.X1,
				Y: src.Y1,
			},
		},
		Stretched: Patch{
			Size: image.Point{
				X: str.X1,
				Y: str.Y1,
			},
		},
	}.Layout(gtx, n.cache)

	// upper middle
	Region{
		Source: Patch{
			Size: image.Point{
				X: src.Stretch().X,
				Y: src.Y1,
			},
			Offset: image.Point{
				X: src.X1,
			},
		},
		Stretched: Patch{
			Size: image.Point{
				X: str.Stretch().X,
				Y: str.Y1,
			},
			Offset: image.Point{
				X: str.X1,
			},
		},
	}.Layout(gtx, n.cache)

	// upper right
	Region{
		Source: Patch{
			Size: image.Point{
				X: src.X2,
				Y: src.Y1,
			},
			Offset: image.Point{
				X: src.X1 + src.Stretch().X,
			},
		},
		Stretched: Patch{
			Size: image.Point{
				X: str.X2,
				Y: str.Y1,
			},
			Offset: image.Point{
				X: str.X1 + str.Stretch().X,
			},
		},
	}.Layout(gtx, n.cache)

	// middle left
	Region{
		Source: Patch{
			Size: image.Point{
				X: src.X1,
				Y: src.Stretch().Y,
			},
			Offset: image.Point{
				Y: src.Y1,
			},
		},
		Stretched: Patch{
			Size: image.Point{
				X: str.X1,
				Y: str.Stretch().Y,
			},
			Offset: image.Point{
				Y: str.Y1,
			},
		},
	}.Layout(gtx, n.cache)

	// middle middle
	Region{
		Source: Patch{
			Size: image.Point{
				X: src.Stretch().X,
				Y: src.Stretch().Y,
			},
			Offset: image.Point{
				X: src.X1,
				Y: src.Y1,
			},
		},
		Stretched: Patch{
			Size: image.Point{
				X: str.Stretch().X,
				Y: str.Stretch().Y,
			},
			Offset: image.Point{
				X: str.X1,
				Y: str.Y1,
			},
		},
	}.Layout(gtx, n.cache)

	// middle right
	Region{
		Source: Patch{
			Size: image.Point{
				X: src.X2,
				Y: src.Stretch().Y,
			},
			Offset: image.Point{
				X: src.X1 + src.Stretch().X,
				Y: src.Y1,
			},
		},
		Stretched: Patch{
			Size: image.Point{
				X: str.X2,
				Y: str.Stretch().Y,
			},
			Offset: image.Point{
				X: str.X1 + str.Stretch().X,
				Y: str.Y1,
			},
		},
	}.Layout(gtx, n.cache)

	// lower left
	Region{
		Source: Patch{
			Size: image.Point{
				X: src.X1,
				Y: src.Y2,
			},
			Offset: image.Point{
				Y: src.Y1 + src.Stretch().Y,
			},
		},
		Stretched: Patch{
			Size: image.Point{
				X: str.X1,
				Y: str.Y2,
			},
			Offset: image.Point{
				Y: str.Y1 + str.Stretch().Y,
			},
		},
	}.Layout(gtx, n.cache)

	// lower middle
	Region{
		Source: Patch{
			Size: image.Point{
				X: src.Stretch().X,
				Y: src.Y2,
			},
			Offset: image.Point{
				X: src.X1,
				Y: src.Y1 + src.Stretch().Y,
			},
		},
		Stretched: Patch{
			Size: image.Point{
				X: str.Stretch().X,
				Y: str.Y2,
			},
			Offset: image.Point{
				X: str.X1,
				Y: str.Y1 + str.Stretch().Y,
			},
		},
	}.Layout(gtx, n.cache)

	// lower right
	Region{
		Source: Patch{
			Size: image.Point{
				X: src.X2,
				Y: src.Y2,
			},
			Offset: image.Point{
				Y: src.Y1 + src.Stretch().Y,
				X: src.X1 + src.Stretch().X,
			},
		},
		Stretched: Patch{
			Size: image.Point{
				X: str.X2,
				Y: str.Y2,
			},
			Offset: image.Point{
				Y: str.Y1 + str.Stretch().Y,
				X: str.X1 + str.Stretch().X,
			},
		},
	}.Layout(gtx, n.cache)

	call.Add(gtx.Ops)

	return dims
}
