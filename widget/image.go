package widget

import (
	"image"

	"gioui.org/op/paint"
)

// CachedImage is a cacheable image operation.
type CachedImage struct {
	op paint.ImageOp
	ch bool
}

// Reload tells the CachedImage to repopulate the cache.
func (img *CachedImage) Reload() {
	img.ch = true
}

// Cache the image if it is not already.
// First call will compute the image operation, subsequent calls will noop.
// When reloaded, cache will re-populated on next invocation.
func (img *CachedImage) Cache(src image.Image) *CachedImage {
	if img == nil || src == nil {
		return img
	}
	if img.op == (paint.ImageOp{}) || img.changed() {
		img.op = paint.NewImageOp(src)
	}
	return img
}

// Op returns the concrete image operation.
func (img CachedImage) Op() paint.ImageOp {
	return paint.ImageOp(img.op)
}

// changed reports whether the underlying image has changed and therefore
// should be cached again.
func (img *CachedImage) changed() bool {
	defer func() { img.ch = false }()
	return img.ch
}
