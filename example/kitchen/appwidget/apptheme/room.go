package apptheme

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
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
		Name:      material.Body1(th, room.Name),
		Summary:   material.Body1(th, latest.Content),
		TimeStamp: material.Body1(th, latest.SentAt.String()),
		Image:     widget.Image{Src: interact.Image.Op()},
	}
}

func (room RoomStyle) Layout(gtx C) D {
	return material.Clickable(gtx, &room.Clickable, func(gtx C) D {
		return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx C) D {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
			}.Layout(
				gtx,
				layout.Rigid(func(gtx C) D {
					return room.Image.Layout(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{
						Axis: layout.Vertical,
					}.Layout(
						gtx,
						layout.Rigid(func(gtx C) D {
							return room.Name.Layout(gtx)
						}),
						layout.Rigid(func(gtx C) D {
							return room.Summary.Layout(gtx)
						}),
					)
				}),
				layout.Rigid(func(gtx C) D {
					return room.TimeStamp.Layout(gtx)
				}),
			)
		})
	})
}
