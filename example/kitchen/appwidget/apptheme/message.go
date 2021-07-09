package apptheme

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/richtext"
	"git.sr.ht/~gioverse/chat/example/kitchen/appwidget"
	"git.sr.ht/~gioverse/chat/example/kitchen/model"
	"git.sr.ht/~gioverse/chat/ninepatch"
	matchat "git.sr.ht/~gioverse/chat/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

// ErrorIcon is the material design outlined error indicator.
var ErrorIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.AlertErrorOutline)
	return icon
}()

// FailedToSend is the message that is displayed to the user when there was a
// problem sending a chat message.
const FailedToSend = "Sending failed"

type (
	C = layout.Context
	D = layout.Dimensions
)

// MessageStyle configures the presentation of a chat message within
// a vertical list of chat messages.
type MessageStyle struct {
	// Local indicates that the message was sent by the local user,
	// and should be left-aligned.
	Local bool
	// Time is the timestamp associated with the message.
	Time material.LabelStyle
	// StatusIcon is an optional icon that will be displayed to the right of
	// the message instead of its timestamp.
	StatusIcon *widget.Icon
	// IconSize defines the size of the StatusIcon (if it is set).
	IconSize unit.Value
	// RightGutterPadding defines the size of the area to the right of the message
	// reserved for the timestamp and/or icon.
	RightGutterPadding layout.Inset
	// StatusMessage defines a warning message to be displayed beneath the
	// chat message.
	StatusMessage material.LabelStyle
	// Surface specifies the background surface of the chat message, typically
	// a chat bubble.
	Surface interface {
		Layout(gtx C, w layout.Widget) D
	}
	// ContentMargin configures space around the chat bubble.
	ContentMargin layout.Inset
	// Image specifies optional image content for the message.
	Image widget.Image
	// Content configures the actual contents of the chat bubble.
	Content richtext.TextStyle
	// ContentPadding defines space around the Content within the Bubble area.
	ContentPadding layout.Inset
	// LeftGutter defines the size of the empty left gutter of the row.
	LeftGutter layout.Spacer
	// Sender is the name of the sender of the message.
	Sender material.LabelStyle
	// MaxMessageWidth constrains messages width-wise.
	// Excess content will wrap vertically.
	MaxMessageWidth unit.Value
	// MaxImageHeight constrains images height-wise.
	// Images will be scaled-down to fit, preserving aspect ratio.
	MaxImageHeight unit.Value
	// Clicks tracks clicks over the message area.
	Clicks *widget.Clickable
}

// NewMessage creates a style type that can lay out the data for a message.
func NewMessage(th *Theme, interact *appwidget.Message, msg model.Message) MessageStyle {
	var (
		hasImage = msg.Image != nil
		isCached = interact.Image != (paint.ImageOp{})
	)
	if hasImage && !isCached {
		interact.Image = paint.NewImageOp(msg.Image)
	}
	bubble := matchat.Bubble(th.Theme)
	ms := MessageStyle{
		Time:    material.Body2(th.Theme, msg.SentAt.Local().Format("15:04")),
		Surface: &bubble,
		Content: richtext.Text(&interact.InteractiveText, th.Shaper, richtext.SpanStyle{
			Font:    th.Fonts[0].Font,
			Size:    material.Body1(th.Theme, "").TextSize,
			Color:   th.Fg,
			Content: msg.Content,
		}),
		Local:              msg.Local,
		IconSize:           unit.Dp(32),
		RightGutterPadding: layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12)},
		ContentPadding:     layout.UniformInset(unit.Dp(8)),
		ContentMargin:      layout.UniformInset(unit.Dp(8)),
		LeftGutter:         layout.Spacer{Width: unit.Dp(24)},
		Sender:             material.Body1(th.Theme, msg.Sender),
		Image: widget.Image{
			Src:      interact.Image,
			Fit:      widget.ScaleDown,
			Position: layout.Center,
		},
		MaxMessageWidth: th.MaxMessageWidth,
		MaxImageHeight:  th.MaxImageHeight,
		Clicks:          &interact.Clickable,
	}
	if msg.Status != "" {
		ms.StatusMessage = material.Body2(th.Theme, msg.Status)
		ms.StatusMessage.Color = th.DangerColor
		ms.StatusIcon = ErrorIcon
		ms.StatusIcon.Color = th.DangerColor
	}
	if !ms.Local {
		userColors := th.UserColor(msg.Sender)
		bubble.Color = userColors.NRGBA
		if userColors.Luminance < .5 {
			for i := range ms.Content.Styles {
				ms.Content.Styles[i].Color = th.Theme.Bg
			}
		}
	}
	return ms
}

// WithNinePatch sets the message surface to a ninepatch image.
func (c MessageStyle) WithNinePatch(th *Theme, np ninepatch.NinePatch) MessageStyle {
	c.Surface = np
	c.ContentMargin = layout.Inset{}
	var (
		b = np.Image.Bounds()
	)
	// TODO(jfm): refine into more robust solution for picking the text color,
	// as needed.
	//
	// Currently, we pick the middle pixel and use a heuristic formula to get
	// relative luminance.
	//
	// Only considers color.NRGBA colors.
	if cl, ok := np.Image.At(b.Dx()/2, b.Dy()/2).(color.NRGBA); ok {
		if Luminance(cl) < 0.5 {
			for i := range c.Content.Styles {
				c.Content.Styles[i].Color = th.Theme.Bg
			}
		}
	}
	return c
}

// Layout the message.
func (c MessageStyle) Layout(gtx C) D {
	messageAlignment := layout.W
	if c.Local {
		messageAlignment = layout.E
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Flex{
				Alignment: layout.Middle,
			}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return D{Size: image.Point{
						X: gtx.Px(c.LeftGutter.Width) +
							gtx.Px(c.ContentMargin.Left),
					}}
				}),
				layout.Flexed(1, func(gtx C) D {
					return messageAlignment.Layout(gtx, func(gtx C) D {
						return c.Sender.Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx C) D {
					return D{Size: image.Point{
						X: gtx.Px(c.IconSize) +
							gtx.Px(c.RightGutterPadding.Left) +
							gtx.Px(c.RightGutterPadding.Right) +
							gtx.Px(c.ContentMargin.Right),
					}}
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{
				Alignment: layout.Middle,
			}.Layout(gtx,
				layout.Rigid(c.LeftGutter.Layout),
				layout.Flexed(1, func(gtx C) D {
					return messageAlignment.Layout(gtx, c.layoutBubble)
				}),
				layout.Rigid(c.layoutTimeOrIcon),
			)
		}),
		layout.Rigid(func(gtx C) D {
			if c.StatusMessage.Text == "" {
				return D{}
			}
			return layout.E.Layout(gtx, func(gtx C) D {
				return c.RightGutterPadding.Layout(gtx, c.StatusMessage.Layout)
			})
		}),
	)
}

// layoutBubble lays out the chat bubble.
func (c MessageStyle) layoutBubble(gtx C) D {
	gtx.Constraints.Max.X = int(float32(gtx.Constraints.Max.X) * 0.8)
	max := gtx.Px(c.MaxMessageWidth)
	if gtx.Constraints.Max.X > max {
		gtx.Constraints.Max.X = max
	}
	return c.ContentMargin.Layout(gtx, func(gtx C) D {
		if c.Image.Src == (paint.ImageOp{}) {
			return c.Surface.Layout(gtx, func(gtx C) D {
				return c.ContentPadding.Layout(gtx, func(gtx C) D {
					return c.Content.Layout(gtx)
				})
			})
		}
		return material.Clickable(gtx, c.Clicks, func(gtx C) D {
			gtx.Constraints.Max.Y = gtx.Px(c.MaxImageHeight)
			defer op.Save(gtx.Ops).Load()
			macro := op.Record(gtx.Ops)
			dims := c.Image.Layout(gtx)
			call := macro.Stop()
			radius := float32(gtx.Px(unit.Dp(8)))
			clip.RRect{
				Rect: f32.Rectangle{Max: layout.FPt(dims.Size)},
				NE:   radius, NW: radius, SE: radius, SW: radius,
			}.Add(gtx.Ops)
			call.Add(gtx.Ops)
			return dims
		})
	})
}

// layoutTimeOrIcon lays out a status icon if one is set, and
// otherwise lays out the time the messages was sent.
func (c MessageStyle) layoutTimeOrIcon(gtx C) D {
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx C) D {
			return c.RightGutterPadding.Layout(gtx, func(gtx C) D {
				sideLength := gtx.Px(c.IconSize)
				gtx.Constraints.Max.X = sideLength
				gtx.Constraints.Max.Y = sideLength
				gtx.Constraints.Min = gtx.Constraints.Constrain(gtx.Constraints.Min)
				if c.StatusIcon != nil {
					return c.StatusIcon.Layout(gtx)
				}
				return D{Size: gtx.Constraints.Max}
			})
		}),
		layout.Expanded(func(gtx C) D {
			if c.StatusIcon != nil {
				return D{}
			}
			return layout.Center.Layout(gtx, c.Time.Layout)
		}),
	)
}

// Luminance computes the relative brightness of a color, normalized between
// [0,1]. Ignores alpha.
func Luminance(c color.NRGBA) float64 {
	return (float64(float64(0.299)*float64(c.R) + float64(0.587)*float64(c.G) + float64(0.114)*float64(c.B))) / 255
}
