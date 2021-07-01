// Package ninepatch implements 9-Patch image rendering in Gio.
// https://developer.android.com/guide/topics/graphics/drawables#nine-patch
package ninepatch

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// Layout is a 9-Patch themed rectangle container, that lays content in the
// content-area.
type Layout struct {
	// Patches maps arbitrary widgets to a 3x3 grid of patches.
	// Widgets are given the appropriate constraints for the patch.
	Patches [3][3]layout.Widget
	// CornerSize is the square dimension of the static corners.
	CornerSize int
	// VerticalGutterHeight defines the static height of the vertical
	// gutter patches.
	VerticalGutterHeight int
	// HorizontalGutterWidth defines the static width of the horizontal
	// gutter patches.
	HorizontalGutterWidth int
}

// Layout content atop the 9-Patch themed rectangle.
//
// 9-Patch layout follows these rules:
// - corners are static (patches 1, 3, 7, 9)
// - vertical gutters exand horizontally (patches 2, 8)
// - horizontal gutters exand vertically (patches 4, 6)
// - content expands both axes sufficient to accomodate the widget content (patch 5)
//
// Patches are mapped to a 3x3 grid which forms a left to right, top to bottom
// representation of the patches.
//
// Patches can be arbitrary widgets, however their relevant static constraints
// are currently specified by the `Static`, `HorCross`, `VerCross` fields.
//
// Therefore, you must know how big the static dimensions should be a-priory.
//
// TODO(jfm):
// - decode stretch area data from 9-Patch image files (`9.png`)
// - iterate on API
func (r Layout) Layout(gtx C, w layout.Widget) D {
	return layout.Flex{Axis: layout.Vertical}.Layout(
		gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(
				gtx,
				layout.Rigid(func(gtx C) D {
					gtx.Constraints.Max = image.Point{X: r.CornerSize, Y: r.CornerSize}
					gtx.Constraints.Min = image.Point{X: r.CornerSize, Y: r.CornerSize}
					return r.Patches[0][0](gtx)
				}),
				layout.Flexed(1, func(gtx C) D {
					gtx.Constraints.Min.Y = r.VerticalGutterHeight
					return r.Patches[0][1](gtx)
				}),
				layout.Rigid(func(gtx C) D {
					gtx.Constraints.Max = image.Point{X: r.CornerSize, Y: r.CornerSize}
					gtx.Constraints.Min = image.Point{X: r.CornerSize, Y: r.CornerSize}
					return r.Patches[0][2](gtx)
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return r.content(gtx, w)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(
				gtx,
				layout.Rigid(func(gtx C) D {
					gtx.Constraints.Max = image.Point{X: r.CornerSize, Y: r.CornerSize}
					gtx.Constraints.Min = image.Point{X: r.CornerSize, Y: r.CornerSize}
					return r.Patches[2][0](gtx)
				}),
				layout.Flexed(1, func(gtx C) D {
					gtx.Constraints.Min.Y = r.VerticalGutterHeight
					return r.Patches[2][1](gtx)
				}),
				layout.Rigid(func(gtx C) D {
					gtx.Constraints.Max = image.Point{X: r.CornerSize, Y: r.CornerSize}
					gtx.Constraints.Min = image.Point{X: r.CornerSize, Y: r.CornerSize}
					return r.Patches[2][2](gtx)
				}),
			)
		}),
	)
}

// content lays the content which wraps vertically. Gutters expand vertically.
func (r Layout) content(gtx C, w layout.Widget) D {
	// Shave off gutters when laying the content.
	macro := op.Record(gtx.Ops)
	gtx.Constraints.Max.X = gtx.Constraints.Max.X - r.HorizontalGutterWidth - r.HorizontalGutterWidth
	dims := w(gtx)
	content := macro.Stop()
	gtx.Constraints.Max.X = gtx.Constraints.Max.X + r.HorizontalGutterWidth + r.HorizontalGutterWidth
	return layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(
		gtx,
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Max.X = r.HorizontalGutterWidth
			gtx.Constraints.Min.X = r.HorizontalGutterWidth
			gtx.Constraints.Min.Y = dims.Size.Y
			return r.Patches[1][0](gtx)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Stack{}.Layout(
				gtx,
				layout.Expanded(func(gtx C) D {
					return r.Patches[1][1](gtx)
				}),
				layout.Stacked(func(gtx C) D {
					content.Add(gtx.Ops)
					return dims
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Max.X = r.HorizontalGutterWidth
			gtx.Constraints.Min.X = r.HorizontalGutterWidth
			gtx.Constraints.Min.Y = dims.Size.Y
			return r.Patches[1][2](gtx)
		}),
	)
}
