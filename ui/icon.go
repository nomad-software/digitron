package ui

import (
	"image"
	"image/color"
)

// IconSize is the size of the logo in art pixels.
const IconSize = 32

// Icon draws the program's logo: a small calculator in the same palette as
// the full face, with a lit binary readout, on a transparent background.
func Icon() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, IconSize, IconSize))
	c := canvas{img}

	// Body with an outline, a lit top-left edge and rounded corners.
	body := image.Rect(3, 1, 29, 31)
	c.fill(body, cOutline)
	c.bevel(body.Inset(1), cBody, cBodyHi, cBodyLo2)
	for _, p := range []image.Point{{body.Min.X, body.Min.Y}, {body.Max.X - 1, body.Min.Y}, {body.Min.X, body.Max.Y - 1}, {body.Max.X - 1, body.Max.Y - 1}} {
		c.px(p.X, p.Y, color.RGBA{})
	}

	// Display: bezel, glass and a lit binary readout.
	bezel := image.Rect(5, 3, 27, 12)
	c.fill(bezel, cOutline)
	c.fill(bezel.Inset(1), cBezel)
	glass := bezel.Inset(2)
	c.fill(glass, cGlass)
	c.tiny(glass.Min.X+1, glass.Min.Y, "1010", cLCD)

	// Keys: three light columns and a dark column of operators.
	for r := range 4 {
		for col := range 4 {
			x, y := 6+col*5, 14+r*4
			pal := lightKey
			if col == 3 {
				pal = darkKey
			}
			c.fill(image.Rect(x, y, x+4, y+3), pal.side)
			c.fill(image.Rect(x, y, x+4, y+2), pal.face)
			c.px(x, y, pal.hi)
		}
	}
	return img
}
