package material

import (
	"image"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	chatlayout "git.sr.ht/~gioverse/chat/layout"
	chatwidget "git.sr.ht/~gioverse/chat/widget"
)

// RowStyle configures the presentation of a chat message within
// a vertical list of chat messages.
type RowStyle struct {
	OuterMargin chatlayout.VerticalMarginStyle
	chatlayout.GutterStyle
	// Local indicates that the message was sent by the local user,
	// and should be right-aligned.
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

// NewRow creates a style type that can lay out the data for a message.
func NewRow(th *material.Theme, interact *chatwidget.Row, menu *component.MenuState, msg RowConfig) RowStyle {
	if interact == nil {
		interact = &chatwidget.Row{}
	}
	if menu == nil {
		menu = &component.MenuState{}
	}
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