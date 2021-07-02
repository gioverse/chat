package appwidget

import "gioui.org/x/richtext"

// Message holds the state necessary to facilitate user
// interactions with messages across frames.
type Message struct {
	richtext.InteractiveText
}
