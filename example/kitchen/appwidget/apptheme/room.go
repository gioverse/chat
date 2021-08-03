package apptheme

import (
	"image"
	"image/color"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"git.sr.ht/~gioverse/chat/example/kitchen/appwidget"
	chatlayout "git.sr.ht/~gioverse/chat/layout"
	matchat "git.sr.ht/~gioverse/chat/widget/material"
)

// RoomStyle lays out a room select card.
type RoomStyle struct {
	*appwidget.Room
	Image     matchat.Image
	Name      material.LabelStyle
	Summary   material.LabelStyle
	TimeStamp material.LabelStyle
	Indicator color.NRGBA
	Overlay   color.NRGBA
}

// RoomConfig configures room item display.
type RoomConfig struct {
	// Name of the room as raw text.
	Name string
	// Image of the room.
	Image image.Image
	// Content of the latest message as raw text.
	Content string
	// SentAt timestamp of the latest message.
	SentAt time.Time
}

// Room creates a style type that can lay out the data for a room.
func Room(th *material.Theme, interact *appwidget.Room, room *RoomConfig) RoomStyle {
	interact.Image.Cache(room.Image)
	return RoomStyle{
		Room: interact,
		// TODO(jfm): name could use bold text.
		Name:      material.Label(th, unit.Sp(14), room.Name),
		Summary:   material.Label(th, unit.Sp(12), room.Content),
		TimeStamp: material.Label(th, unit.Sp(12), room.SentAt.Local().Format("15:04")),
		Image: matchat.Image{
			Image: widget.Image{
				Src: interact.Image.Op(),
				Fit: widget.Contain,
			},
			Radii:  unit.Dp(8),
			Height: unit.Dp(25),
			Width:  unit.Dp(25),
		},
		Indicator: th.ContrastBg,
		Overlay:   component.WithAlpha(th.Fg, 50),
	}
}

func (room RoomStyle) Layout(gtx C) D {
	var (
		surface = func(gtx C, w layout.Widget) D { return w(gtx) }
		dims    layout.Dimensions
	)
	if room.Active {
		surface = chatlayout.Background(room.Overlay).Layout
		defer func() {
			// Close-over the dimensions and layout the indicator atop everything
			// else.
			component.Rect{
				Size: image.Point{
					X: gtx.Px(unit.Dp(3)),
					Y: dims.Size.Y,
				},
				Color: room.Indicator,
			}.Layout(gtx)
		}()
	}
	dims = surface(gtx, func(gtx C) D {
		return material.Clickable(gtx, &room.Clickable, func(gtx C) D {
			return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx C) D {
				return layout.Flex{
					Axis:      layout.Horizontal,
					Alignment: layout.Middle,
				}.Layout(
					gtx,
					layout.Rigid(func(gtx C) D {
						return room.Image.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
					layout.Flexed(1, func(gtx C) D {
						return layout.Flex{
							Axis: layout.Vertical,
						}.Layout(
							gtx,
							layout.Rigid(func(gtx C) D {
								return room.Name.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
							layout.Rigid(func(gtx C) D {
								return component.TruncatingLabelStyle(room.Summary).Layout(gtx)
							}),
						)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
					layout.Rigid(func(gtx C) D {
						return room.TimeStamp.Layout(gtx)
					}),
				)
			})
		})
	})
	return dims
}
