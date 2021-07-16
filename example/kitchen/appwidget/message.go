package appwidget

import (
	"gioui.org/widget"
	"gioui.org/x/component"
	"gioui.org/x/richtext"
)

// Message holds the state necessary to facilitate user
// interactions with messages across frames.
type Message struct {
	richtext.InteractiveText
	// Clickable tracks clicks on the message image.
	widget.Clickable
	// Image caches the image operation.
	Image Image
	// Avatar caches the avatar image operation.
	Avatar Image
	// ContextArea holds the clicks state for the right-click context menu.
	component.ContextArea
}
