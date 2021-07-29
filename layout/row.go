package layout

import (
	"gioui.org/layout"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// Row lays out a central widget with gutters either side.
// The central widget can be arbitrarily aligned and gutters can have
// supplimentary widgets stacked atop them.
type Row struct {
	// Margin between rows.
	Margin VerticalMarginStyle
	// Padding around the central widget.
	Padding VerticalMarginStyle
	// Gutter handles the left-right gutters of the row that provides padding and
	// can contain other widgets.
	Gutter GutterStyle
	// Direction of widgets within this row.
	// Typically, non-local widgets are aligned W, and local widgets aligned E.
	Direction layout.Direction
}

// RowChild specifies a content widget and two gutter widgets either side.
// RowChild is used to layout composite rows made up of any number of interal
// rows.
type RowChild struct {
	Left    layout.Widget
	Content layout.Widget
	Right   layout.Widget
}

// FullRow returns a RowChild that lays out content with optional gutter widgets
// either side.
func FullRow(l, w, r layout.Widget) RowChild {
	return RowChild{
		Left:    l,
		Content: w,
		Right:   r,
	}
}

// ContentRow returns a RowChild that lays out a content with no gutter widgets.
func ContentRow(w layout.Widget) RowChild {
	return RowChild{Content: w}
}

// Layout the Row with any number of internal rows.
func (r *Row) Layout(gtx C, w ...RowChild) D {
	if r.Margin == (VerticalMarginStyle{}) {
		r.Margin = VerticalMargin()
	}
	if r.Padding == (VerticalMarginStyle{}) {
		r.Padding = VerticalMargin()
	}
	var fl = make([]layout.FlexChild, len(w))
	for ii := range w {
		ii := ii
		fl[ii] = layout.Rigid(func(gtx C) D {
			return r.Gutter.Layout(gtx,
				w[ii].Left,
				func(gtx C) D {
					return r.Direction.Layout(gtx, func(gtx C) D {
						return r.Padding.Layout(gtx, func(gtx C) D {
							if w[ii].Content == nil {
								return D{}
							}
							return w[ii].Content(gtx)
						})
					})
				},
				w[ii].Right,
			)
		})
	}
	return r.Margin.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, fl...)
	})
}
