package widget

import (
	"gioui.org/widget"
	"gioui.org/x/richtext"
)

// Message holds the state necessary to facilitate user
// interactions with messages across frames.
type Message struct {
	richtext.InteractiveText
	// Clickable tracks clicks on the message image.
	widget.Clickable
	// Image contains the cached image op for the message.
	Image CachedImage
}
