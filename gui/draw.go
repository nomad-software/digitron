package gui

// Drawing the calculator: the frame rendered by package ui is scaled to
// the window, and mouse positions are mapped back onto it.

import (
	"image"
	"math"
	"unsafe"

	"github.com/nomad-software/qute/qtcore"
	"github.com/nomad-software/qute/qtgui"

	"digitron/ui"
)

// upload enlarges the Go frame k times into a QImage, repeating each art
// pixel as a k×k block.
func (a *App) upload(k int) *qtgui.QImage {
	if a.up == nil || a.upK != k {
		if a.up != nil {
			a.up.Delete()
		}
		size := qtcore.QSize{Width: ui.W * k, Height: ui.H * k}
		a.up, a.upK = qtgui.NewQImageWithSize(size, qtgui.QImageFormat_RGBA8888), k
	}
	bpl := a.up.BytesPerLine()
	dst := unsafe.Slice((*byte)(a.up.Bits()), bpl*ui.H*k)
	for y := range ui.H {
		src := a.frame.Pix[y*a.frame.Stride : y*a.frame.Stride+ui.W*4]
		row := dst[y*k*bpl : y*k*bpl+ui.W*k*4]
		for x := range ui.W {
			px := src[x*4 : x*4+4]
			for i := range k {
				copy(row[(x*k+i)*4:], px)
			}
		}
		for r := 1; r < k; r++ {
			copy(dst[(y*k+r)*bpl:], row)
		}
	}
	return a.up
}

// paint stretches the calculator to fill the whole window, scaling each
// axis independently. To keep the pixel art sharp at any size it uses "sharp bilinear" scaling:
// a nearest-neighbour enlargement to the next whole multiple of the device
// size, then a smooth shrink to the exact size, so only the seams between
// art pixels are blended, never the pixels themselves.
func (a *App) paint() {
	ui.Render(a.frame, a.eng, &a.view)
	defer a.afterFirstPaint()

	ww, wh := a.w.Width(), a.w.Height()
	a.sx, a.sy = float64(ww)/ui.W, float64(wh)/ui.H

	// Enlarge enough for the larger of the two axes, so neither is ever
	// stretched by the smooth pass.
	dpr := a.w.AsQPaintDevice().DevicePixelRatioF()
	k := max(1, int(math.Ceil(max(a.sx, a.sy)*dpr-1e-9)))
	img := a.upload(k)

	p := qtgui.NewQPainterOn(a.w)
	defer p.Delete()
	p.SetRenderHint(qtgui.QPainterSmoothPixmapTransform)
	p.DrawImageIn(qtcore.QRect{Width: ww, Height: wh}, img)
	p.End()
}

func (a *App) toVirtual(x, y float64) image.Point {
	return image.Pt(int(x/a.sx), int(y/a.sy))
}

// setIcon uses the logo, enlarged with sharp pixels, as the window icon.
func (a *App) setIcon() {
	logo := ui.IconAt(8)
	n := logo.Bounds().Dx()
	img := qtgui.NewQImageFromBytes(logo.Pix, n, n, logo.Stride, qtgui.QImageFormat_RGBA8888)
	defer img.Delete()
	pm := qtgui.QPixmapFromImage(img)
	defer pm.Delete()
	icon := qtgui.NewQIconWithPixmap(pm)
	defer icon.Delete()
	a.w.SetWindowIcon(icon)
}
