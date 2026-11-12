package ui

import "image"

// A pop-up (the word-size menu, the volume slider) is a raised
// body-coloured panel that sits just above the key that opens it, with a
// notch pointing down at that key and a title engraved along its top.

const (
	popupPad    = 5
	popupTitleH = 12
	popupNotch  = 4
)

// popupRect places a panel w×h just above key k, right-aligned with it and
// kept inside the body.
func popupRect(k Key, w, h int) image.Rectangle {
	right := min(k.Rect.Max.X+4, W-4)
	bottom := k.Rect.Min.Y - popupNotch
	return image.Rect(right-w, bottom-h, right, bottom)
}

// popupTop is the top of a panel's contents, below its title.
func popupTop(r image.Rectangle) int {
	return r.Min.Y + popupPad + popupTitleH
}

// popup draws the panel r, its notch pointing at key k, and its title.
func (c canvas) popup(r image.Rectangle, k Key, title string) {
	// Drop shadow: a dithered dark band down and to the right.
	sh := r.Add(image.Pt(2, 2))
	for y := sh.Min.Y; y < sh.Max.Y; y++ {
		for x := sh.Min.X; x < sh.Max.X; x++ {
			if !image.Pt(x, y).In(r) && (x+y)&1 == 0 {
				c.px(x, y, cOutline)
			}
		}
	}

	// Panel: outline, raised bevel, body face.
	c.fill(r, cOutline)
	for _, p := range []image.Point{{r.Min.X, r.Min.Y}, {r.Max.X - 1, r.Min.Y}, {r.Min.X, r.Max.Y - 1}, {r.Max.X - 1, r.Max.Y - 1}} {
		c.px(p.X, p.Y, cBodyLo)
	}
	c.bevel(r.Inset(1), cBody, cBodyHi, cBodyLo2)

	// Notch pointing at the key.
	cx := (k.Rect.Min.X + k.Rect.Max.X) / 2
	for i := range popupNotch {
		y := r.Max.Y + i - 1
		hw := popupNotch - i
		c.hline(cx-hw, cx+hw+1, y, cBody)
		c.px(cx-hw-1, y, cOutline)
		c.px(cx+hw+1, y, cOutline)
	}
	c.px(cx, r.Max.Y+popupNotch-1, cOutline)

	// Title with an engraved rule underneath.
	tx := r.Min.X + (r.Dx()-textWidth(title, 1))/2
	c.text(tx, r.Min.Y+popupPad, title, 1, cBodyLo2)
	ry := r.Min.Y + popupPad + glyphH + 2
	c.hline(r.Min.X+popupPad, r.Max.X-popupPad, ry, cBodyLo)
	c.hline(r.Min.X+popupPad, r.Max.X-popupPad, ry+1, cBodyHi)
}
