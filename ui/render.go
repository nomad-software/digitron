// Package ui draws the calculator as true low-resolution pixel art into an
// image.RGBA (the "virtual screen"); the Qt layer scales it up with
// nearest-neighbour sampling so every art pixel stays a crisp square.
package ui

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"digitron/calc"
)

// The layout from the top down, in art pixels. Everything below the binary
// display is placed from its foot, so changing its height moves the keys
// and the bottom of the case with it.
const (
	binGlassH = 28 // height of the binary display's glass

	binTop      = 110                // top of the binary display's glass
	binFoot     = binTop + binGlassH // its foot
	keysSep     = binFoot + 9        // the line between it and the keys
	smallKeysY  = keysSep + 9        // top of the two rows of small keys
	mainKeysY   = smallKeysY + 62    // top of the main keys
	bottomLedge = mainKeysY + 154    // the ledge below them
)

// Virtual screen size in art pixels.
const (
	W = 372
	H = bottomLedge + 26
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
	cLCDErr    = cLCD

	lightKey = keyColors{face: rgb(0xe4d5b9), hi: rgb(0xf5ecda), side: rgb(0xa6957a), text: rgb(0x2b2520)}
	darkKey  = keyColors{face: rgb(0x6b625a), hi: rgb(0x8a8077), side: rgb(0x433c36), text: rgb(0xece2cf)}
)

type keyColors struct{ face, hi, side, text color.RGBA }

// Key is one clickable button on the virtual screen.
type Key struct {
	ID    string
	Label string
	Rect  image.Rectangle
	Dark  bool
	Scale int // label scale (1 small keys, 2 main keys)
}

// Display regions (glass, inside the bezel).
var (
	MainGlass = image.Rect(16, 52, 356, 94)
	BinGlass  = image.Rect(16, binTop, 356, binFoot)
)

// Keys is the full keyboard layout.
var Keys = buildKeys()

func buildKeys() []Key {
	var ks []Key
	small := func(row int, ids, labels []string) {
		for i := range ids {
			x := 13 + i*35
			y := smallKeysY + row*28
			ks = append(ks, Key{ID: ids[i], Label: labels[i], Rect: image.Rect(x, y, x+31, y+22), Dark: true, Scale: 1})
		}
	}
	small(0,
		[]string{"bin", "oct", "dec", "hex", "shl", "shr", "and", "or", "xor", "not"},
		[]string{"BIN", "OCT", "DEC", "HEX", "<<", ">>", "AND", "OR", "XOR", "NOT"})
	small(1,
		[]string{"(", ")", "hA", "hB", "hC", "hD", "hE", "hF", "ws", "bs"},
		[]string{"(", ")", "A", "B", "C", "D", "E", "F", "WS", "⌫"})

	main := [][]string{
		{"mc:MC", "7:7", "8:8", "9:9", "div:÷", "c:C", "ac:AC"},
		{"mr:MR", "4:4", "5:5", "6:6", "mul:×", "pct:%", "sqrt:√"},
		{"m+:M+", "1:1", "2:2", "3:3", "sub:−", "eq:="},
		{"m-:M−", "0:0", ".:.", "neg:+/-", "add:+"},
	}
	for r, row := range main {
		for c, spec := range row {
			id, label, _ := strings.Cut(spec, ":")
			x := 14 + c*50
			y := mainKeysY + r*38
			rc := image.Rect(x, y, x+44, y+32)
			if id == "eq" {
				rc = image.Rect(x, y, x+94, y+70)
			}
			dark := c >= 4
			ks = append(ks, Key{ID: id, Label: label, Rect: rc, Dark: dark, Scale: 2})
		}
	}
	return ks
}

// KeyAt returns the key under a virtual-screen point.
func KeyAt(p image.Point) (Key, bool) {
	for _, k := range Keys {
		if p.In(k.Rect) {
			return k, true
		}
	}
	return Key{}, false
}

// BitHit maps a clickable bit on the binary display to its index.
type BitHit struct {
	Rect  image.Rectangle
	Index int
}

// View holds UI state that is not part of the calculator engine.
type View struct {
	Pressed    string // key id currently held down
	MainScroll int    // first visible line of an overflowing main display

	// Word-size menu.
	Menu    bool // the menu is open
	MenuSel int  // highlighted item (hover or keyboard)

	// Filled in by Render:
	Bits []BitHit
}

// bigFit is how many characters the main display holds in its big digits.
var bigFit = (MainGlass.Dx() - 8 + 2) / (advance * 2)

const (
	mainVisibleLines = 3
	bitsPerRow       = 32
)

// ---- primitive drawing -----------------------------------------------------

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

// glyph draws one character at scale; over draws an overline (repeating digit).
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
		x += advance * scale
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

// lcd draws a bezel and glass around a glass rectangle.
func (c canvas) lcd(g image.Rectangle) {
	outer := g.Inset(-6)
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
	if k.Dark {
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
	return map[int]string{2: "BIN", 8: "OCT", 10: "DEC", 16: "HEX"}[b]
}

// KeyEnabled reports whether a key does anything in the current mode.
func KeyEnabled(e *calc.Engine, id string) bool {
	switch id {
	case "hA", "hB", "hC", "hD", "hE", "hF":
		return e.Base == 16
	case ".":
		return e.Base == 10
	case "2", "3", "4", "5", "6", "7":
		return e.Base >= 8
	case "8", "9":
		return e.Base >= 10
	}
	return true
}

func keyActive(e *calc.Engine, v *View, id string) bool {
	switch id {
	case "ws":
		return v.Menu
	case "bin":
		return e.Base == 2
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
	c.lcd(MainGlass)
	c.lcd(BinGlass)
	c.mainDisplay(e, v)
	c.binDisplay(e, v)
	for _, k := range Keys {
		c.key(k, v.Pressed == k.ID, keyActive(e, v, k.ID), KeyEnabled(e, k.ID))
	}
	if v.Menu {
		c.menu(e, v)
	}
}

func (c canvas) mainDisplay(e *calc.Engine, v *View) {
	g := MainGlass
	// Status row: mode flags on the left, pending expression on the right.
	status := fmt.Sprintf("%s %d", baseName(e.Base), e.Width)
	if e.HasMemory() {
		status += " M"
	}
	f := e.Display()
	if (e.Approx() || f.Rounded) && e.Err() == nil {
		status += " ≈"
	}
	if d := e.OpenParens(); d > 0 {
		status += fmt.Sprintf(" (%d", d)
	}
	c.text(g.Min.X+3, g.Min.Y+3, status, 1, cLCDDim)
	statusW := textWidth(status, 1) + 12

	trail := strings.ToUpper(e.Trail())
	maxTrail := (g.Dx() - 6 - statusW) / advance
	if r := []rune(trail); len(r) > maxTrail && maxTrail > 1 {
		trail = "…" + string(r[len(r)-maxTrail+1:])
	}
	c.text(g.Max.X-3-textWidth(trail, 1), g.Min.Y+3, trail, 1, cLCDDim)

	if err := e.Err(); err != nil {
		msg := "E " + err.Error()
		c.text(g.Max.X-4-textWidth(msg, 2), g.Max.Y-18, msg, 2, cLCDErr)
		return
	}

	cells := f.Cells
	if len(cells) <= bigFit {
		v.MainScroll = 0
		x := g.Max.X - 4 - (len(cells)*advance*2 - 2)
		y := g.Max.Y - 18
		for _, cl := range cells {
			c.glyph(x, y, cl.Ch, 2, cLCD)
			if cl.Over {
				c.fill(image.Rect(x, y-4, x+glyphW*2, y-2), cLCD)
			}
			x += advance * 2
		}
		return
	}

	// Too long for the big font: wrap in the small font, scrollable.
	perLine := (g.Dx() - 14) / advance
	var lines [][]calc.Cell
	for i := 0; i < len(cells); i += perLine {
		lines = append(lines, cells[i:min(i+perLine, len(cells))])
	}
	v.MainScroll = max(0, min(v.MainScroll, len(lines)-mainVisibleLines))
	for i := 0; i < mainVisibleLines && v.MainScroll+i < len(lines); i++ {
		y := g.Min.Y + 13 + i*9
		x := g.Min.X + 4
		for _, cl := range lines[v.MainScroll+i] {
			c.glyph(x, y, cl.Ch, 1, cLCD)
			if cl.Over {
				c.hline(x, x+glyphW, y-2, cLCD)
			}
			x += advance
		}
	}
	// Scroll markers and digit count.
	if v.MainScroll > 0 {
		c.glyph(g.Max.X-8, g.Min.Y+13, '▲', 1, cLCDDim)
	}
	if v.MainScroll+mainVisibleLines < len(lines) {
		c.glyph(g.Max.X-8, g.Min.Y+31, '▼', 1, cLCDDim)
	}
}

// binDisplay shows the word's bits, most significant first. 8- and 16-bit
// words fit one row in the big font; 32 and 64 bits use rows of 32 in the
// mid-size bit font. An integer's bits are grouped in fours; a fraction is
// shown as floating point (calc.FloatBitsOf), labelled FP and the width,
// with its sign, exponent and mantissa set apart and the mantissa grouped
// in fours from its end. Only an integer's bits can be clicked.
func (c canvas) binDisplay(e *calc.Engine, v *View) {
	g := BinGlass
	v.Bits = v.Bits[:0]
	if e.Err() != nil {
		return
	}
	float := e.IsFloat()
	if !float && !e.FitsWord() {
		// DEC has no word size, and this value does not fit the one chosen:
		// its bits would mean nothing, so none are shown.
		msg := fmt.Sprintf("BEYOND %d BITS", e.Width)
		c.text(g.Min.X+(g.Dx()-textWidth(msg, 1))/2, g.Min.Y+(g.Dy()-glyphH)/2, msg, 1, cLCDDim)
		return
	}
	var bits []bool
	if float {
		bits = calc.FloatBitsOf(e.Value(), e.Width)
	} else {
		bits = calc.BitsOf(e.Value(), e.Width)
	}
	n := len(bits)
	perRow := min(n, bitsPerRow)
	rows := (n + perRow - 1) / perRow

	// glyph draws one bit; gw, gh are its size and bitAdv the column pitch.
	var glyph func(x, y int, on bool, col color.RGBA)
	var gw, gh, bitAdv, nibGap, fieldGap, rowGap int
	if n <= 16 {
		gw, gh, bitAdv, nibGap, fieldGap = glyphW*2, glyphH*2, 7*2, 5*2, 9*2
		glyph = func(x, y int, on bool, col color.RGBA) {
			ch := '0'
			if on {
				ch = '1'
			}
			c.glyph(x, y, ch, 2, col)
		}
	} else {
		gw, gh, bitAdv, nibGap, fieldGap, rowGap = bitGlyphW, bitGlyphH, bitGlyphW+2, 4, 10, 4
		glyph = c.bitGlyph
	}
	// gap is the space to the right of bit i, between it and bit i-1.
	f := calc.FloatFormats[e.Width]
	gap := func(i int) int {
		switch {
		case !float:
			if i%4 == 0 {
				return nibGap
			}
		case i == f.MantBits || i == f.MantBits+f.ExpBits:
			return fieldGap
		case i < f.MantBits && i%4 == 0:
			return nibGap
		}
		return 0
	}
	// offset is how far left of the row's right edge bit i starts.
	offset := func(lo, i int) int {
		x := (i-lo+1)*bitAdv - (bitAdv - gw)
		for k := lo + 1; k <= i; k++ {
			x += gap(k)
		}
		return x
	}
	rowW := 0
	for row := range rows {
		rowW = max(rowW, offset(row*perRow, row*perRow+perRow-1))
	}
	right := g.Min.X + (g.Dx()+rowW)/2 // x just past the LSB column
	rowH := gh + rowGap
	top := g.Min.Y + (g.Dy()-(rows*rowH-rowGap)+1)/2

	for vr := range rows {
		row := rows - 1 - vr // top row holds the highest bits
		lo := row * perRow
		hi := lo + perRow - 1
		y := top + vr*rowH
		x := right
		for i := lo; i <= hi; i++ {
			x = right - offset(lo, i)
			col := cLCDGhost
			if bits[i] {
				col = cLCD
			}
			glyph(x, y, bits[i], col)
			if !float {
				v.Bits = append(v.Bits, BitHit{Rect: image.Rect(x-1, y-1, x+gw+1, y+gh+1), Index: i})
			}
		}
		// Labels: the high bit's index left of the row, or, on the top row of
		// a float, its format; the low bit's index right of it.
		ly := y + (gh-tinyH)/2
		left := fmt.Sprint(hi)
		if float && vr == 0 {
			left = fmt.Sprintf("FP%d", e.Width)
		}
		c.tiny(x-5-len(left)*tinyAdv, ly, left, cLCDDim)
		c.tiny(right+4, ly, fmt.Sprint(lo), cLCDDim)
	}
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
