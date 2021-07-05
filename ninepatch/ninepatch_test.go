package ninepatch

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"testing"

	"gioui.org/layout"
	"gioui.org/unit"
	"git.sr.ht/~gioverse/chat/res"
)

var (
	platocookie = open("9-Patch/iap_platocookie_asset_2.png")
	hotdog      = open("9-Patch/iap_hotdog_asset.png")
)

// TestDecodeNinePatch tests that 9-Patch data is successfully read from a
// source image.
func TestDecodeNinePatch(t *testing.T) {
	for _, tt := range []struct {
		Label string
		Src   image.Image
		NP    NP
	}{
		{
			Label: "empty image",
			Src:   NewImg(image.Pt(0, 0)),
			NP:    NP{},
		},
		{
			// An image with no stretch markers will be considered "completely
			// static", and therefore will not resize in any way.
			//
			// An image with no content inset will have no padding around
			// content.
			//
			// Both are still "valid" 9-Patch images, however unusable.
			Label: "image with no border",
			Src:   NewImg(image.Pt(100, 100)),
			NP:    NP{Grid: Grid{Size: image.Point{X: 100, Y: 100}}},
		},
		{
			Label: "image with no content inset",
			Src:   NewImg(image.Pt(100, 100)).TopBorder(25, 50).LeftBorder(25, 50),
			NP: NP{
				Grid: Grid{
					Size: image.Point{X: 100, Y: 100},
					X1:   25, X2: 25,
					Y1: 25, Y2: 25,
				},
			},
		},
		{
			Label: "image with no stretch regions",
			Src:   NewImg(image.Pt(100, 100)).BottomBorder(25, 50).RightBorder(25, 50),
			NP: NP{
				Content: layout.Inset{
					Top:    unit.Px(25),
					Right:  unit.Px(25),
					Bottom: unit.Px(25),
					Left:   unit.Px(25),
				},
				Grid: Grid{Size: image.Point{X: 100, Y: 100}},
			},
		},
		{
			Label: "image with content inset and stretch regions",
			Src: NewImg(image.Pt(100, 100)).
				TopBorder(25, 50).
				LeftBorder(25, 50).
				BottomBorder(25, 50).
				RightBorder(25, 50),
			NP: NP{
				Content: layout.Inset{
					Top:    unit.Px(25),
					Right:  unit.Px(25),
					Bottom: unit.Px(25),
					Left:   unit.Px(25),
				},
				Grid: Grid{
					Size: image.Point{X: 100, Y: 100},
					X1:   25, X2: 25,
					Y1: 25, Y2: 25,
				},
			},
		},
		{
			Label: "platocookie",
			Src:   platocookie,
			NP: NP{
				Content: layout.Inset{
					Top:    unit.Px(31),
					Right:  unit.Px(70),
					Bottom: unit.Px(27),
					Left:   unit.Px(70),
				},
				Grid: Grid{
					Size: image.Point{
						X: platocookie.Bounds().Dx(),
						Y: platocookie.Bounds().Dy(),
					},
					X1: 86, X2: 61,
					Y1: 55, Y2: 47,
				},
			},
		},
		{
			Label: "hotdog",
			Src:   hotdog,
			NP: NP{
				Content: layout.Inset{
					Top:    unit.Px(31),
					Right:  unit.Px(70),
					Bottom: unit.Px(27),
					Left:   unit.Px(70),
				},
				Grid: Grid{
					Size: image.Point{
						X: hotdog.Bounds().Dx(),
						Y: hotdog.Bounds().Dy(),
					},
					X1: 86, X2: 61,
					Y1: 55, Y2: 47,
				},
			},
		},
	} {
		t.Run(tt.Label, func(t *testing.T) {
			np := DecodeNinePatch(tt.Src)
			got := NP{
				Content: np.Content,
				Grid:    np.Grid,
			}
			want := tt.NP
			if got != want {
				t.Fatalf("\n got:{%v} \nwant:{%v}\n", got, want)
			}
		})
	}
}

// NP wraps the layout data for a NinePatch for convenient equality testing.
type NP struct {
	Content layout.Inset
	Grid
}

func (np NP) String() string {
	return fmt.Sprintf(
		"Content: %+v, Stretch: {X1:%dpx, X2:%dpx, Y1:%dpx, Y2:%dpx}",
		np.Content, np.X1, np.X2, np.Y1, np.Y2)
}

// Img wraps an image.NRGBA with mutators for creating mock 9-Patch images.
type Img struct {
	*image.NRGBA
}

// NewImg allocates an Img for the given size.
func NewImg(sz image.Point) *Img {
	return &Img{
		NRGBA: image.NewNRGBA(image.Rectangle{Max: sz}),
	}
}

// LeftBorder renders a line along the first column of pixels.
func (img *Img) LeftBorder(start, size int) *Img {
	for ii := start; ii < start+size-1; ii++ {
		img.Set(img.Bounds().Min.X, ii, color.NRGBA{A: 255})
	}
	return img
}

// RightBorder renders a line along the last column of pixels.
func (img *Img) RightBorder(start, size int) *Img {
	for ii := start; ii < start+size-1; ii++ {
		img.Set(img.Bounds().Max.X-1, ii, color.NRGBA{A: 255})
	}
	return img
}

// TopBorder renders a line along the first row of pixels.
func (img *Img) TopBorder(start, size int) *Img {
	for ii := start; ii < start+size-1; ii++ {
		img.Set(ii, img.Bounds().Min.Y, color.NRGBA{A: 255})
	}
	return img
}

// BottomBorder renders a line along the last row of pixels.
func (img *Img) BottomBorder(start, size int) *Img {
	for ii := start; ii < start+size-1; ii++ {
		img.Set(ii, img.Bounds().Max.Y-1, color.NRGBA{A: 255})
	}
	return img
}

// open and decode a png from resources. Panic on failure.
func open(path string) image.Image {
	imgf, err := res.Resources.Open(path)
	if err != nil {
		panic(fmt.Errorf("opening 9-Patch image: %v", err))
	}
	defer imgf.Close()
	img, err := png.Decode(imgf)
	if err != nil {
		panic(fmt.Errorf("decoding png: %v", err))
	}
	return img
}
