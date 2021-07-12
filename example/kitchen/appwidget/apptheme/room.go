package apptheme

import (
	"image"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"git.sr.ht/~gioverse/chat/example/kitchen/appwidget"
	"git.sr.ht/~gioverse/chat/example/kitchen/model"
)

type RoomStyle struct {
	*appwidget.Room
	Image     widget.Image
	Name      material.LabelStyle
	Summary   material.LabelStyle
	TimeStamp material.LabelStyle
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
		Image: widget.Image{
			Src: interact.Image.Op(),
			Fit: widget.Contain,
		},
	}
}

func (room RoomStyle) Layout(gtx C) D {
	return material.Clickable(gtx, &room.Clickable, func(gtx C) D {
		return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx C) D {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Start,
			}.Layout(
				gtx,
				layout.Rigid(func(gtx C) D {
					gtx.Constraints.Max.X = gtx.Px(unit.Dp(25))
					gtx.Constraints.Min = gtx.Constraints.Constrain(gtx.Constraints.Min)
					return room.Image.Layout(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					return D{Size: image.Point{X: gtx.Px(unit.Dp(10))}}
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
							return component.TruncatingLabelStyle(room.Summary).Layout(gtx)
						}),
					)
				}),
				layout.Rigid(func(gtx C) D {
					return D{Size: image.Point{X: gtx.Px(unit.Dp(10))}}
				}),
				layout.Rigid(func(gtx C) D {
					return room.TimeStamp.Layout(gtx)
				}),
			)
		})
	})
}
