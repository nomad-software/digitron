package ui

// The drawing primitives everything is made of, at art-pixel resolution.

import (
	"image"
	"image/color"
)

type canvas struct{ img *image.RGBA }

func (c canvas) px(x, y int, col color.RGBA) {
	if image.Pt(x, y).In(c.img.Rect) {
		c.img.SetRGBA(x, y, col)
	}
}

func (c canvas) fill(r image.Rectangle, col color.RGBA) {
	r = r.Intersect(c.img.Rect)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			c.img.SetRGBA(x, y, col)
		}
	}
}

func (c canvas) hline(x0, x1, y int, col color.RGBA) { c.fill(image.Rect(x0, y, x1, y+1), col) }
func (c canvas) vline(x, y0, y1 int, col color.RGBA) { c.fill(image.Rect(x, y0, x+1, y1), col) }

// glyph draws one character of the 5x7 font at scale.
func (c canvas) glyph(x, y int, ch rune, scale int, col color.RGBA) {
	g, ok := font5x7[ch]
	if !ok {
		g = font5x7['?']
	}
	for gy, row := range g {
		for gx, b := range row {
			if b == '#' {
				c.fill(image.Rect(x+gx*scale, y+gy*scale, x+(gx+1)*scale, y+(gy+1)*scale), col)
			}
		}
	}
}

func (c canvas) text(x, y int, s string, scale int, col color.RGBA) {
	for _, ch := range s {
		c.glyph(x, y, ch, scale, col)
		x += glyphAdvance(ch) * scale
	}
}

func (c canvas) tiny(x, y int, s string, col color.RGBA) {
	for _, ch := range s {
		g := font3x5[ch]
		for gy, row := range g {
			for gx, b := range row {
				if b == '#' {
					c.px(x+gx, y+gy, col)
				}
			}
		}
		x += tinyAdv
	}
}

// bevel draws a raised (or sunken) rectangle with 1px edges.
func (c canvas) bevel(r image.Rectangle, face, hi, lo color.RGBA) {
	c.fill(r, face)
	c.hline(r.Min.X, r.Max.X, r.Min.Y, hi)
	c.vline(r.Min.X, r.Min.Y, r.Max.Y, hi)
	c.hline(r.Min.X, r.Max.X, r.Max.Y-1, lo)
	c.vline(r.Max.X-1, r.Min.Y, r.Max.Y, lo)
}

func (c canvas) bitGlyph(x, y int, on bool, col color.RGBA) {
	g := bitFont[0]
	if on {
		g = bitFont[1]
	}
	for gy, row := range g {
		for gx, b := range row {
			if b == '#' {
				c.px(x+gx, y+gy, col)
			}
		}
	}
}
