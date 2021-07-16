package appwidget

import (
	"gioui.org/widget"
	"gioui.org/x/component"
	"gioui.org/x/richtext"
	chatwidget "git.sr.ht/~gioverse/chat/widget"
)

// Message holds the state necessary to facilitate user
// interactions with messages across frames.
type Message struct {
	richtext.InteractiveText
	// Clickable tracks clicks on the message image.
	widget.Clickable
	// Image contains the cached image op for the message.
	Image chatwidget.CachedImage
	// Avatar contains the cached image op for the avatar.
	Avatar chatwidget.CachedImage
	// ContextArea holds the clicks state for the right-click context menu.
	component.ContextArea
}
