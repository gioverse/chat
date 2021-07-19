package material

import (
	"image"
	"image/color"
	"time"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"gioui.org/x/richtext"
	chatlayout "git.sr.ht/~gioverse/chat/layout"
	"git.sr.ht/~gioverse/chat/ninepatch"
	chatwidget "git.sr.ht/~gioverse/chat/widget"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

// Note: the values choosen are a best-guess heuristic, open to change.
var (
	DefaultMaxImageHeight  = unit.Dp(400)
	DefaultMaxMessageWidth = unit.Dp(600)
	DefaultAvatarSize      = unit.Dp(24)
	DefaultDangerColor     = color.NRGBA{R: 200, A: 255}
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

// RowConfig describes the aspects of a chat message relevant for
// displaying it within a widget.
type RowConfig struct {
	Sender  string
	Avatar  image.Image
	Content string
	SentAt  time.Time
	Image   image.Image
	Local   bool
	Status  string
}

// UserInfoStyle defines the presentation of information about a user.
// It can present the user's name and avatar with a space between them.
type UserInfoStyle struct {
	// Username configures the presentation of the user name text.
	Username material.LabelStyle
	// Avatar defines the image shown as the user's avatar.
	Avatar Image
	// Spacer is inserted between the username and avatar fields.
	layout.Spacer
	// Local controls the Left-to-Right ordering of layout. If false,
	// the Left-to-Right order will be:
	//   - Avatar
	//   - Spacer
	//   - Username
	// If true, the order is reversed.
	Local bool
}

// UserInfo constructs a UserInfoStyle with sensible defaults.
func UserInfo(th *material.Theme, interact *chatwidget.UserInfo, username string, avatar image.Image) UserInfoStyle {
	interact.Avatar.Cache(avatar)
	return UserInfoStyle{
		Username: material.Body1(th, username),
		Avatar: Image{
			Image: widget.Image{
				Src:      interact.Avatar.Op(),
				Fit:      widget.Cover,
				Position: layout.Center,
			},
			Radii:  unit.Dp(8),
			Width:  DefaultAvatarSize,
			Height: DefaultAvatarSize,
		},
		Spacer: layout.Spacer{Width: unit.Dp(8)},
	}
}

// Layout the user information.
func (ui UserInfoStyle) Layout(gtx C) D {
	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
	}.Layout(gtx,
		chatlayout.Reverse(ui.Local,
			layout.Rigid(ui.Avatar.Layout),
			layout.Rigid(ui.Spacer.Layout),
			layout.Rigid(ui.Username.Layout),
		)...,
	)
}

// RowStyle configures the presentation of a chat message within
// a vertical list of chat messages.
type RowStyle struct {
	OuterMargin chatlayout.VerticalMarginStyle
	chatlayout.GutterStyle
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
	// StatusMessage defines a warning message to be displayed beneath the
	// chat message.
	StatusMessage material.LabelStyle
	// ContentMargin configures space around the chat bubble.
	ContentMargin chatlayout.VerticalMarginStyle
	// UserInfoStyle configures how the sender's information is displayed.
	UserInfoStyle
	// MessageStyle configures how the text and its background are presented.
	MessageStyle
	// Interaction holds the interactive state of this message.
	Interaction *chatwidget.Row
	// Menu configures the right-click context menu for this message.
	Menu component.MenuStyle
}

// NewRow creates a style type that can lay out the data for a message.
func NewRow(th *material.Theme, interact *chatwidget.Row, menu *component.MenuState, msg RowConfig) RowStyle {
	ms := RowStyle{
		OuterMargin:   chatlayout.VerticalMargin(),
		GutterStyle:   chatlayout.Gutter(),
		Time:          material.Body2(th, msg.SentAt.Local().Format("15:04")),
		Local:         msg.Local,
		IconSize:      unit.Dp(32),
		ContentMargin: chatlayout.VerticalMargin(),
		UserInfoStyle: UserInfo(th, &interact.UserInfo, msg.Sender, msg.Avatar),
		Interaction:   interact,
		Menu:          component.Menu(th, menu),
		MessageStyle:  Message(th, &interact.Message, msg.Content, msg.Image),
	}
	ms.UserInfoStyle.Local = msg.Local
	if msg.Status != "" {
		ms.StatusMessage = material.Body2(th, msg.Status)
		ms.StatusMessage.Color = DefaultDangerColor
		ms.StatusIcon = ErrorIcon
		ms.StatusIcon.Color = DefaultDangerColor
	}
	return ms
}

// Layout the message.
func (c RowStyle) Layout(gtx C) D {
	return c.OuterMargin.Layout(gtx, func(gtx C) D {
		messageAlignment := layout.W
		if c.Local {
			messageAlignment = layout.E
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return c.GutterStyle.Layout(gtx,
					nil,
					func(gtx C) D {
						return messageAlignment.Layout(gtx, c.UserInfoStyle.Layout)
					},
					nil,
				)
			}),
			layout.Rigid(func(gtx C) D {
				return c.GutterStyle.Layout(gtx,
					nil,
					func(gtx C) D {
						return messageAlignment.Layout(gtx, c.layoutBubble)
					},
					c.layoutTimeOrIcon,
				)
			}),
			layout.Rigid(func(gtx C) D {
				if c.StatusMessage.Text == "" {
					return D{}
				}
				return layout.E.Layout(gtx, c.StatusMessage.Layout)
			}),
		)
	})
}

// MessageStyle configures the presentation of a chat message.
type MessageStyle struct {
	// Interaction holds the stateful parts of this message.
	Interaction *chatwidget.Message
	// MaxMessageWidth constrains the display width of the message's background.
	MaxMessageWidth unit.Value
	// MaxImageHeight constrains the maximum height of an image message. The image
	// will be scaled to fit within this height.
	MaxImageHeight unit.Value
	// ContentPadding separates the Content field from the edges of the background.
	ContentPadding layout.Inset
	// BubbleStyle configures a chat bubble beneath the message. If NinePatch is
	// non-nil, this field is ignored.
	BubbleStyle
	// Ninepatch provides a ninepatch stretchable image background. Only used if
	// non-nil.
	*ninepatch.NinePatch
	// Content is the actual styled text of the message.
	Content richtext.TextStyle
	// Image is the optional image content of the message.
	Image
}

// Message constructs a MessageStyle with sensible defaults.
func Message(th *material.Theme, interact *chatwidget.Message, content string, img image.Image) MessageStyle {
	interact.Image.Cache(img)
	l := material.Body1(th, "")
	return MessageStyle{
		BubbleStyle: Bubble(th),
		Content: richtext.Text(&interact.InteractiveText, th.Shaper, richtext.SpanStyle{
			Font:    l.Font,
			Size:    l.TextSize,
			Color:   th.Fg,
			Content: content,
		}),
		ContentPadding: layout.UniformInset(unit.Dp(8)),
		Image: Image{
			Image: widget.Image{
				Src:      interact.Image.Op(),
				Fit:      widget.ScaleDown,
				Position: layout.Center,
			},
			Radii: unit.Dp(8),
		},
		MaxMessageWidth: DefaultMaxMessageWidth,
		MaxImageHeight:  DefaultMaxImageHeight,
		Interaction:     interact,
	}
}

// WithNinePatch sets the message surface to a ninepatch image.
func (c MessageStyle) WithNinePatch(th *material.Theme, np ninepatch.NinePatch) MessageStyle {
	c.NinePatch = &np
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
				c.Content.Styles[i].Color = th.Bg
			}
		}
	}
	return c
}

// WithBubbleColor sets the message surface to a ninepatch image.
func (c MessageStyle) WithBubbleColor(th *material.Theme, col color.NRGBA, luminance float64) MessageStyle {
	c.BubbleStyle.Color = col
	if luminance < .5 {
		for i := range c.Content.Styles {
			c.Content.Styles[i].Color = th.Bg
		}
	}
	return c
}

// Layout the message atop its background.
func (m MessageStyle) Layout(gtx C) D {
	gtx.Constraints.Max.X = int(float32(gtx.Constraints.Max.X) * 0.8)
	max := gtx.Px(m.MaxMessageWidth)
	if gtx.Constraints.Max.X > max {
		gtx.Constraints.Max.X = max
	}
	if m.Image.Src == (paint.ImageOp{}) {
		surface := m.BubbleStyle.Layout
		if m.NinePatch != nil {
			surface = m.NinePatch.Layout
		}
		return surface(gtx, func(gtx C) D {
			return m.ContentPadding.Layout(gtx, func(gtx C) D {
				return m.Content.Layout(gtx)
			})
		})
	}
	defer pointer.CursorNameOp{Name: pointer.CursorPointer}.Add(gtx.Ops)
	return material.Clickable(gtx, &m.Interaction.Clickable, func(gtx C) D {
		gtx.Constraints.Max.Y = gtx.Px(m.MaxImageHeight)
		return m.Image.Layout(gtx)
	})
}

// layoutBubble lays out the chat bubble.
func (c RowStyle) layoutBubble(gtx C) D {
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx C) D {
			return c.ContentMargin.Layout(gtx, c.MessageStyle.Layout)
		}),
		layout.Expanded(func(gtx C) D {
			return c.Interaction.ContextArea.Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min = image.Point{}
				return c.Menu.Layout(gtx)
			})
		}),
	)
}

// layoutTimeOrIcon lays out a status icon if one is set, and
// otherwise lays out the time the messages was sent.
func (c RowStyle) layoutTimeOrIcon(gtx C) D {
	return layout.Center.Layout(gtx, func(gtx C) D {
		if c.StatusIcon == nil {
			return c.Time.Layout(gtx)
		}
		sideLength := gtx.Px(c.IconSize)
		gtx.Constraints.Max.X = sideLength
		gtx.Constraints.Max.Y = sideLength
		gtx.Constraints.Min = gtx.Constraints.Constrain(gtx.Constraints.Min)
		return c.StatusIcon.Layout(gtx)
	})
}

// Luminance computes the relative brightness of a color, normalized between
// [0,1]. Ignores alpha.
func Luminance(c color.NRGBA) float64 {
	return (float64(float64(0.299)*float64(c.R) + float64(0.587)*float64(c.G) + float64(0.114)*float64(c.B))) / 255
}
