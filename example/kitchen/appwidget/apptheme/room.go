package apptheme

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"git.sr.ht/~gioverse/chat/example/kitchen/appwidget"
	"git.sr.ht/~gioverse/chat/example/kitchen/model"
)

// RoomStyle lays out a room select card.
type RoomStyle struct {
	*appwidget.Room
	Image     Image
	Name      material.LabelStyle
	Summary   material.LabelStyle
	TimeStamp material.LabelStyle
	Indicator color.NRGBA
}

// Room creates a style type that can lay out the data for a room.
func Room(th *material.Theme, interact *appwidget.Room, room *model.Room) RoomStyle {
	interact.Image.Cache(room.Image)
	var (
		latest model.Message
	)
	if l := room.Latest; l != nil {
		latest = *l
	}
	return RoomStyle{
		Room: interact,
		// TODO(jfm): name could use bold text.
		Name:      material.Label(th, unit.Sp(14), room.Name),
		Summary:   material.Label(th, unit.Sp(12), latest.Content),
		TimeStamp: material.Label(th, unit.Sp(12), latest.SentAt.Local().Format("15:04")),
		Image: Image{
			Image: widget.Image{
				Src: interact.Image.Op(),
				Fit: widget.Contain,
			},
			Radii:  unit.Dp(8),
			Height: unit.Dp(25),
			Width:  unit.Dp(25),
		},
		Indicator: th.ContrastBg,
	}
}

func (room RoomStyle) Layout(gtx C) D {
	// NOTE(jfm): need the vertical dims.
	// Tried using flex and stack to no avail, using macro as a stop-gap.
	macro := op.Record(gtx.Ops)
	dims := material.Clickable(gtx, &room.Clickable, func(gtx C) D {
		return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx C) D {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
			}.Layout(
				gtx,
				layout.Rigid(func(gtx C) D {
					return room.Image.Layout(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					return D{Size: image.Point{X: gtx.Px(unit.Dp(5))}}
				}),
				layout.Flexed(1, func(gtx C) D {
					return layout.Flex{
						Axis: layout.Vertical,
					}.Layout(
						gtx,
						layout.Rigid(func(gtx C) D {
							return room.Name.Layout(gtx)
						}),
						layout.Rigid(func(gtx C) D {
							return D{Size: image.Point{Y: gtx.Px(unit.Dp(5))}}
						}),
						layout.Rigid(func(gtx C) D {
							return component.TruncatingLabelStyle(room.Summary).Layout(gtx)
						}),
					)
				}),
				layout.Rigid(func(gtx C) D {
					return D{Size: image.Point{X: gtx.Px(unit.Dp(5))}}
				}),
				layout.Rigid(func(gtx C) D {
					return room.TimeStamp.Layout(gtx)
				}),
			)
		})
	})
	call := macro.Stop()
	return layout.Flex{Axis: layout.Horizontal}.Layout(
		gtx,
		layout.Rigid(func(gtx C) D {
			sz := image.Point{
				X: gtx.Px(unit.Dp(3)),
				Y: dims.Size.Y,
			}
			if !room.Active {
				return D{Size: sz}
			}
			return component.Rect{
				Size:  sz,
				Color: room.Indicator,
			}.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			call.Add(gtx.Ops)
			return dims
		}),
	)
}
