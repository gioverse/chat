package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math/rand"
	"strconv"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"

	"git.sr.ht/~gioverse/chat/example/kitchen/appwidget/apptheme"
	"git.sr.ht/~gioverse/chat/example/kitchen/model"
	chatlayout "git.sr.ht/~gioverse/chat/layout"
	"git.sr.ht/~gioverse/chat/list"
	"git.sr.ht/~gioverse/chat/ninepatch"
	"git.sr.ht/~gioverse/chat/res"
	chatwidget "git.sr.ht/~gioverse/chat/widget"
	matchat "git.sr.ht/~gioverse/chat/widget/material"

	lorem "github.com/drhodes/golorem"
)

var (
	// SidebarMaxWidth specifies how large the side bar should be on
	// desktop layouts.
	SidebarMaxWidth = unit.Dp(250)
	// Breakpoint at which to switch from desktop to mobile layout.
	Breakpoint = unit.Dp(600)
)

// UI manages the state for the entire application's UI.
type UI struct {
	// Rooms is the root of the data, containing messages chunked by
	// room.
	// It also contains interact state, rather than maintaining two
	// separate lists for the model and state.
	Rooms Rooms
	// RoomList for the sidebar.
	RoomList widget.List
	// Modal can show widgets atop the rest of the ui.
	Modal component.ModalState
	// Bg is the background color of the content area.
	Bg color.NRGBA
	// Back button navigates out of a room.
	Back widget.Clickable
	// InsideRoom if we are currently in the room view.
	// Used to decide when to render the sidebar on small viewports.
	InsideRoom bool

	// AddBtn holds click state for a button that adds a new message to
	// the current room.
	AddBtn widget.Clickable

	// DeleteBtn holds click state for a button that removes a message
	// from the current room.
	DeleteBtn widget.Clickable
	// MessageMenu is the context menu available on messages.
	MessageMenu component.MenuState
	// ContextMenuTarget tracks the message state on which the context
	// menu is currently acting.
	ContextMenuTarget *model.Message
}

// NewUI constructs a UI and populates it with dummy data.
func NewUI(w *app.Window) *UI {
	var ui UI

	ui.Modal.VisibilityAnimation.Duration = time.Millisecond * 250

	ui.MessageMenu = component.MenuState{
		Options: []func(gtx C) D{
			component.MenuItem(th.Theme, &ui.DeleteBtn, "Delete").Layout,
		},
	}

	var (
		cookie = func() ninepatch.NinePatch {
			imgf, err := res.Resources.Open("9-Patch/iap_platocookie_asset_2.png")
			if err != nil {
				panic(fmt.Errorf("opening image: %w", err))
			}
			defer imgf.Close()
			img, err := png.Decode(imgf)
			if err != nil {
				panic(fmt.Errorf("decoding png: %w", err))
			}
			return ninepatch.DecodeNinePatch(img)
		}()
		hotdog = func() ninepatch.NinePatch {
			imgf, err := res.Resources.Open("9-Patch/iap_hotdog_asset.png")
			if err != nil {
				panic(fmt.Errorf("opening image: %w", err))
			}
			defer imgf.Close()
			img, err := png.Decode(imgf)
			if err != nil {
				panic(fmt.Errorf("decoding png: %w", err))
			}
			return ninepatch.DecodeNinePatch(img)
		}()
	)

	for ii := rand.Intn(10) + 5; ii > 0; ii-- {
		rt := NewExampleData(100)
		ui.Rooms.List = append(ui.Rooms.List, Room{
			Room: model.Room{
				Name: lorem.Sentence(1, 5),
				Image: func() image.Image {
					img, err := randomImage(image.Pt(64, 64))
					if err != nil {
						panic(err)
					}
					return img
				}(),
			},
			Messages: rt,
			ListState: list.NewManager(25,
				list.Hooks{
					// Define an allocator function that can instaniate the appropriate
					// state type for each kind of row data in our list.
					Allocator: func(data list.Element) interface{} {
						switch data.(type) {
						case model.Message:
							return &chatwidget.Row{}
						default:
							return nil
						}
					},
					// Define a presenter that can transform each kind of row data
					// and state into a widget.
					Presenter: func(data list.Element, state interface{}) layout.Widget {
						switch data := data.(type) {
						case model.Message:
							state, ok := state.(*chatwidget.Row)
							if !ok {
								return func(C) D { return D{} }
							}
							msg := matchat.NewRow(th.Theme, state, &ui.MessageMenu, FromModel(data))
							switch data.Theme {
							case "hotdog":
								msg.MessageStyle = msg.WithNinePatch(th.Theme, hotdog)
							case "cookie":
								msg.MessageStyle = msg.WithNinePatch(th.Theme, cookie)
							default:
								uc := th.LocalUserColor()
								if !msg.Local {
									uc = th.UserColor(msg.Username.Text)
								}
								msg.MessageStyle.BubbleStyle.Color = uc.NRGBA
								for i := range msg.Content.Styles {
									msg.Content.Styles[i].Color = th.Contrast(uc.Luminance)
								}
							}
							return func(gtx C) D {
								if state.Clicked() {
									ui.Modal.Show(gtx.Now, func(gtx C) D {
										return layout.UniformInset(unit.Dp(25)).Layout(gtx, func(gtx C) D {
											return widget.Image{
												Src:      state.Image.Op(),
												Fit:      widget.ScaleDown,
												Position: layout.Center,
											}.Layout(gtx)
										})
									})
								}
								if state.ContextArea.Active() {
									// If the right-click context area for this message is activated,
									// inform the UI that this message is the target of any action
									// taken within that menu.
									ui.ContextMenuTarget = &data
								}
								return msg.Layout(gtx)
							}
						case model.DateBoundary:
							return matchat.DateSeparator(th.Theme, data.Date).Layout
						case model.UnreadBoundary:
							return matchat.UnreadSeparator(th.Theme).Layout
						default:
							return func(gtx C) D { return D{} }
						}
					},
					// NOTE(jfm): awkard coupling between message data and `list.Manager`.
					Loader:      rt.Load,
					Synthesizer: synth,
					Comparator:  rowLessThan,
					Invalidator: w.Invalidate,
				},
			),
		})
	}

	ui.Rooms.Select(0)
	for ii := range ui.Rooms.List {
		ui.Rooms.List[ii].List.ScrollToEnd = true
		ui.Rooms.List[ii].List.Axis = layout.Vertical
	}

	ui.Bg = th.Palette.Bg

	return &ui
}

// Layout the application UI.
func (ui *UI) Layout(gtx C) D {
	small := gtx.Constraints.Max.X < gtx.Px(Breakpoint)
	for ii := range ui.Rooms.List {
		r := &ui.Rooms.List[ii]
		if r.Interact.Clicked() {
			ui.Rooms.Select(ii)
			ui.InsideRoom = true
			break
		}
	}
	if ui.Back.Clicked() {
		ui.InsideRoom = false
	}
	paint.Fill(gtx.Ops, ui.Bg)
	if small {
		if !ui.InsideRoom {
			return ui.layoutRoomList(gtx)
		}
		return layout.Flex{
			Axis: layout.Vertical,
		}.Layout(
			gtx,
			layout.Rigid(func(gtx C) D {
				return ui.layoutTopbar(gtx)
			}),
			layout.Flexed(1, func(gtx C) D {
				return layout.Stack{}.Layout(gtx,
					layout.Stacked(func(gtx C) D {
						gtx.Constraints.Min = gtx.Constraints.Max
						return ui.layoutChat(gtx)
					}),
					layout.Expanded(func(gtx C) D {
						return ui.layoutModal(gtx)
					}),
				)
			}),
		)
	}
	return layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(
		gtx,
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Max.X = gtx.Px(SidebarMaxWidth)
			gtx.Constraints.Min = gtx.Constraints.Constrain(gtx.Constraints.Min)
			return ui.layoutRoomList(gtx)
		}),
		layout.Flexed(1, func(gtx C) D {
			return layout.Stack{}.Layout(gtx,
				layout.Stacked(func(gtx C) D {
					gtx.Constraints.Min = gtx.Constraints.Max
					return ui.layoutChat(gtx)
				}),
				layout.Expanded(func(gtx C) D {
					return ui.layoutModal(gtx)
				}),
			)
		}),
	)
}

// layoutChat lays out the chat interface with associated controls.
func (ui *UI) layoutChat(gtx C) D {
	room := ui.Rooms.Active()
	var (
		scrollWidth unit.Value
		list        = &room.List
		state       = room.ListState
	)
	listStyle := material.List(th.Theme, list)
	scrollWidth = listStyle.ScrollbarStyle.Width(gtx.Metric)
	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Flexed(1, func(gtx C) D {
			return listStyle.Layout(gtx,
				state.UpdatedLen(&list.List),
				state.Layout,
			)
		}),
		layout.Rigid(func(gtx C) D {
			return chatlayout.Background(th.Palette.BgSecondary).Layout(gtx, func(gtx C) D {
				if ui.AddBtn.Clicked() {
					ui.Rooms.Active().SendMessage()
				}
				if ui.DeleteBtn.Clicked() {
					serial := ui.ContextMenuTarget.Serial()
					ui.Rooms.Active().DeleteRow(serial)
				}
				return layout.Inset{
					Bottom: unit.Dp(8),
					Top:    unit.Dp(8),
				}.Layout(gtx, func(gtx C) D {
					gutter := chatlayout.Gutter()
					gutter.RightWidth = unit.Add(gtx.Metric, gutter.RightWidth, scrollWidth)
					return gutter.Layout(gtx,
						nil,
						func(gtx C) D {
							return ui.layoutEditor(gtx)
						},
						material.IconButton(th.Theme, &ui.AddBtn, Send).Layout,
					)
				})
			})
		}),
	)
}

// layoutTopbar lays out a context bar that contains a "back" button and
// room title for context.
func (ui *UI) layoutTopbar(gtx C) D {
	room := ui.Rooms.Active()
	return layout.Stack{}.Layout(
		gtx,
		layout.Expanded(func(gtx C) D {
			return component.Rect{
				Size: image.Point{
					X: gtx.Constraints.Max.X,
					Y: gtx.Constraints.Min.Y,
				},
				Color: th.Palette.Surface,
			}.Layout(gtx)
		}),
		layout.Stacked(func(gtx C) D {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
			}.Layout(
				gtx,
				layout.Rigid(func(gtx C) D {
					btn := material.IconButton(th.Theme, &ui.Back, NavBack)
					btn.Color = th.Fg
					btn.Background = color.NRGBA{}
					return btn.Layout(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					return matchat.Image{
						Image: widget.Image{
							Src: room.Interact.Image.Op(),
						},
						Width:  unit.Dp(24),
						Height: unit.Dp(24),
					}.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx C) D {
					return material.Label(th.Theme, unit.Sp(14), room.Name).Layout(gtx)
				}),
			)
		}),
	)
}

// layoutRoomList lays out a list of rooms that can be clicked to view
// the messages in that room.
func (ui *UI) layoutRoomList(gtx C) D {
	return layout.Stack{}.Layout(
		gtx,
		layout.Expanded(func(gtx C) D {
			return component.Rect{
				Size: image.Point{
					X: gtx.Constraints.Min.X,
					Y: gtx.Constraints.Max.Y,
				},
				Color: th.Palette.Surface,
			}.Layout(gtx)
		}),
		layout.Stacked(func(gtx C) D {
			ui.RoomList.Axis = layout.Vertical
			gtx.Constraints.Min = gtx.Constraints.Max
			return material.List(th.Theme, &ui.RoomList).Layout(gtx, len(ui.Rooms.List), func(gtx C, ii int) D {
				r := &ui.Rooms.List[ii]
				return apptheme.Room(th.Theme, &r.Interact, &r.Room).Layout(gtx)
			})
		}),
	)
}

// layoutEditor lays out the message editor.
func (ui *UI) layoutEditor(gtx C) D {
	return chatlayout.Rounded(unit.Dp(8)).Layout(gtx, func(gtx C) D {
		return chatlayout.Background(th.Palette.Surface).Layout(gtx, func(gtx C) D {
			return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx C) D {
				return material.Editor(th.Theme, &ui.Rooms.Active().Editor, "Send a message").Layout(gtx)
			})
		})
	})
}

func (ui *UI) layoutModal(gtx C) D {
	if ui.Modal.Clicked() {
		ui.Modal.ToggleVisibility(gtx.Now)
	}
	// NOTE(jfm): scrim should be dark regardless of theme.
	// Perhaps "scrim color" could be specified on the theme.
	t := *th.Theme
	t.Fg = apptheme.Dark.Surface
	return component.Modal(&t, &ui.Modal).Layout(gtx)
}

// FromModel converts a domain-specific model of a chat message into
// the general-purpose MessageConfig.
func FromModel(m model.Message) matchat.RowConfig {
	return matchat.RowConfig{
		Sender:  m.Sender,
		Avatar:  m.Avatar,
		Content: m.Content,
		SentAt:  m.SentAt,
		Image:   m.Image,
		Local:   m.Local,
		Status:  m.Status,
	}
}

// newRow returns a new synthetic row of chat data.
func newRow(serial int) list.Element {
	return model.Message{
		SerialID: fmt.Sprintf("%05d", serial),
		Content:  lorem.Paragraph(1, 5),
		SentAt:   time.Now().Add(time.Hour * time.Duration(serial)),
		Sender:   lorem.Word(3, 10),
		Local:    serial%2 == 0,
		Status: func() string {
			if rand.Int()%10 == 0 {
				return matchat.FailedToSend
			}
			return ""
		}(),
		Theme: func() string {
			switch val := rand.Intn(10); val {
			case 0:
				return "cookie"
			case 1:
				return "hotdog"
			default:
				return ""
			}
		}(),
		Image: func() image.Image {
			if rand.Float32() < 0.7 {
				return nil
			}
			sizes := []image.Point{
				image.Pt(1792, 828),
				image.Pt(828, 1792),
				image.Pt(600, 600),
				image.Pt(300, 300),
			}
			img, err := randomImage(sizes[rand.Intn(len(sizes))])
			if err != nil {
				log.Print(err)
				return nil
			}
			return img
		}(),
		Avatar: func() image.Image {
			sizes := []image.Point{
				image.Pt(64, 64),
				image.Pt(32, 32),
				image.Pt(24, 24),
			}
			img, err := randomImage(sizes[rand.Intn(len(sizes))])
			if err != nil {
				log.Print(err)
				return nil
			}
			return img
		}(),
		Read: func() bool {
			return serial < 95
		}(),
	}
}

// synth inserts date separators and unread separators
// between chat rows as a list.Synthesizer.
func synth(previous, row list.Element) []list.Element {
	var out []list.Element
	asMessage, ok := row.(model.Message)
	if !ok {
		out = append(out, row)
		return out
	}
	if previous == nil {
		if !asMessage.Read {
			out = append(out, model.UnreadBoundary{})
		}
		out = append(out, row)
		return out
	}
	lastMessage, ok := previous.(model.Message)
	if !ok {
		out = append(out, row)
		return out
	}
	if !asMessage.Read && lastMessage.Read {
		out = append(out, model.UnreadBoundary{})
	}
	y, m, d := asMessage.SentAt.Local().Date()
	yy, mm, dd := lastMessage.SentAt.Local().Date()
	if y == yy && m == mm && d == dd {
		out = append(out, row)
		return out
	}
	out = append(out, model.DateBoundary{Date: asMessage.SentAt}, row)
	return out
}

// rowLessThan acts as a list.Comparator, returning whether a sorts before b.
func rowLessThan(a, b list.Element) bool {
	aID := string(a.Serial())
	bID := string(b.Serial())
	aAsInt, _ := strconv.Atoi(aID)
	bAsInt, _ := strconv.Atoi(bID)
	return aAsInt < bAsInt
}
