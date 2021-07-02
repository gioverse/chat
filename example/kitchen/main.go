// Package kitchen demonstrates the various chat components and features.
package main

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/richtext"
	lorem "github.com/drhodes/golorem"
	colorful "github.com/lucasb-eyer/go-colorful"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

// ToNRGBA converts a colorful.Color to the nearest representable color.NRGBA.
func ToNRGBA(c colorful.Color) color.NRGBA {
	r, g, b, a := c.RGBA()
	return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
}

// ErrorIcon is the material design outlined error indicator.
var ErrorIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.AlertErrorOutline)
	return icon
}()

// FailedToSend is the message that is displayed to the user when there was a
// problem sending a chat message.
const FailedToSend = "Sending failed"

func main() {
	var (
		// Instantiate the chat window.
		w = app.NewWindow(
			app.Title("Chat"),
			app.Size(unit.Dp(800), unit.Dp(600)),
		)
		// Define an operation list for gio.
		ops op.Ops
		// Instantiate our UI state.
		ui = NewUI()
	)

	go func() {
		// Event loop executes indefinitely, until the app is signalled to quit.
		// Integrate external services here.
		for event := range w.Events() {
			switch event := event.(type) {
			case system.DestroyEvent:
				if err := event.Err; err != nil {
					fmt.Printf("error: premature window close: %v\n", err)
					os.Exit(1)
				}
				os.Exit(0)
			case system.FrameEvent:
				ui.Layout(layout.NewContext(&ops, event))
				event.Frame(&ops)
			}
		}
	}()
	// Surrender main thread to OS.
	// Necessary for certain platforms.
	app.Main()
}

// Type alias common layout types for legibility.
type (
	C = layout.Context
	D = layout.Dimensions
)

// th is the active theme object.
var (
	fonts = gofont.Collection()
	th    = NewTheme(fonts)
)

// Message represents a chat message.
type Message struct {
	Serial                  string
	Sender, Content, Status string
	SentAt                  time.Time
	Local                   bool
}

// ID returns the unique identifier for this message.
func (m Message) ID() MessageID {
	return MessageID(m.Serial)
}

// MessageInteraction holds the state necessary to facilitate user
// interactions with messages across frames.
type MessageInteraction struct {
	richtext.InteractiveText
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

// DateBoundary represents a change in the date during a chat.
type DateBoundary struct {
	Date time.Time
}

// ID returns the unique ID of the message.
func (d DateBoundary) ID() MessageID {
	return NoID
}

// UnreadBoundary represents the boundary between the last read message
// in a chat and the next unread message.
type UnreadBoundary struct{}

// ID returns the unique identifier for the boundary.
func (u UnreadBoundary) ID() MessageID {
	return NoID
}

// SeparatorStyle configures the presentation of the unread indicator.
type SeparatorStyle struct {
	Message    material.LabelStyle
	TextMargin layout.Inset
	LineMargin layout.Inset
	LineWidth  unit.Value
}

// UnreadSeparator fills in a SeparatorStyle with sensible defaults.
func UnreadSeparator(th *material.Theme, ub UnreadBoundary) SeparatorStyle {
	us := SeparatorStyle{
		Message:    material.Body1(th, "New Messages"),
		TextMargin: layout.UniformInset(unit.Dp(8)),
		LineMargin: layout.UniformInset(unit.Dp(8)),
		LineWidth:  unit.Dp(2),
	}
	us.Message.Color = th.ContrastBg
	return us
}

// DateSeparator makes a SeparatorStyle with reasonable defaults out of
// the provided DateBoundary.
func DateSeparator(th *material.Theme, db DateBoundary) SeparatorStyle {
	return SeparatorStyle{
		Message:    material.Body1(th, db.Date.Format("Mon Jan 2, 2006")),
		TextMargin: layout.UniformInset(unit.Dp(8)),
		LineMargin: layout.UniformInset(unit.Dp(8)),
		LineWidth:  unit.Dp(2),
	}
}

// Layout the Separator.
func (u SeparatorStyle) Layout(gtx C) D {
	layoutLine := func(gtx C) D {
		return u.LineMargin.Layout(gtx, func(gtx C) D {
			size := image.Point{
				X: gtx.Constraints.Max.X,
				Y: gtx.Px(u.LineWidth),
			}
			paint.FillShape(gtx.Ops, u.Message.Color, clip.Rect(image.Rectangle{Max: size}).Op())
			return D{Size: size}
		})
	}
	return layout.Flex{
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Flexed(.5, layoutLine),
		layout.Rigid(func(gtx C) D {
			return u.TextMargin.Layout(gtx, u.Message.Layout)
		}),
		layout.Flexed(.5, layoutLine),
	)
}

// MessageStyle configures the presentation of a chat message within
// a vertical list of chat messages.
type MessageStyle struct {
	// Local indicates that the message was sent by the local user,
	// and should be left-aligned.
	Local bool
	// Time is the timestamp associated with the message.
	Time material.LabelStyle
	// StatusIcon is an optional icon that will be displayed to the right of
	// the message instead of its timestamp.
	StatusIcon *widget.Icon
	// IconSize defines the size of the StatusIcon (if it is set).
	IconSize unit.Value
	// RightGutterPadding defines the size of the area to the right of the message
	// reserved for the timestamp and/or icon.
	RightGutterPadding layout.Inset
	// StatusMessage defines a warning message to be displayed beneath the
	// chat message.
	StatusMessage material.LabelStyle
	// Bubble configures the background bubble of the chat.
	Bubble BubbleStyle
	// BubbleMargin configures space around the chat bubble.
	BubbleMargin layout.Inset
	// Content configures the actual contents of the chat bubble.
	Content richtext.TextStyle
	// ContentPadding defines space around the Content within the Bubble area.
	ContentPadding layout.Inset
	// LeftGutter defines the size of the empty left gutter of the row.
	LeftGutter layout.Spacer
}

// NewMessage creates a style type that can lay out the data for a message.
func NewMessage(th *Theme, interact *MessageInteraction, msg Message) MessageStyle {
	ms := MessageStyle{
		Time:   material.Body2(th.Theme, msg.SentAt.Local().Format("15:04")),
		Bubble: Bubble(th.Theme),
		Content: richtext.Text(&interact.InteractiveText, th.Shaper, richtext.SpanStyle{
			Font:    fonts[0].Font,
			Size:    material.Body1(th.Theme, "").TextSize,
			Color:   th.Fg,
			Content: msg.Content,
		}),
		Local:              msg.Local,
		IconSize:           unit.Dp(32),
		RightGutterPadding: layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12)},
		ContentPadding:     layout.UniformInset(unit.Dp(8)),
		BubbleMargin:       layout.UniformInset(unit.Dp(8)),
		LeftGutter:         layout.Spacer{Width: unit.Dp(24)},
	}
	if msg.Status != "" {
		ms.StatusMessage = material.Body2(th.Theme, msg.Status)
		ms.StatusMessage.Color = th.DangerColor
		ms.StatusIcon = ErrorIcon
		ms.StatusIcon.Color = th.DangerColor
	}
	if !ms.Local {
		userColors := th.UserColor(msg.Sender)
		ms.Bubble.Color = userColors.NRGBA
		if userColors.Luminance < .5 {
			for i := range ms.Content.Styles {
				ms.Content.Styles[i].Color = th.Theme.Bg
			}
		}
	}
	return ms
}

// Layout the message.
func (c MessageStyle) Layout(gtx C) D {
	messageAlignment := layout.W
	if c.Local {
		messageAlignment = layout.E
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Flex{
				Alignment: layout.Middle,
			}.Layout(gtx,
				layout.Rigid(c.LeftGutter.Layout),
				layout.Flexed(1, func(gtx C) D {
					return messageAlignment.Layout(gtx, c.layoutBubble)
				}),
				layout.Rigid(c.layoutTimeOrIcon),
			)
		}),
		layout.Rigid(func(gtx C) D {
			if c.StatusMessage.Text == "" {
				return D{}
			}
			return layout.E.Layout(gtx, func(gtx C) D {
				return c.RightGutterPadding.Layout(gtx, c.StatusMessage.Layout)
			})
		}),
	)
}

// layoutBubble lays out the chat bubble.
func (c MessageStyle) layoutBubble(gtx C) D {
	gtx.Constraints.Max.X = int(float32(gtx.Constraints.Max.X) * 0.8)
	maxSaneSize := gtx.Px(unit.Dp(600))
	if gtx.Constraints.Max.X > maxSaneSize {
		gtx.Constraints.Max.X = maxSaneSize
	}
	return c.BubbleMargin.Layout(gtx, func(gtx C) D {
		return c.Bubble.Layout(gtx, func(gtx C) D {
			return c.ContentPadding.Layout(gtx, c.Content.Layout)
		})
	})
}

// layoutTimeOrIcon lays out a status icon if one is set, and
// otherwise lays out the time the messages was sent.
func (c MessageStyle) layoutTimeOrIcon(gtx C) D {
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx C) D {
			return c.RightGutterPadding.Layout(gtx, func(gtx C) D {
				sideLength := gtx.Px(c.IconSize)
				gtx.Constraints.Max.X = sideLength
				gtx.Constraints.Max.Y = sideLength
				gtx.Constraints.Min = gtx.Constraints.Constrain(gtx.Constraints.Min)
				if c.StatusIcon != nil {
					return c.StatusIcon.Layout(gtx)
				}
				return D{Size: gtx.Constraints.Max}
			})
		}),
		layout.Expanded(func(gtx C) D {
			if c.StatusIcon != nil {
				return D{}
			}
			return layout.Center.Layout(gtx, c.Time.Layout)
		}),
	)
}

// UI manages the state for the entire application's UI.
type UI struct {
	RowsList layout.List
	*MessageManager
	Bg color.NRGBA
}

func NewUI() *UI {
	var ui UI

	ui.MessageManager = NewManager(
		// Define an allocator function that can instaniate the appropriate
		// state type for each kind of message data in our list.
		func(data MessageData) interface{} {
			switch data.(type) {
			case Message:
				return &MessageInteraction{}
			default:
				return nil
			}
		},
		// Define a presenter that can transform each kind of message data
		// and state into a widget.
		func(data MessageData, state interface{}) layout.Widget {
			switch data := data.(type) {
			case Message:
				return NewMessage(th, state.(*MessageInteraction), data).Layout
			case DateBoundary:
				return DateSeparator(th.Theme, data).Layout
			case UnreadBoundary:
				return UnreadSeparator(th.Theme, data).Layout
			default:
				return func(gtx C) D { return D{} }
			}
		})

	// Configure a pleasing light gray background color.
	ui.Bg = color.NRGBA{220, 220, 220, 255}

	// Populate the UI with dummy random messages.
	max := 100
	for i := 0; i < max; i++ {
		var rowData MessageData
		if i%10 == 0 {
			rowData = DateBoundary{Date: time.Now().Add(time.Hour * 24 * time.Duration(-(100 - i)))}
		} else if i == 5 {
			rowData = UnreadBoundary{}
		} else {
			rowData = Message{
				Serial:  fmt.Sprintf("%d", i),
				Content: lorem.Paragraph(1, 5),
				SentAt:  time.Now().Add(time.Minute * time.Duration(-(100 - i))),
				Sender:  lorem.Word(3, 10),
				Local:   i%2 == 0,
				Status: func() string {
					if rand.Int()%10 == 0 {
						return FailedToSend
					}
					return ""
				}(),
			}
		}
		ui.MessageManager.Messages = append(ui.MessageManager.Messages, rowData)
	}

	return &ui
}

// Layout the application UI.
func (ui *UI) Layout(gtx C) D {
	paint.Fill(gtx.Ops, ui.Bg)
	ui.RowsList.Axis = layout.Vertical
	return ui.RowsList.Layout(gtx, ui.MessageManager.Len(), ui.MessageManager.Layout)
}

// BubbleStyle defines the visual aspects of a material design surface
// with (optionally) rounded corners and a drop shadow.
type BubbleStyle struct {
	// The radius of the corners of the rectangle casting the surface.
	// Non-rounded rectangles can just provide a zero.
	CornerRadius unit.Value
	Color        color.NRGBA
}

// Bubble creates a Bubble style for the provided theme with the theme
// background color and rounded corners.
func Bubble(th *material.Theme) BubbleStyle {
	return BubbleStyle{
		CornerRadius: unit.Dp(8),
		Color:        th.Bg,
	}
}

// Layout renders the BubbleStyle, taking the dimensions of the surface from
// gtx.Constraints.Min.
func (c BubbleStyle) Layout(gtx C, w layout.Widget) D {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			surface := clip.UniformRRect(f32.Rectangle{
				Max: layout.FPt(gtx.Constraints.Min),
			}, float32(gtx.Px(c.CornerRadius)))
			paint.FillShape(gtx.Ops, c.Color, surface.Op(gtx.Ops))
			return D{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(w),
	)
}

// MessageID uniquely identifies a message (regardless of the kind of message).
type MessageID string

const NoID = MessageID("")

// MessageData is a type that can be presented by a MessageManager.
type MessageData interface {
	ID() MessageID
}

// Presenter is a function that can transform the data for a chat
// component into a widget to be laid out in the chat
// interface.
type Presenter func(current MessageData, state interface{}) layout.Widget

// Allocator is a function that can allocate the appropriate state
// type for a given MessageData.
type Allocator func(current MessageData) (state interface{})

// MessageManager presents heterogenous message data.
type MessageManager struct {
	// Messages is the list of data to present.
	Messages []MessageData
	// Presenter is a function that can transform a single MessageData into
	// a presentable widget.
	Presenter
	// Allocator is a function that can instantiate the state for a particular
	// MessageData.
	Allocator
	// MessageState is a map storing the state for the MessageDatas managed
	// by the manager.
	MessageState map[MessageID]interface{}
}

// NewManager constructs a manager with the given allocator and presenter.
func NewManager(allocator Allocator, presenter Presenter) *MessageManager {
	return &MessageManager{
		Presenter:    presenter,
		Allocator:    allocator,
		MessageState: make(map[MessageID]interface{}),
	}
}

// Layout the MessageData at position index within the manager's MessageData
// list.
func (m *MessageManager) Layout(gtx C, index int) D {
	data := m.Messages[index]
	id := data.ID()
	state, ok := m.MessageState[id]
	if !ok && id != NoID {
		state = m.Allocator(data)
		m.MessageState[id] = state
	}
	widget := m.Presenter(data, state)
	return widget(gtx)
}

// Len returns the number of messages managed by this manager.
func (m *MessageManager) Len() int {
	return len(m.Messages)
}
