package material

import (
	"image"
	"time"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// SeparatorStyle configures the presentation of the unread indicator.
type SeparatorStyle struct {
	Message    material.LabelStyle
	TextMargin layout.Inset
	LineMargin layout.Inset
	LineWidth  unit.Value
}

// UnreadSeparator fills in a SeparatorStyle with sensible defaults.
func UnreadSeparator(th *material.Theme) SeparatorStyle {
	us := SeparatorStyle{
		Message:    material.Body1(th, "New Messages"),
		TextMargin: layout.UniformInset(unit.Dp(8)),
		LineMargin: layout.UniformInset(unit.Dp(8)),
		LineWidth:  unit.Dp(2),
	}
	us.Message.Color = th.ContrastBg
	return us
}

// DateSeparator makes a SeparatorStyle with indicating the transition to
// the date provided in the time.Time.
func DateSeparator(th *material.Theme, date time.Time) SeparatorStyle {
	return SeparatorStyle{
		Message:    material.Body1(th, date.Format("Mon Jan 2, 2006")),
		TextMargin: layout.UniformInset(unit.Dp(8)),
		LineMargin: layout.UniformInset(unit.Dp(8)),
		LineWidth:  unit.Dp(2),
	}
}

// Layout the Separator.
func (u SeparatorStyle) Layout(gtx layout.Context) layout.Dimensions {
	layoutLine := func(gtx layout.Context) layout.Dimensions {
		return u.LineMargin.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			size := image.Point{
				X: gtx.Constraints.Max.X,
				Y: gtx.Px(u.LineWidth),
			}
			paint.FillShape(gtx.Ops, u.Message.Color, clip.Rect(image.Rectangle{Max: size}).Op())
			return layout.Dimensions{Size: size}
		})
	}
	return layout.Flex{
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Flexed(.5, layoutLine),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.TextMargin.Layout(gtx, u.Message.Layout)
		}),
		layout.Flexed(.5, layoutLine),
	)
}
