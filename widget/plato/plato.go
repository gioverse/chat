// Package plato implements themed styles for Plato Team Inc.
// https://www.platoapp.com/
package plato

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	chatlayout "git.sr.ht/~gioverse/chat/layout"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

var (
	DefaultMaxImageHeight  = unit.Dp(400)
	DefaultMaxMessageWidth = unit.Dp(600)
	DefaultMinMessageWidth = unit.Dp(80)
	DefaultAvatarSize      = unit.Dp(28)
	LocalMessageColor      = color.NRGBA{R: 63, G: 133, B: 232, A: 255}
	NonLocalMessageColor   = color.NRGBA{R: 50, G: 50, B: 50, A: 255}
)

// TickIcon used for read receipts.
var TickIcon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.NavigationCheck)
	return icon
}()

// UserInfoStyle defines the presentation of information about a user.
type UserInfoStyle struct {
	// Username configures the presentation of the user name text.
	Username material.LabelStyle
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
func UserInfo(th *material.Theme, username string) UserInfoStyle {
	return UserInfoStyle{
		Username: material.Body1(th, username),
		Spacer:   layout.Spacer{Width: unit.Dp(8)},
	}
}

// Layout the user information.
func (ui UserInfoStyle) Layout(gtx C) D {
	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
	}.Layout(gtx,
		chatlayout.Reverse(ui.Local,
			layout.Rigid(ui.Spacer.Layout),
			layout.Rigid(ui.Username.Layout),
		)...,
	)
}

// Luminance computes the relative brightness of a color, normalized between
// [0,1]. Ignores alpha.
func Luminance(c color.NRGBA) float64 {
	return (float64(float64(0.299)*float64(c.R) + float64(0.587)*float64(c.G) + float64(0.114)*float64(c.B))) / 255
}
