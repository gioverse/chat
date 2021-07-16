package apptheme

import (
	"image/color"

	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/lucasb-eyer/go-colorful"
)

// Note: the values choosen are a best-guess heuristic, open to change.
var (
	DefaultMaxImageHeight  = unit.Dp(400)
	DefaultMaxMessageWidth = unit.Dp(600)
	DefaultAvatarSize      = unit.Dp(24)
)

// ToNRGBA converts a colorful.Color to the nearest representable color.NRGBA.
func ToNRGBA(c colorful.Color) color.NRGBA {
	r, g, b, a := c.RGBA()
	return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
}

// Theme wraps the material.Theme with useful application-specific
// theme information.
type Theme struct {
	*material.Theme
	// UserColors tracks a mapping from chat username to the color
	// chosen to represent that user.
	UserColors map[string]UserColorData
	// DangerColor is the color used to indicate errors.
	DangerColor color.NRGBA
	// AvatarSize specifies how large the avatar image should be.
	AvatarSize unit.Value
}

// UserColorData tracks both a color and its luminance.
type UserColorData struct {
	color.NRGBA
	Luminance float64
}

// NewTheme instantiates a theme using the provided fonts.
func NewTheme(fonts []text.FontFace) *Theme {
	return &Theme{
		Theme:       material.NewTheme(fonts),
		UserColors:  make(map[string]UserColorData),
		DangerColor: color.NRGBA{R: 200, A: 255},
		AvatarSize:  DefaultAvatarSize,
	}
}

// UserColor returns a color for the provided username. It will choose a
// new color if the username is new.
func (t *Theme) UserColor(username string) UserColorData {
	if c, ok := t.UserColors[username]; ok {
		return c
	}
	c := colorful.FastHappyColor().Clamped()

	uc := UserColorData{
		NRGBA: ToNRGBA(c),
	}
	uc.Luminance = (0.299*float64(uc.NRGBA.R) + 0.587*float64(uc.NRGBA.G) + 0.114*float64(uc.NRGBA.B)) / 255
	t.UserColors[username] = uc
	return uc
}
