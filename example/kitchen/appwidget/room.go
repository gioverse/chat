package appwidget

import (
	"gioui.org/widget"
	chatwidget "git.sr.ht/~gioverse/chat/widget"
)

// Room selector state.
type Room struct {
	widget.Clickable
	Image  chatwidget.CachedImage
	Active bool
}
