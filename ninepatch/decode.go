package ninepatch

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
)

// DecodeNinePatch from source image.
//
// Note: Any colored pixel around the border will be considered a 9-Patch marker.
func DecodeNinePatch(src image.Image) NinePatch {
	var (
		b      = src.Bounds()
		inset  = layout.Inset{}
		x1, x2 = 0, 0
		y1, y2 = 0, 0
	)
	right := walk(src, b.Max.X-1, layout.Vertical)
	if right.IsValid() {
		inset.Top = unit.Px(float32(right.Start))
		inset.Bottom = unit.Px(float32(b.Max.Y - right.End))
	}
	bottom := walk(src, b.Max.Y-1, layout.Horizontal)
	if bottom.IsValid() {
		inset.Left = unit.Px(float32(bottom.Start))
		inset.Right = unit.Px(float32(b.Max.X - bottom.End))
	}
	top := walk(src, 0, layout.Vertical)
	if top.IsValid() {
		y1, y2 = top.Start, b.Max.Y-top.End
	}
	left := walk(src, 0, layout.Horizontal)
	if left.IsValid() {
		x1, x2 = left.Start, b.Max.X-left.End
	}
	return NinePatch{
		Image:   eraseBorder(src),
		Content: inset,
		Grid: Grid{
			Size: image.Point{
				X: b.Dx(),
				Y: b.Dy(),
			},
			X1: x1, X2: x2,
			Y1: y1, Y2: y2,
		},
	}
}

// eraseBorder clears the 1px border around the image containing the 9-Patch
// region specifiers (1px black lines).
//
// TODO(jfm) [performance]: type switch src to see if we can mutate it directly
// and avoid copying it.
//
// The goal of loading a 9-Patch image is, at least ostensibly, to use the
// NinePatch type. It would be unexpected to then want that data for something
// else, post NinePatch allocation.
//
// However, if that were the case then mutating the src may be a bad idea.
//
// TODO(jfm) [performance]: current implemenation leaves 1px border of
// transparent pixels, which consumes memory for no gain.
func eraseBorder(src image.Image) *image.NRGBA {
	var (
		b   = src.Bounds()
		out = image.NewNRGBA(b)
	)
	// Copy image data.
	for xx := b.Min.X; xx < b.Max.X; xx++ {
		for yy := b.Min.Y; yy < b.Max.Y; yy++ {
			out.Set(xx, yy, src.At(xx, yy))
		}
	}
	// Clear out the borders which contain 1px 9-Patch stretch region
	// identifiers.
	for xx := b.Min.X; xx < b.Max.X; xx++ {
		out.Set(xx, b.Min.Y, color.NRGBA{})
		out.Set(xx, b.Max.Y-1, color.NRGBA{})
	}
	for yy := b.Min.Y; yy < b.Max.Y; yy++ {
		out.Set(b.Min.X, yy, color.NRGBA{})
		out.Set(b.Max.X-1, yy, color.NRGBA{})
	}
	return out
}

// line encodes a one-dimensional line.
type line struct {
	Start, End int
}

func (l line) IsValid() bool {
	return l.Start > -1 && l.End > -1
}

// walk pixels in the source image, along the specified main axis, and offset
// along the cross axis, returning a line that describes the length of any
// squence of colored pixels.
//
// NOTE(jfm): in time we may want tighter control over what is considered
// "colored". For now, any color that is not zero will suffice.
func walk(src image.Image, offset int, axis layout.Axis) line {
	var (
		end  = axis.Convert(src.Bounds().Max).X
		line = line{Start: -1, End: -1}
	)
	for ii := 0; ii < end; ii++ {
		pt := axis.Convert(image.Point{X: ii, Y: offset})
		r, g, b, a := src.At(pt.X, pt.Y).RGBA()
		var (
			colorIsSet = r > 0 || g > 0 || b > 0 || a > 0
			startIsSet = line.Start > -1
			endIsSet   = line.End > -1
		)
		if colorIsSet && !startIsSet {
			line.Start = ii
		}
		if !colorIsSet && startIsSet {
			line.End = ii
		}
		if startIsSet && endIsSet {
			break
		}
	}
	return line
}
