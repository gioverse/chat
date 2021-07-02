package apptheme

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"git.sr.ht/~gioverse/chat/example/kitchen/model"
)

// SeparatorStyle configures the presentation of the unread indicator.
type SeparatorStyle struct {
	Message    material.LabelStyle
	TextMargin layout.Inset
	LineMargin layout.Inset
	LineWidth  unit.Value
}

// UnreadSeparator fills in a SeparatorStyle with sensible defaults.
func UnreadSeparator(th *material.Theme, ub model.UnreadBoundary) SeparatorStyle {
	us := SeparatorStyle{
		Message:    material.Body1(th, "New Messages"),
		TextMargin: layout.UniformInset(unit.Dp(8)),
		LineMargin: layout.UniformInset(unit.Dp(8)),
		LineWidth:  unit.Dp(2),
	}
	us.Message.Color = th.ContrastBg
	return us
}

// DateSeparator makes a SeparatorStyle with reasonable defaults out of
// the provided DateBoundary.
func DateSeparator(th *material.Theme, db model.DateBoundary) SeparatorStyle {
	return SeparatorStyle{
		Message:    material.Body1(th, db.Date.Format("Mon Jan 2, 2006")),
		TextMargin: layout.UniformInset(unit.Dp(8)),
		LineMargin: layout.UniformInset(unit.Dp(8)),
		LineWidth:  unit.Dp(2),
	}
}

// Layout the Separator.
func (u SeparatorStyle) Layout(gtx C) D {
	layoutLine := func(gtx C) D {
		return u.LineMargin.Layout(gtx, func(gtx C) D {
			size := image.Point{
				X: gtx.Constraints.Max.X,
				Y: gtx.Px(u.LineWidth),
			}
			paint.FillShape(gtx.Ops, u.Message.Color, clip.Rect(image.Rectangle{Max: size}).Op())
			return D{Size: size}
		})
	}
	return layout.Flex{
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Flexed(.5, layoutLine),
		layout.Rigid(func(gtx C) D {
			return u.TextMargin.Layout(gtx, u.Message.Layout)
		}),
		layout.Flexed(.5, layoutLine),
	)
}
