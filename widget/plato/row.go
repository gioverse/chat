package plato

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
	chatmaterial "git.sr.ht/~gioverse/chat/widget/material"
)

// RowStyle configures the presentation of a chat message within
// a vertical list of chat messages.
//
// In particular, RowStyle is repsonsible for gutters and anchoring of
// messages.
type RowStyle struct {
	OuterMargin chatlayout.VerticalMarginStyle
	chatlayout.GutterStyle
	// Local indicates that the message was sent by the local user,
	// and should be right-aligned.
	Local bool
	// Time is the timestamp associated with the message.
	Time material.LabelStyle
	// StatusMessage defines a warning message to be displayed beneath the
	// chat message.
	StatusMessage material.LabelStyle
	// ContentMargin configures space around the chat bubble.
	ContentMargin chatlayout.VerticalMarginStyle
	// UserInfoStyle configures how the sender's information is displayed.
	UserInfoStyle
	// Avatar image for the user.
	Avatar chatmaterial.Image
	// MessageStyle configures how the text and its background are presented.
	MessageStyle
	// Interaction holds the interactive state of this message.
	Interaction *chatwidget.Row
	// Menu configures the right-click context menu for this message.
	Menu component.MenuStyle
}

// RowConfig describes the aspects of a chat row relevant for displaying
// it within a widget.
type RowConfig struct {
	Sender  string
	Avatar  image.Image
	Content string
	SentAt  time.Time
	Image   image.Image
	Local   bool
}

// NewRow creates a style type that can lay out the data for a message.
func NewRow(
	th *material.Theme,
	interact *chatwidget.Row,
	menu *component.MenuState,
	msg RowConfig,
) RowStyle {
	interact.Avatar.Cache(msg.Avatar)
	ms := RowStyle{
		OuterMargin: chatlayout.VerticalMargin(),
		GutterStyle: chatlayout.GutterStyle{
			LeftWidth:  unit.Dp(unit.Dp(12).V + DefaultAvatarSize.V),
			RightWidth: unit.Dp(unit.Dp(12).V + DefaultAvatarSize.V),
			Alignment:  layout.Start,
		},
		Time:          material.Body2(th, msg.SentAt.Local().Format("15:04")),
		Local:         msg.Local,
		ContentMargin: chatlayout.VerticalMargin(),
		UserInfoStyle: UserInfo(th, msg.Sender),
		Avatar: chatmaterial.Image{
			Image: widget.Image{
				Src:      interact.Avatar.Op(),
				Fit:      widget.Cover,
				Position: layout.Center,
			},
			// Half size radius makes for a circle.
			Radii:  unit.Dp(DefaultAvatarSize.V * 0.5),
			Width:  DefaultAvatarSize,
			Height: DefaultAvatarSize,
		},
		Interaction: interact,
		Menu:        component.Menu(th, menu),
		MessageStyle: Message(th, &interact.Message, MessageConfig{
			Content: msg.Content,
			Seen:    true,
			Time:    msg.SentAt,
			Local:   msg.Local,
		}),
	}
	ms.UserInfoStyle.Local = msg.Local
	return ms
}

// Layout the message.
func (c RowStyle) Layout(gtx C) D {
	return c.OuterMargin.Layout(gtx, func(gtx C) D {
		alignment := layout.W
		if c.Local {
			alignment = layout.E
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return c.GutterStyle.Layout(gtx,
					nil,
					func(gtx C) D {
						return alignment.Layout(gtx, c.UserInfoStyle.Layout)
					},
					nil,
				)
			}),
			layout.Rigid(func(gtx C) D {
				return c.GutterStyle.Layout(gtx,
					func(gtx C) D {
						if !c.Local {
							return c.layoutAvatar(gtx)
						}
						return D{}
					},
					func(gtx C) D {
						return alignment.Layout(gtx, c.layoutBubble)
					},
					func(gtx C) D {
						if c.Local {
							return c.layoutAvatar(gtx)
						}
						return D{}
					},
				)
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

// layoutAvatar lays out the user avatar image.
func (c RowStyle) layoutAvatar(gtx C) D {
	return layout.Inset{
		Top:    unit.Dp(12),
		Bottom: unit.Dp(6),
		Left:   unit.Dp(6),
		Right:  unit.Dp(6),
	}.Layout(gtx, func(gtx C) D {
		return c.Avatar.Layout(gtx)
	})
}
