package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"git.sr.ht/~gioverse/chat/example/kitchen/appwidget"
	"git.sr.ht/~gioverse/chat/example/kitchen/appwidget/apptheme"
	"git.sr.ht/~gioverse/chat/example/kitchen/model"
	"git.sr.ht/~gioverse/chat/list"
	"git.sr.ht/~gioverse/chat/ninepatch"
	"git.sr.ht/~gioverse/chat/res"
	lorem "github.com/drhodes/golorem"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

var NavBack *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.NavigationArrowBack)
	return icon
}()

// UI manages the state for the entire application's UI.
type UI struct {
	// Rooms is the root of the data, containing messages chunked by
	// room.
	// It also contains interact state, rather than maintaining two
	// separate lists for the model and state.
	Rooms Rooms
	// RoomList for the sidebar.
	RoomList widget.List
	// RowsList for the active room messages.
	RowsList widget.List
	// Modal can show widgets atop the rest of the ui.
	Modal *component.ModalLayer
	// Bg is the background color of the content area.
	Bg color.NRGBA
	// Back button navigates out of a room.
	Back widget.Clickable
	// InsideRoom if we are currently in the room view.
	// Used to decide when to render the sidebar on small viewports.
	InsideRoom bool
}

// NewUI constructs a UI and populates it with dummy data.
func NewUI(w *app.Window) *UI {
	var ui UI

	// TODO(jfm) [modernize]: upstream a modernized modal implementation.
	// For now, use retained version.
	ui.Modal = component.NewModal()
	ui.Modal.FinalAlpha = 250

	ui.RowsList.ScrollToEnd = true
	ui.RowsList.Axis = layout.Vertical

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
				// TODO(jfm): sync this with something like `Room.Update()`.
				// Latest needs to update when the message model for the room
				/// changes.
				Latest: func() *model.Message {
					msg, ok := rt.Rows[len(rt.Rows)-1].(model.Message)
					if !ok {
						return nil
					}
					return &msg
				}(),
			},
			Messages: rt,
			List: list.NewManager(25,
				list.Hooks{
					// Define an allocator function that can instaniate the appropriate
					// state type for each kind of row data in our list.
					Allocator: func(data list.Element) interface{} {
						switch data.(type) {
						case model.Message:
							return &appwidget.Message{}
						default:
							return nil
						}
					},
					// Define a presenter that can transform each kind of row data
					// and state into a widget.
					Presenter: func(data list.Element, state interface{}) layout.Widget {
						switch data := data.(type) {
						case model.Message:
							state, ok := state.(*appwidget.Message)
							if !ok {
								return func(C) D { return D{} }
							}
							if state.Clicked() {
								ui.Modal.Widget = func(gtx C, th *material.Theme, anim *component.VisibilityAnimation) D {
									return layout.UniformInset(unit.Dp(25)).Layout(gtx, func(gtx C) D {
										return widget.Image{
											Src:      state.Image,
											Fit:      widget.ScaleDown,
											Position: layout.Center,
										}.Layout(gtx)
									})
								}
								// NOTE(jfm): don't have access to a gtx, so use click history.
								ui.Modal.Appear(state.Clickable.History()[0].Start)
							}
							msg := apptheme.NewMessage(th, state, data)
							switch data.Theme {
							case "hotdog":
								msg = msg.WithNinePatch(th, hotdog)
							case "cookie":
								msg = msg.WithNinePatch(th, cookie)
							}
							return msg.Layout
						case model.DateBoundary:
							return apptheme.DateSeparator(th.Theme, data).Layout
						case model.UnreadBoundary:
							return apptheme.UnreadSeparator(th.Theme, data).Layout
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

	// Configure a pleasing light gray background color.
	ui.Bg = color.NRGBA{220, 220, 220, 255}

	return &ui
}

// TODO(jfm): find proper place for this.
const (
	// sideBarMaxWidth species the max width on large viewports.
	sidebarMaxWidth = 250
	// breakpoint at which the viewport becomes considered "small",
	// and the UI layout changes to compensate.
	breakpoint = 600
)

// Layout the application UI.
func (ui *UI) Layout(gtx C) D {
	small := gtx.Constraints.Max.X < gtx.Px(unit.Dp(breakpoint))
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
						return material.List(th.Theme, &ui.RowsList).Layout(gtx,
							ui.Rooms.Active().List.UpdatedLen(&ui.RowsList.List),
							ui.Rooms.Active().List.Layout,
						)
					}),
					layout.Expanded(func(gtx C) D {
						return ui.layoutModal(gtx)
					}),
				)
			}),
		)
	} else {
		return layout.Flex{
			Axis: layout.Horizontal,
		}.Layout(
			gtx,
			layout.Rigid(func(gtx C) D {
				gtx.Constraints.Max.X = gtx.Px(unit.Dp(sidebarMaxWidth))
				gtx.Constraints.Min = gtx.Constraints.Constrain(gtx.Constraints.Min)
				return ui.layoutRoomList(gtx)
			}),
			layout.Flexed(1, func(gtx C) D {
				return layout.Stack{}.Layout(gtx,
					layout.Stacked(func(gtx C) D {
						gtx.Constraints.Min = gtx.Constraints.Max
						return material.List(th.Theme, &ui.RowsList).Layout(gtx,
							ui.Rooms.Active().List.UpdatedLen(&ui.RowsList.List),
							ui.Rooms.Active().List.Layout,
						)
					}),
					layout.Expanded(func(gtx C) D {
						return ui.layoutModal(gtx)
					}),
				)
			}),
		)
	}
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
				Color: th.Bg,
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
					return apptheme.Image{
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
				Color: th.Bg,
			}.Layout(gtx)
		}),
		layout.Stacked(func(gtx C) D {
			ui.RoomList.Axis = layout.Vertical
			return material.List(th.Theme, &ui.RoomList).Layout(gtx, len(ui.Rooms.List), func(gtx C, ii int) D {
				r := &ui.Rooms.List[ii]
				return apptheme.Room(th.Theme, &r.Interact, &r.Room).Layout(gtx)
			})
		}),
	)
}

// RowTracker is a stand-in for an application's data access logic.
// It stores a set of chat messages and can load them on request.
// It simulates network latency during the load operations for
// realism.
type RowTracker struct {
	Rows          []list.Element
	SerialToIndex map[list.Serial]int
}

// NewExampleData constructs a RowTracker populated with the provided
// quantity of messages.
func NewExampleData(size int) RowTracker {
	rt := RowTracker{
		SerialToIndex: make(map[list.Serial]int),
	}
	for i := 0; i < size; i++ {
		r := newRow(i)
		rt.SerialToIndex[r.Serial()] = i
		rt.Rows = append(rt.Rows, r)
	}
	return rt
}

// Load simulates loading chat history from a database or API. It
// sleeps for a random number of milliseconds and then returns
// some messages.
func (r RowTracker) Load(dir list.Direction, relativeTo list.Serial) []list.Element {
	duration := time.Millisecond * time.Duration(rand.Intn(1000))
	log.Println("sleeping", duration)
	time.Sleep(duration)
	numRows := len(r.Rows)
	if relativeTo == list.NoSerial {
		// If loading relative to nothing, likely the chat interface is empty.
		// We should load the most recent messages first in this case, regardless
		// of the direction parameter.
		return r.Rows[numRows-min(10, numRows):]
	}
	idx := r.SerialToIndex[relativeTo]
	if dir == list.After {
		return r.Rows[idx+1 : min(numRows, idx+11)]
	}
	return r.Rows[maximum(0, idx-11):idx]
}

func maximum(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// newRow returns a new synthetic row of chat data.
func newRow(serial int) list.Element {
	var rowData list.Element
	rowData = model.Message{
		SerialID: fmt.Sprintf("%05d", serial),
		Content:  lorem.Paragraph(1, 5),
		SentAt:   time.Now().Add(time.Hour * time.Duration(serial)),
		Sender:   lorem.Word(3, 10),
		Local:    serial%2 == 0,
		Status: func() string {
			if rand.Int()%10 == 0 {
				return apptheme.FailedToSend
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
	return rowData
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

func (ui *UI) layoutModal(gtx C) D {
	if ui.Modal.Clicked() {
		ui.Modal.ToggleVisibility(gtx.Now)
	}
	return ui.Modal.Layout(gtx, th.Theme)
}

// randomImage returns a random image at the given size.
// Downloads some number of random images from unplash and caches them on disk.
//
// TODO(jfm) [performance]: download images concurrently (parallel downloads,
// async to the gui event loop).
func randomImage(sz image.Point) (image.Image, error) {
	cache := filepath.Join(os.TempDir(), "chat", fmt.Sprintf("%dx%d", sz.X, sz.Y))
	if err := os.MkdirAll(cache, 0755); err != nil {
		return nil, fmt.Errorf("preparing cache directory: %w", err)
	}
	entries, err := ioutil.ReadDir(cache)
	if err != nil {
		return nil, fmt.Errorf("reading cache entries: %w", err)
	}
	entries = filter(entries, isFile)
	if len(entries) == 0 {
		for ii := 0; ii < 10; ii++ {
			ii := ii
			if err := func() error {
				r, err := http.Get(fmt.Sprintf("https://source.unsplash.com/random/%dx%d", sz.X, sz.Y))
				if err != nil {
					return fmt.Errorf("fetching image data: %w", err)
				}
				defer r.Body.Close()
				imgf, err := os.Create(filepath.Join(cache, strconv.Itoa(ii)))
				if err != nil {
					return fmt.Errorf("creating image file on disk: %w", err)
				}
				defer imgf.Close()
				if _, err := io.Copy(imgf, r.Body); err != nil {
					return fmt.Errorf("downloading image: %w", err)
				}
				return nil
			}(); err != nil {
				return nil, fmt.Errorf("populating image cache: %w", err)
			}
		}
		return randomImage(sz)
	}
	selection := entries[rand.Intn(len(entries))]
	imgf, err := os.Open(filepath.Join(cache, selection.Name()))
	if err != nil {
		return nil, fmt.Errorf("opening image file: %w", err)
	}
	defer imgf.Close()
	img, _, err := image.Decode(imgf)
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}
	return img, nil
}

// isFile filters out non-file entries.
func isFile(info fs.FileInfo) bool {
	return !info.IsDir()
}

func filter(list []fs.FileInfo, predicate func(fs.FileInfo) bool) (filtered []fs.FileInfo) {
	for _, item := range list {
		if predicate(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
