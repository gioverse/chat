package appwidget

import (
	"image"

	"gioui.org/op/paint"
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
	Image paint.ImageOp
	// Avatar caches the avatar image.
	Avatar paint.ImageOp
	// ContextArea holds the clicks state for the right-click context menu.
	component.ContextArea
}

// SetAvatar to the provided image.
// Image texture will be cached, changes to image will be ignored.
func (m *Message) SetAvatar(avatar image.Image) {
	bake(&m.Avatar, avatar)
}

// SetImage to the provided image.
// Image texture will be cached, changes to image will be ignored.
func (m *Message) SetImage(img image.Image) {
	bake(&m.Image, img)
}

// bake the image into a paint.ImageOp, if not already.
func bake(cache *paint.ImageOp, img image.Image) {
	if cache == nil || img == nil {
		return
	}
	if *cache == (paint.ImageOp{}) {
		*cache = paint.NewImageOp(img)
	}
}
