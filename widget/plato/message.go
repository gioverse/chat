package plato

import (
	"image/color"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"gioui.org/x/richtext"
	"git.sr.ht/~gioverse/chat/ninepatch"
	chatwidget "git.sr.ht/~gioverse/chat/widget"
	chatmaterial "git.sr.ht/~gioverse/chat/widget/material"
)

// MessageStyle configures the presentation of a chat message.
type MessageStyle struct {
	// Interaction holds the stateful parts of this message.
	Interaction *chatwidget.Message
	// MaxMessageWidth constrains the display width of the message's background.
	MaxMessageWidth unit.Value
	// MinMessageWidth constrains the display width of the message's background.
	MinMessageWidth unit.Value
	// MaxImageHeight constrains the maximum height of an image message. The image
	// will be scaled to fit within this height.
	MaxImageHeight unit.Value
	// ContentPadding separates the Content field from the edges of the background.
	ContentPadding layout.Inset
	// BubbleStyle configures a chat bubble beneath the message. If NinePatch is
	// non-nil, this field is ignored.
	chatmaterial.BubbleStyle
	// Ninepatch provides a ninepatch stretchable image background. Only used if
	// non-nil.
	*ninepatch.NinePatch
	// Content is the actual styled text of the message.
	Content richtext.TextStyle
	// Seen if this message has been seen, show a read receipt.
	Seen bool
	// Time is the timestamp associated with the message.
	Time material.LabelStyle
	// Receipt lays out the read receipt.
	Receipt *widget.Icon
	// Clickable indicates whether the message content should be able to receive
	// click events.
	Clickable bool
	// Compact mode avoids laying out timestamp and read-receipt.
	Compact bool
}

// MessageConfig describes aspects of a chat message.
type MessageConfig struct {
	// Content specifies the raw textual content of the message.
	Content string
	// Seen indicates whether this message has been "seen" by other users.
	Seen bool
	// Time indicates when this message was sent.
	Time time.Time
	// Color of the message bubble.
	// Defaults to LocalMessageColor.
	Color color.NRGBA
	// Compact mode avoids laying out timestamp and read-receipt.
	Compact bool
}

// Message constructs a MessageStyle with sensible defaults.
func Message(th *material.Theme, interact *chatwidget.Message, msg MessageConfig) MessageStyle {
	l := material.Body1(th, "")
	TickIcon.Color = color.NRGBA{G: 200, B: 50, A: 255}
	return MessageStyle{
		BubbleStyle: func() chatmaterial.BubbleStyle {
			b := chatmaterial.Bubble(th)
			if msg.Color == (color.NRGBA{}) {
				msg.Color = LocalMessageColor
			}
			b.Color = msg.Color
			return b
		}(),
		Content: richtext.Text(&interact.InteractiveText, th.Shaper, richtext.SpanStyle{
			Font:    l.Font,
			Size:    l.TextSize,
			Color:   th.Fg,
			Content: msg.Content,
		}),
		ContentPadding:  layout.UniformInset(unit.Dp(8)),
		MaxMessageWidth: DefaultMaxMessageWidth,
		MinMessageWidth: DefaultMinMessageWidth,
		MaxImageHeight:  DefaultMaxImageHeight,
		Interaction:     interact,
		Time: func() material.LabelStyle {
			l := material.Label(th, unit.Sp(11), msg.Time.Local().Format("3:04 PM"))
			l.Color = component.WithAlpha(l.Color, 200)
			return l
		}(),
		Receipt: TickIcon,
		Compact: msg.Compact,
	}
}

// WithNinePatch sets the message surface to a ninepatch image.
func (c MessageStyle) WithNinePatch(th *material.Theme, np ninepatch.NinePatch) MessageStyle {
	c.NinePatch = &np
	c.ContentPadding = layout.Inset{}
	return c
}

// WithBubbleColor sets the message bubble color and selects a contrasted text color.
func (c MessageStyle) WithBubbleColor(th *material.Theme, col color.NRGBA, luminance float64) MessageStyle {
	c.BubbleStyle.Color = col
	if luminance < .5 {
		for i := range c.Content.Styles {
			c.Content.Styles[i].Color = th.Bg
		}
	}
	return c
}

func (c *MessageStyle) TextColor(cl color.NRGBA) {
	c.Time.Color = cl
	for i := range c.Content.Styles {
		c.Content.Styles[i].Color = cl
	}
}

// Layout the message atop its background.
func (m MessageStyle) Layout(gtx C) D {
	gtx.Constraints.Max.X = int(float32(gtx.Constraints.Max.X) * 0.8)
	max := gtx.Px(m.MaxMessageWidth)
	if gtx.Constraints.Max.X > max {
		gtx.Constraints.Max.X = max
	}
	surface := m.BubbleStyle.Layout
	if m.NinePatch != nil {
		surface = m.NinePatch.Layout
	}
	if m.Compact {
		return surface(gtx, func(gtx C) D {
			return m.ContentPadding.Layout(gtx, func(gtx C) D {
				return m.Content.Layout(gtx)
			})
		})
	}
	macro := op.Record(gtx.Ops)
	dims := m.ContentPadding.Layout(gtx, func(gtx C) D {
		return m.Content.Layout(gtx)
	})
	call := macro.Stop()
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			if !m.Clickable {
				return D{}
			}
			return m.Interaction.Clickable.Layout(gtx)
		}),
		layout.Stacked(func(gtx C) D {
			return surface(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						call.Add(gtx.Ops)
						return dims
					}),
					layout.Rigid(func(gtx C) D {
						width := gtx.Px(m.MinMessageWidth)
						if dims.Size.X > width {
							width = dims.Size.X
						}
						gtx.Constraints.Max.X = width
						return m.ContentPadding.Layout(gtx, func(gtx C) D {
							return layout.Flex{
								Axis:      layout.Horizontal,
								Alignment: layout.Middle,
							}.Layout(gtx,
								layout.Flexed(1, func(gtx C) D {
									return D{Size: gtx.Constraints.Min}
								}),
								layout.Rigid(func(gtx C) D {
									return m.Time.Layout(gtx)
								}),
								layout.Rigid(func(gtx C) D {
									return m.Receipt.Layout(gtx)
								}),
							)
						})
					}),
				)
			})
		}),
	)
}
