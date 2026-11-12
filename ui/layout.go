package ui

// The layout of the calculator's face: its size, the displays' glass and
// the keys, which drawing and hit-testing share.

import (
	"image"
	"strings"
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

// Key is one clickable button on the virtual screen.
type Key struct {
	ID    string
	Label string
	Rect  image.Rectangle
	Dark  bool
	Deep  bool // a dark key a shade darker: the keys that open a pop-up
	Scale int  // label scale (1 small keys, 2 main keys)
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
			deep := ids[i] == "ws" || ids[i] == "snd"
			ks = append(ks, Key{ID: ids[i], Label: labels[i], Rect: image.Rect(x, y, x+31, y+22), Dark: true, Deep: deep, Scale: 1})
		}
	}
	small(0,
		[]string{"oct", "dec", "hex", "shl", "shr", "and", "or", "xor", "not", "bs"},
		[]string{"OCT", "DEC", "HEX", "<<", ">>", "AND", "OR", "XOR", "NOT", "⌫"})
	small(1,
		[]string{"(", ")", "hA", "hB", "hC", "hD", "hE", "hF", "ws", "snd"},
		[]string{"(", ")", "A", "B", "C", "D", "E", "F", "WRD", "VOL"})

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

// keyByID returns the key with the given id.
func keyByID(id string) Key {
	for _, k := range Keys {
		if k.ID == id {
			return k
		}
	}
	panic("ui: no key " + id)
}
