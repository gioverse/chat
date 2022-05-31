package apptheme

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/lucasb-eyer/go-colorful"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// Note: the values choosen are a best-guess heuristic, open to change.
var (
	DefaultMaxImageHeight  = unit.Dp(400)
	DefaultMaxMessageWidth = unit.Dp(600)
	DefaultAvatarSize      = unit.Dp(24)
)

var (
	Light = Palette{
		Error:         rgb(0xB00020),
		Surface:       rgb(0xFFFFFF),
		Bg:            rgb(0xDCDCDC),
		BgSecondary:   rgb(0xEBEBEB),
		OnError:       rgb(0xFFFFFF),
		OnSurface:     rgb(0x000000),
		OnBg:          rgb(0x000000),
		OnBgSecondary: rgb(0x000000),
	}
	Dark = Palette{
		Error:         rgb(0xB00020),
		Surface:       rgb(0x222222),
		Bg:            rgb(0x000000),
		BgSecondary:   rgb(0x444444),
		OnError:       rgb(0xFFFFFF),
		OnSurface:     rgb(0xFFFFFF),
		OnBg:          rgb(0xEEEEEE),
		OnBgSecondary: rgb(0xFFFFFF),
	}
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
	// AvatarSize specifies how large the avatar image should be.
	AvatarSize unit.Dp
	// Palette specifies semantic colors.
	Palette Palette
}

// Palette defines non-brand semantic colors.
//
// `On` colors define a color that is appropriate to display atop it's
// counterpart.
type Palette struct {
	// Error used to indicate errors.
	Error   color.NRGBA
	OnError color.NRGBA
	// Surface affect surfaces of components, such as cards, sheets and menus.
	Surface   color.NRGBA
	OnSurface color.NRGBA
	// Bg appears behind scrollable content.
	Bg   color.NRGBA
	OnBg color.NRGBA
	// BgSecondary appears behind scrollable content.
	BgSecondary   color.NRGBA
	OnBgSecondary color.NRGBA
}

// UserColorData tracks both a color and its luminance.
type UserColorData struct {
	color.NRGBA
	Luminance float64
}

// NewTheme instantiates a theme using the provided fonts.
func NewTheme(fonts []text.FontFace) *Theme {
	var (
		base = material.NewTheme(fonts)
	)
	th := Theme{
		Theme:      base,
		UserColors: make(map[string]UserColorData),
		AvatarSize: DefaultAvatarSize,
	}
	th.UsePalette(Light)
	return &th
}

// UsePalette changes to the specified palette.
func (t *Theme) UsePalette(p Palette) {
	t.Palette = p
	t.Theme.Bg = t.Palette.Bg
	t.Theme.Fg = t.Palette.OnBg
}

// Toggle the active theme between pre-configured Light and Dark palettes.
func (t *Theme) Toggle() {
	if t.Palette == Light {
		t.UsePalette(Dark)
	} else {
		t.UsePalette(Light)
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

// LocalUserColor returns a color for the "local" user.
// Local user color is defined as the theme's surface color and it's luminance.
func (t *Theme) LocalUserColor() UserColorData {
	c := t.Palette.Surface
	return UserColorData{
		NRGBA:     c,
		Luminance: (0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)) / 255,
	}
}

// Contrast against a given luminance.
//
// Defaults to a color that contrasts the background color, if the threshold
// is met, the background color itself is returned.
//
// Note this will depend on the specific palette in question, and may not be a
// good generalization particularly for low-contrast palettes.
func (t *Theme) Contrast(luminance float64) color.NRGBA {
	var (
		contrast = luminance < 0.5
	)
	if t.Palette == Dark {
		contrast = luminance > 0.5
	}
	if contrast {
		return t.Palette.Bg
	}
	return t.Palette.OnBg
}

func rgb(c uint32) color.NRGBA {
	return argb(0xff000000 | c)
}

func argb(c uint32) color.NRGBA {
	return color.NRGBA{A: uint8(c >> 24), R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c)}
}
