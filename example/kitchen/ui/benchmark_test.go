package ui

import (
	"image"
	"testing"

	"gioui.org/gpu/headless"
	"gioui.org/io/router"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

func BenchmarkKitchen(b *testing.B) {
	const scale = 2
	sz := image.Point{X: 800 * scale, Y: 600 * scale}
	w, err := headless.NewWindow(sz.X, sz.Y)
	if err != nil {
		b.Error(err)
	}
	ui := NewUI(func() {}, Config{
		Theme:      "light",
		LoadSize:   10,
		BufferSize: 100,
	})
	gtx := layout.Context{
		Ops: new(op.Ops),
		Metric: unit.Metric{
			PxPerDp: scale,
			PxPerSp: scale,
		},
		Constraints: layout.Exact(sz),
		Queue:       new(router.Router),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ui.Layout(gtx)
		w.Frame(gtx.Ops)
	}
}
