package appwidget

import (
	"image"

	"gioui.org/op/paint"
	"gioui.org/widget"
)

// Room selector state.
type Room struct {
	widget.Clickable
	Image  Image
	Active bool
}

// Image is a cacheable `paint.ImageOp`.
type Image paint.ImageOp

// Cache the image if it is not already set.
func (img *Image) Cache(src image.Image) {
	bake((*paint.ImageOp)(img), src)
}

// Op returns the concrete image operation.
func (img Image) Op() paint.ImageOp {
	return paint.ImageOp(img)
}
