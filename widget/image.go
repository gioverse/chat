package widget

import (
	"image"

	"gioui.org/op/paint"
)

// CachedImage is a cacheable image operation.
type CachedImage paint.ImageOp

// Changer can report that is has changed since the last call.
type Changer interface {
	Changed() bool
}

// ToNRGBA can render an image.NRGBA image.
type ToNRGBA interface {
	ToNRGBA() *image.NRGBA
}

// Cache the image if it is not already.
//
// First call will compute the image operation, subsequent calls will noop.
//
// If image implements Changer, and Change returns true, the image operation
// will be re-computed.
//
// If image implements ToNRGBA, the *image.NRGBA will be used to compute the
// image operation. This is an optimization since Gio uses a fast-path for
// image.NRGBA images.
func (img *CachedImage) Cache(src image.Image) {
	bake((*paint.ImageOp)(img), src)
}

// Op returns the concrete image operation.
func (img CachedImage) Op() paint.ImageOp {
	return paint.ImageOp(img)
}

// bake the image into a paint.ImageOp, if not already.
func bake(cache *paint.ImageOp, src image.Image) {
	if cache == nil || src == nil {
		return
	}
	var (
		img image.Image = src
	)
	if nrgba, ok := src.(ToNRGBA); ok {
		img = nrgba.ToNRGBA()
	}
	if changer, ok := src.(Changer); (ok && changer.Changed()) || *cache == (paint.ImageOp{}) {
		*cache = paint.NewImageOp(img)
	}
}
