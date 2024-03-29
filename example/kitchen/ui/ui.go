package ui

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"strconv"
	"time"

	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	lorem "github.com/drhodes/golorem"

	"git.sr.ht/~gioverse/chat/async"
	"git.sr.ht/~gioverse/chat/example/kitchen/appwidget/apptheme"
	"git.sr.ht/~gioverse/chat/example/kitchen/gen"
	"git.sr.ht/~gioverse/chat/example/kitchen/model"
	chatlayout "git.sr.ht/~gioverse/chat/layout"
	"git.sr.ht/~gioverse/chat/list"
	"git.sr.ht/~gioverse/chat/ninepatch"
	"git.sr.ht/~gioverse/chat/res"
	chatwidget "git.sr.ht/~gioverse/chat/widget"
	matchat "git.sr.ht/~gioverse/chat/widget/material"
	"git.sr.ht/~gioverse/chat/widget/plato"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

type Config struct {
	// theme to use {light,dark}.
	Theme string
	// usePlato to use plato themed widgets.
	UsePlato bool
	// latency specifies maximum latency (in millis) to simulate
	Latency int
	// loadSize specifies maximum number of items to load at a time.
	LoadSize int
	// bufferSize specifies how many elements to hold in memory before
	// compacting the list.
	BufferSize int
}

// th is the active theme object.
var (
	fonts = gofont.Collection()
	th    = apptheme.NewTheme(fonts)
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
	// Loader loads resources asynchronously.
	// Deallocates stale resources.
	// Stale is defined as "not being scheduled frequently".
	async.Loader
	// Rooms is the root of the data, containing messages chunked by
	// room.
	// It also contains interact state, rather than maintaining two
	// separate lists for the model and state.
	Rooms Rooms
	// Local user for this client.
	Local *model.User
	// Users contains user data.
	Users *model.Users
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

	usePlato bool
}

// loadNinePatch from the embedded resources package.
func loadNinePatch(path string) ninepatch.NinePatch {
	imgf, err := res.Resources.Open(path)
	if err != nil {
		panic(fmt.Errorf("opening image: %w", err))
	}
	defer imgf.Close()
	img, err := png.Decode(imgf)
	if err != nil {
		panic(fmt.Errorf("decoding png: %w", err))
	}
	return ninepatch.DecodeNinePatch(img)
}

var (
	cookie = loadNinePatch("9-Patch/iap_platocookie_asset_2.png")
	hotdog = loadNinePatch("9-Patch/iap_hotdog_asset.png")
)

// NewUI constructs a UI and populates it with dummy data.
func NewUI(invalidator func(), conf Config) *UI {
	var ui UI

	ui.usePlato = conf.UsePlato

	switch conf.Theme {
	case "light":
		th.UsePalette(apptheme.Light)
	case "dark":
		th.UsePalette(apptheme.Dark)
	}

	ui.Modal.VisibilityAnimation.Duration = time.Millisecond * 250

	ui.MessageMenu = component.MenuState{
		Options: []func(gtx C) D{
			component.MenuItem(th.Theme, &ui.DeleteBtn, "Delete").Layout,
		},
	}

	g := &gen.Generator{
		FetchImage: func(sz image.Point) image.Image {
			img, _ := randomImage(sz)
			return img
		},
	}

	// Generate most of the model data.
	var (
		rooms = g.GenRooms(3, 10)
		users = g.GenUsers(10, 30)
		local = users.Random()
	)

	ui.Users = users
	ui.Local = local

	for _, r := range rooms.List() {
		rt := NewExampleData(users, local, g, 100)
		rt.SimulateLatency = conf.Latency
		rt.MaxLoads = conf.LoadSize
		rt.ScrollToEnd = false
		lm := list.NewManager(conf.BufferSize,
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
				Presenter: ui.presentChatRow,
				// NOTE(jfm): awkard coupling between message data and `list.Manager`.
				Loader:      rt.Load,
				Synthesizer: synth,
				Comparator:  rowLessThan,
				Invalidator: invalidator,
			},
		)
		lm.Stickiness = list.After
		ui.Rooms.List = append(ui.Rooms.List, Room{
			Room:      r,
			Messages:  rt,
			ListState: lm,
		})
	}

	// spin up a bunch of async actors to send messages to rooms.
	for _, u := range users.List() {
		u := u
		if u.Name == local.Name {
			continue
		}
		go func() {
			for {
				var (
					respond = time.Second * time.Duration(1)
					compose = time.Second * time.Duration(1)
					room    = ui.Rooms.Random()
				)
				func() {
					time.Sleep(respond)
					room.SetComposing(u.Name, true)
					time.Sleep(compose)
					room.SetComposing(u.Name, false)
					room.Send(u.Name, lorem.Paragraph(1, 4))
				}()
			}
		}()
	}

	ui.Rooms.Select(0)
	for ii := range ui.Rooms.List {
		ui.Rooms.List[ii].List.ScrollToEnd = ui.Rooms.List[ii].Messages.ScrollToEnd
		ui.Rooms.List[ii].List.Axis = layout.Vertical
	}

	ui.Bg = th.Palette.Bg

	return &ui
}

// Layout the application UI.
func (ui *UI) Layout(gtx C) D {
	return ui.Loader.Frame(gtx, ui.layout)
}

func (ui *UI) layout(gtx C) D {
	small := gtx.Constraints.Max.X < gtx.Dp(Breakpoint)
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
	paint.FillShape(gtx.Ops, ui.Bg, clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Op())
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
			gtx.Constraints.Max.X = gtx.Dp(SidebarMaxWidth)
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
		scrollWidth unit.Dp
		list        = &room.List
		state       = room.ListState
	)
	listStyle := material.List(th.Theme, list)
	scrollWidth = listStyle.ScrollbarStyle.Width()
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
					active := ui.Rooms.Active()
					active.SendLocal(active.Editor.Text())
					active.Editor.SetText("")
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
					gutter.RightWidth = gutter.RightWidth + scrollWidth
					return gutter.Layout(gtx,
						nil,
						func(gtx C) D {
							return ui.layoutEditor(gtx)
						},
						material.IconButton(th.Theme, &ui.AddBtn, Send, "Send").Layout,
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
					btn := material.IconButton(th.Theme, &ui.Back, NavBack, "Back")
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
				r := ui.Rooms.Index(ii)
				latest := r.Latest()
				return apptheme.Room(th.Theme, &r.Interact, &apptheme.RoomConfig{
					Name:    r.Room.Name,
					Image:   r.Room.Image,
					Content: latest.Content,
					SentAt:  latest.SentAt,
				}).Layout(gtx)
			})
		}),
	)
}

// layoutEditor lays out the message editor.
func (ui *UI) layoutEditor(gtx C) D {
	return chatlayout.Rounded(unit.Dp(8)).Layout(gtx, func(gtx C) D {
		return chatlayout.Background(th.Palette.Surface).Layout(gtx, func(gtx C) D {
			return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx C) D {
				active := ui.Rooms.Active()
				editor := &active.Editor
				for _, e := range editor.Events() {
					switch e.(type) {
					case widget.SubmitEvent:
						active.SendLocal(editor.Text())
						editor.SetText("")
					}
				}
				editor.Submit = true
				editor.SingleLine = true
				return material.Editor(th.Theme, editor, "Send a message").Layout(gtx)
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

// synth inserts date separators and unread separators
// between chat rows as a list.Synthesizer.
func synth(previous, row, next list.Element) []list.Element {
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

// presentChatRow returns a widget closure that can layout the given chat item.
// `data` contains managed data for this chat item, `state` contains UI defined
// interactive state.
func (ui *UI) presentChatRow(data list.Element, state interface{}) layout.Widget {
	switch data := data.(type) {
	case model.Message:
		state, ok := state.(*chatwidget.Row)
		if !ok {
			return func(C) D { return D{} }
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
			return ui.row(ui.usePlato, data, state)(gtx)
		}
	case model.DateBoundary:
		return matchat.DateSeparator(th.Theme, data.Date).Layout
	case model.UnreadBoundary:
		return matchat.UnreadSeparator(th.Theme).Layout
	default:
		return func(gtx C) D { return D{} }
	}
}

// row returns either a plato.RowStyle or a chatmaterial.RowStyle based on the
// provided boolean.
func (ui *UI) row(usePlato bool, data model.Message, state *chatwidget.Row) layout.Widget {
	user, ok := ui.Users.Lookup(data.Sender)
	if !ok {
		return func(C) D { return D{} }
	}
	np := func() *ninepatch.NinePatch {
		switch user.Theme {
		case model.ThemeHotdog:
			return &hotdog
		case model.ThemePlatoCookie:
			return &cookie
		}
		return nil
	}()
	var (
		avatar image.Image
		body   image.Image
	)
	if data.Avatar != "" {
		avatar = avatarPlaceholder
		if img := loadImage(string(data.Serial())+"-avatar", data.Avatar, &ui.Loader); img != nil {
			state.Avatar.Reload()
			avatar = img
		}
	}
	if data.Image != "" {
		body = imageMessagePlaceholder
		if img := loadImage(string(data.Serial())+"-body", data.Image, &ui.Loader); img != nil {
			state.Image.Reload()
			body = img
		}
	}
	if usePlato {
		msg := plato.NewRow(th.Theme, state, &ui.MessageMenu, plato.RowConfig{
			Sender:  data.Sender,
			Content: data.Content,
			Avatar:  avatar,
			SentAt:  data.SentAt,
			Local:   user.Name == ui.Local.Name,
		})
		if np != nil {
			msg.MessageStyle = msg.WithNinePatch(th.Theme, *np)
			if cl, ok := np.Image.At(np.Bounds().Dx()/2, np.Bounds().Dy()/2).(color.NRGBA); ok {
				msg.TextColor(th.Contrast(matchat.Luminance(cl)))
			}
		} else {
			msg.TextColor(th.Contrast(matchat.Luminance(msg.BubbleStyle.Color)))
		}
		return msg.Layout
	}
	msg := matchat.NewRow(th.Theme, state, &ui.MessageMenu, matchat.RowConfig{
		Sender:  data.Sender,
		Content: data.Content,
		SentAt:  data.SentAt,
		Avatar:  avatar,
		Image:   body,
		Local:   user.Name == ui.Local.Name,
	})
	if np != nil {
		msg.MessageStyle = msg.WithNinePatch(th.Theme, *np)
	}
	msg.MessageStyle.BubbleStyle.Color = user.Color
	for i := range msg.Content.Styles {
		msg.Content.Styles[i].Color = th.Contrast(matchat.Luminance(user.Color))
	}
	return msg.Layout
}

var (
	// placeholderColor to use for placeholder images.
	placeholderColor = color.NRGBA{R: 50, G: 50, B: 50, A: 255}
	// avatarPlaceholder used when avatar image has not been loaded yet.
	avatarPlaceholder *image.NRGBA = placeholder(image.Pt(64, 64), placeholderColor)
	// imageMessagePlaceholder used when message image has not been loaded yet.
	imageMessagePlaceholder *image.NRGBA = placeholder(image.Pt(320, 320), placeholderColor)
)

// placeholder helper generates a rectangle image of the given size for the
// given color.
func placeholder(sz image.Point, c color.NRGBA) (ph *image.NRGBA) {
	ph = image.NewNRGBA(image.Rectangle{Max: sz})
	for xx := ph.Bounds().Min.X; xx < ph.Bounds().Max.X; xx++ {
		for yy := ph.Bounds().Min.Y; yy < ph.Bounds().Max.Y; yy++ {
			ph.SetNRGBA(xx, yy, c)
		}
	}
	return ph
}

// loadImage helper schedules an image to be downloaded and returns it if ready.
func loadImage(id, u string, l *async.Loader) image.Image {
	r := l.Schedule(id, func(_ context.Context) interface{} {
		img, err := fetch(id, u)
		if err != nil {
			log.Printf("loading image: %v", err)
		}
		return img
	})
	switch r.State {
	case async.Queued, async.Loading:
	case async.Loaded:
		if img, ok := r.Value.(image.Image); ok {
			return img
		}
	}
	return nil
}
