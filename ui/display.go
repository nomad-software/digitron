package ui

// The main display and the binary display.

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"digitron/calc"
)

// bigFit is how many characters the main display holds in its big digits.
var bigFit = (MainGlass.Dx() - 8 + 2) / (advance * 2)

const (
	mainVisibleLines = 3
	bitsPerRow       = 32
)

func (c canvas) mainDisplay(e *calc.Engine, v *View) {
	g := MainGlass
	// Status row: mode flags on the left, pending expression on the right.
	// The word size is left out while the value is beyond it, as the binary
	// display then shows no bits of that size.
	status := baseName(e.Base)
	if !beyondWord(e) {
		status += fmt.Sprintf(" %d", e.Width)
	}
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
	if beyondWord(e) {
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
		// a float, its format; the low bit's index right of it, or, on the top
		// row of a float too big for its format, +INF or -INF.
		ly := y + (gh-tinyH)/2
		left, rightLabel := fmt.Sprint(hi), fmt.Sprint(lo)
		if float && vr == 0 {
			left = fmt.Sprintf("FP%d", e.Width)
			switch calc.FloatInfinity(e.Value(), e.Width) {
			case 1:
				rightLabel = "+INF"
			case -1:
				rightLabel = "-INF"
			}
		}
		c.tiny(x-5-len(left)*tinyAdv, ly, left, cLCDDim)
		c.tiny(right+4, ly, rightLabel, cLCDDim)
	}
}

// beyondWord reports whether the value is a whole number too big for the
// word size, which only DEC allows: its bits would mean nothing, so none are
// shown. A fraction never is; it is shown as floating point instead.
func beyondWord(e *calc.Engine) bool {
	return e.Err() == nil && !e.IsFloat() && !e.FitsWord()
}
