package appwidget

import (
	"gioui.org/op/paint"
	"gioui.org/widget"
	"gioui.org/x/richtext"
)

// Message holds the state necessary to facilitate user
// interactions with messages across frames.
type Message struct {
	richtext.InteractiveText
	// Clickable tracks clicks on the message image.
	widget.Clickable
	// Image caches the image operation.
	Image paint.ImageOp
}
