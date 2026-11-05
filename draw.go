package main

// Drawing the calculator: the frame rendered by package ui is scaled to
// the window, and mouse positions are mapped back onto it.

import (
	"image"
	"math"
	"unsafe"

	qt "github.com/mappu/miqt/qt6"

	"digitron/ui"
)

// upload enlarges the Go frame k times into a QImage, repeating each art
// pixel as a k×k block.
func (a *app) upload(k int) *qt.QImage {
	if a.up == nil || a.upK != k {
		if a.up != nil {
			a.up.Delete()
		}
		a.up, a.upK = qt.NewQImage3(ui.W*k, ui.H*k, qt.QImage__Format_RGBA8888), k
	}
	bpl := int(a.up.BytesPerLine())
	dst := unsafe.Slice(a.up.Bits(), bpl*ui.H*k)
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
func (a *app) paint() {
	ui.Render(a.frame, a.eng, &a.view)
	defer a.afterFirstPaint()

	ww, wh := a.w.Width(), a.w.Height()
	a.sx, a.sy = float64(ww)/ui.W, float64(wh)/ui.H

	// Enlarge enough for the larger of the two axes, so neither is ever
	// stretched by the smooth pass.
	dpr := a.w.DevicePixelRatioF()
	k := max(1, int(math.Ceil(max(a.sx, a.sy)*dpr-1e-9)))
	img := a.upload(k)

	p := qt.NewQPainter2(a.w.QPaintDevice)
	defer p.Delete()
	p.SetRenderHint(qt.QPainter__SmoothPixmapTransform)
	target := qt.NewQRect4(0, 0, ww, wh)
	defer target.Delete()
	p.DrawImage6(target, img)
	p.End()
}

func (a *app) toVirtual(x, y float64) image.Point {
	return image.Pt(int(x/a.sx), int(y/a.sy))
}

// setIcon uses the logo, enlarged with sharp pixels, as the window icon.
func (a *app) setIcon() {
	const k = 8
	logo := ui.Icon()
	n := ui.IconSize * k
	img := qt.NewQImage3(n, n, qt.QImage__Format_RGBA8888)
	defer img.Delete()
	bpl := int(img.BytesPerLine())
	dst := unsafe.Slice(img.Bits(), bpl*n)
	for y := range n {
		for x := range n {
			src := logo.PixOffset(x/k, y/k)
			copy(dst[y*bpl+x*4:], logo.Pix[src:src+4])
		}
	}
	pm := qt.QPixmap_FromImage(img) // a copy with a finalizer, not deleted here
	icon := qt.NewQIcon2(pm)
	defer icon.Delete()
	a.w.SetWindowIcon(icon)
}
