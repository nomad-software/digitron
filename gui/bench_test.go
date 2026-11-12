package gui

// Benchmarks of the Qt layer: painting the window, which renders the frame,
// enlarges it into a QImage and draws that scaled to the window, and the
// handling of a key and of a mouse click, each a call from Qt into Go and
// back. They run headlessly, on Qt's offscreen platform.
//
//	go test -run '^$' -bench . -benchmem ./gui

import (
	"testing"

	"github.com/nomad-software/qute"
	"github.com/nomad-software/qute/qtcore"
	"github.com/nomad-software/qute/qtgui"
	"github.com/nomad-software/qute/qutetest"

	"digitron/settings"
	"digitron/ui"
)

// TestMain starts Qt, on its offscreen platform, for the benchmarks.
func TestMain(m *testing.M) { qutetest.TestMain(m) }

// benchApp shows the calculator at k times its size, with its settings in
// a scratch folder.
func benchApp(b *testing.B, k int) *App {
	b.Helper()
	var a *App
	painted := false
	dir := b.TempDir()
	qutetest.Run(b, func() {
		c := settings.Defaults()
		c.Width, c.Height = ui.W*k, ui.H*k
		a = newApp(c, dir+"/settings.json")
		// The sound is never started: playing it is not the bindings' work.
		a.soundPending = false
		a.show()
		// The window is laid out and exposed, so it can be painted.
		qute.ProcessEvents()
		a.w.Repaint()
		painted = a.upK != 0
	})
	b.Cleanup(func() { qutetest.Run(b, func() { a.w.Hide() }) })
	if !painted {
		b.Fatal("the window was not painted")
	}
	return a
}

// BenchmarkRender draws the frame in Go, as every paint does first, without
// Qt: what a paint takes beyond this is Qt's and the bindings'.
func BenchmarkRender(b *testing.B) {
	a := benchApp(b, 1)
	b.ReportAllocs()
	for b.Loop() {
		ui.Render(a.frame, a.eng, &a.view)
	}
}

func benchPaint(b *testing.B, k int) {
	a := benchApp(b, k)
	b.ReportAllocs()
	b.ResetTimer()
	qutetest.Run(b, func() {
		for b.Loop() {
			a.w.Repaint()
		}
	})
}

func BenchmarkPaint1x(b *testing.B) { benchPaint(b, 1) }
func BenchmarkPaint3x(b *testing.B) { benchPaint(b, 3) }

// BenchmarkResizePaint paints the window at sizes needing a different
// enlargement each time, so the QImage is made again for every frame.
func BenchmarkResizePaint(b *testing.B) {
	a := benchApp(b, 2)
	b.ReportAllocs()
	b.ResetTimer()
	qutetest.Run(b, func() {
		k := 2
		for b.Loop() {
			k = 5 - k
			a.w.Resize(ui.W*k, ui.H*k)
			a.w.Repaint()
		}
	})
}

// BenchmarkKey sends the window a key press (F5, HEX), handled without
// painting.
func BenchmarkKey(b *testing.B) {
	a := benchApp(b, 2)
	b.ReportAllocs()
	b.ResetTimer()
	qutetest.Run(b, func() {
		for b.Loop() {
			ev := qtgui.NewQKeyEvent(qtcore.QEventKeyPress, int(qtcore.Key_F5), qtcore.NoModifier)
			qtcore.QCoreApplicationSendEvent(a.w, ev)
			ev.Delete()
		}
	})
}

// BenchmarkClick presses and releases the mouse on the HEX key, handled
// without painting.
func BenchmarkClick(b *testing.B) {
	a := benchApp(b, 2)
	var r = ui.Keys[0].Rect
	for _, k := range ui.Keys {
		if k.ID == "hex" {
			r = k.Rect
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	qutetest.Run(b, func() {
		at := qtcore.QPointF{X: float64(r.Min.X + r.Max.X), Y: float64(r.Min.Y + r.Max.Y)} // the middle, at 2x
		for b.Loop() {
			for _, t := range []qtcore.QEventType{qtcore.QEventMouseButtonPress, qtcore.QEventMouseButtonRelease} {
				ev := qtgui.NewQMouseEvent(t, at, at, qtcore.LeftButton, qtcore.LeftButton, qtcore.NoModifier)
				qtcore.QCoreApplicationSendEvent(a.w, ev)
				ev.Delete()
			}
		}
	})
}
