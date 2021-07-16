package widget

import "gioui.org/x/component"

// Row holds persistent state for a single row of a chat.
type Row struct {
	// ContextArea holds the clicks state for the right-click context menu.
	component.ContextArea

	Message
	UserInfo
}
