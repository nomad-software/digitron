// Package ui draws the calculator as true low-resolution pixel art into an
// image.RGBA (the "virtual screen"), which the Qt layer scales to the window
// with sharp edges between art pixels. This file holds the palette, the
// View, and the drawing of the case, the keys and the whole frame.
package ui

import (
	"image"
	"image/color"

	"digitron/calc"
)

func rgb(hex uint32) color.RGBA {
	return color.RGBA{uint8(hex >> 16), uint8(hex >> 8), uint8(hex), 255}
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	l := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t + 0.5) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
}

// Palette.
var (
	cOutline   = rgb(0x2b2520)
	cBody      = rgb(0xd8c8ab)
	cBodyHi    = rgb(0xece1ca)
	cBodyLo    = rgb(0xb4a487)
	cBodyLo2   = rgb(0x998a6f)
	cGrille    = rgb(0x6f6353)
	cBezel     = rgb(0x5d564f)
	cBezelHi   = rgb(0x7d756c)
	cBezelLo   = rgb(0x3c3631)
	cGlass     = rgb(0x0c150e)
	cGlassDith = rgb(0x0f1a11)
	cGlassShd  = rgb(0x060a07)
	cLCD       = rgb(0x5cdc53) // lit segments
	cLCDDim    = mix(cGlass, cLCD, 0.45)
	cLCDGhost  = mix(cGlass, cLCD, 0.16)
	cLCDErr    = cLCDDim // an error is a message, not a value

	lightKey = keyColors{face: rgb(0xe4d5b9), hi: rgb(0xf5ecda), side: rgb(0xa6957a), text: rgb(0x2b2520)}
	darkKey  = keyColors{face: rgb(0x6b625a), hi: rgb(0x8a8077), side: rgb(0x433c36), text: rgb(0xece2cf)}
	deepKey  = keyColors{face: rgb(0x48413b), hi: rgb(0x645c55), side: rgb(0x2b2622), text: rgb(0xece2cf)}
)

type keyColors struct{ face, hi, side, text color.RGBA }

// View holds UI state that is not part of the calculator engine.
type View struct {
	Pressed    string // key id currently held down
	MainScroll int    // first visible line of an overflowing main display

	// Word-size menu.
	Menu    bool // the menu is open
	MenuSel int  // highlighted item (hover or keyboard)

	// Volume slider.
	VolumeOpen bool // the slider is open
	Volume     int  // the level, from 0 (mute) to MaxVolume

	// Filled in by Render:
	Bits []BitHit
}

// ---- calculator body -------------------------------------------------------

func (c canvas) body() {
	full := image.Rect(0, 0, W, H)
	c.fill(full, cOutline)
	inner := full.Inset(1)
	c.fill(inner, cBody)
	c.hline(1, W-1, 1, cBodyHi)
	c.vline(1, 1, H-1, cBodyHi)
	c.vline(W-2, 1, H-1, cBodyLo)
	c.hline(1, W-1, H-2, cBodyLo2)
	// Round the four outer corners by a pixel.
	for _, p := range []image.Point{{0, 0}, {W - 1, 0}, {0, H - 1}, {W - 1, H - 1}} {
		c.px(p.X, p.Y, color.RGBA{})
	}

	// Top vent grille: recessed slots with a lit lower lip.
	c.fill(image.Rect(6, 4, W-6, 38), cBodyLo)
	c.fill(image.Rect(7, 5, W-7, 37), cBody)
	for i := range 5 {
		y := 8 + i*6
		c.hline(9, W-9, y, cGrille)
		c.hline(9, W-9, y+1, cBodyLo2)
		c.hline(9, W-9, y+2, cBodyHi)
	}
	// Ledge under the grille.
	c.hline(2, W-2, 40, cBodyLo2)
	c.hline(2, W-2, 41, cBodyHi)

	// Section separators, as on the concept.
	c.hline(4, W-4, 102, cBodyLo)
	c.hline(4, W-4, 103, cBodyHi)
	c.hline(4, W-4, keysSep, cBodyLo)
	c.hline(4, W-4, keysSep+1, cBodyHi)

	// Bottom ledge.
	c.hline(2, W-2, bottomLedge, cBodyLo2)
	c.hline(2, W-2, bottomLedge+1, cBodyHi)
	c.fill(image.Rect(2, bottomLedge+2, W-2, H-2), mix(cBody, cBodyLo, 0.35))
	c.hline(2, W-2, bottomLedge+14, cBodyLo)
	c.hline(2, W-2, bottomLedge+15, cBodyHi)
}

// displayBezel is the width of the frame around the displays' glass.
const displayBezel = 6

// lcd draws a bezel of the given width, outline included, and glass around
// a glass rectangle. It takes at least 4 for any grey to show.
func (c canvas) lcd(g image.Rectangle, bezel int) {
	outer := g.Inset(-bezel)
	c.fill(outer, cOutline)
	c.bevel(outer.Inset(1), cBezel, cBezelLo, cBezelHi) // sunken frame
	c.fill(outer.Inset(2), cBezel)
	c.fill(g.Inset(-1), cGlassShd)
	c.fill(g, cGlass)
	// Faint pixel dither so the glass reads as an LCD matrix.
	for y := g.Min.Y; y < g.Max.Y; y++ {
		for x := g.Min.X + (y & 1); x < g.Max.X; x += 2 {
			c.img.SetRGBA(x, y, cGlassDith)
		}
	}
	c.hline(g.Min.X, g.Max.X, g.Min.Y, cGlassShd)
	c.vline(g.Min.X, g.Min.Y, g.Max.Y, cGlassShd)
}

func (c canvas) key(k Key, pressed, active, enabled bool) {
	pal := lightKey
	switch {
	case k.Deep:
		pal = deepKey
	case k.Dark:
		pal = darkKey
	}
	r := k.Rect
	// Outline with 1px rounded corners.
	c.fill(r, cOutline)
	for _, p := range []image.Point{{r.Min.X, r.Min.Y}, {r.Max.X - 1, r.Min.Y}, {r.Min.X, r.Max.Y - 1}, {r.Max.X - 1, r.Max.Y - 1}} {
		c.px(p.X, p.Y, cBody)
	}
	in := r.Inset(1)
	sideH := 3
	if k.Scale == 1 {
		sideH = 2
	}
	off := 0
	if pressed {
		off = sideH - 1
		sideH = 1
	}
	// Side (key thickness) at the bottom, face above.
	c.fill(in, pal.side)
	face := image.Rect(in.Min.X, in.Min.Y+off, in.Max.X, in.Max.Y-sideH)
	c.fill(face, pal.face)
	if !pressed {
		c.hline(face.Min.X, face.Max.X-1, face.Min.Y, pal.hi)
		c.vline(face.Min.X, face.Min.Y, face.Max.Y-1, pal.hi)
	}
	c.vline(face.Max.X-1, face.Min.Y+1, face.Max.Y, mix(pal.face, pal.side, 0.5))

	col := pal.text
	if active {
		col = cLCD
	}
	if !enabled {
		col = mix(pal.face, pal.text, 0.3)
	}
	tw := textWidth(k.Label, k.Scale)
	th := glyphH * k.Scale
	tx := face.Min.X + (face.Dx()-tw+1)/2
	ty := face.Min.Y + (face.Dy()-th+1)/2
	// Main keys use a faux-bold (double-struck) label like the concept.
	c.text(tx, ty, k.Label, k.Scale, col)
	if k.Scale == 2 {
		c.text(tx+1, ty, k.Label, k.Scale, col)
	}
}

// ---- the whole frame -------------------------------------------------------

func baseName(b int) string {
	return map[int]string{8: "OCT", 10: "DEC", 16: "HEX"}[b]
}

// KeyEnabled reports whether a key does anything in the current mode.
func KeyEnabled(e *calc.Engine, id string) bool {
	switch id {
	case "hA", "hB", "hC", "hD", "hE", "hF":
		return e.Base == 16
	case ".":
		return e.Base == 10
	case "8", "9":
		return e.Base >= 10
	}
	return true
}

func keyActive(e *calc.Engine, v *View, id string) bool {
	switch id {
	case "ws":
		return v.Menu
	case "snd":
		return v.VolumeOpen
	case "oct":
		return e.Base == 8
	case "dec":
		return e.Base == 10
	case "hex":
		return e.Base == 16
	}
	return false
}

// Render draws the full calculator into img (which must be W×H).
func Render(img *image.RGBA, e *calc.Engine, v *View) {
	c := canvas{img}
	c.body()
	c.lcd(MainGlass, displayBezel)
	c.lcd(BinGlass, displayBezel)
	c.mainDisplay(e, v)
	c.binDisplay(e, v)
	for _, k := range Keys {
		c.key(k, v.Pressed == k.ID, keyActive(e, v, k.ID), KeyEnabled(e, k.ID))
	}
	if v.Menu {
		c.menu(e, v)
	}
	if v.VolumeOpen {
		c.volume(v)
	}
}
