package ui

import (
	"fmt"
	"image"

	"digitron/calc"
)

// The word-size menu pops up above the WS key: a raised body-coloured panel
// holding one small dark key per size, with a notch pointing down at WS.

const (
	menuItemW   = 56
	menuItemH   = 16
	menuItemGap = 3
	menuPad     = 5
	menuTitleH  = 12
	menuNotch   = 4
)

// MenuItem is one choice in the word-size menu.
type MenuItem struct {
	Rect  image.Rectangle
	Width int
}

var (
	// MenuRect is the menu panel, MenuItems its choices, top to bottom.
	MenuRect, MenuItems = buildMenu()
)

func wsKey() Key {
	for _, k := range Keys {
		if k.ID == "ws" {
			return k
		}
	}
	panic("ui: no WS key")
}

func buildMenu() (image.Rectangle, []MenuItem) {
	ws := wsKey().Rect
	n := len(calc.Widths)
	w := menuItemW + 2*menuPad
	h := menuPad + menuTitleH + n*menuItemH + (n-1)*menuItemGap + menuPad
	// Right-align with the WS key (clamped to the body) and sit just above it.
	right := min(ws.Max.X+4, W-4)
	bottom := ws.Min.Y - menuNotch
	panel := image.Rect(right-w, bottom-h, right, bottom)
	var items []MenuItem
	y := panel.Min.Y + menuPad + menuTitleH
	for _, wd := range calc.Widths {
		x := panel.Min.X + menuPad
		items = append(items, MenuItem{Rect: image.Rect(x, y, x+menuItemW, y+menuItemH), Width: wd})
		y += menuItemH + menuItemGap
	}
	return panel, items
}

// MenuItemAt returns the index of the menu item under p, or -1.
func MenuItemAt(p image.Point) int {
	for i, it := range MenuItems {
		if p.In(it.Rect) {
			return i
		}
	}
	return -1
}

// OpenMenu shows the word-size menu with the current size selected.
func (v *View) OpenMenu(e *calc.Engine) {
	v.Menu = true
	v.MenuDown = -1
	v.MenuSel = 0
	for i, it := range MenuItems {
		if it.Width == e.Width {
			v.MenuSel = i
		}
	}
}

// CloseMenu hides the word-size menu.
func (v *View) CloseMenu() {
	v.Menu = false
	v.MenuDown = -1
}

// MenuMove moves the menu selection by d items, stopping at the ends.
func (v *View) MenuMove(d int) {
	v.MenuSel = max(0, min(len(MenuItems)-1, v.MenuSel+d))
}

// MenuChoose applies the selected word size and closes the menu.
func (v *View) MenuChoose(e *calc.Engine) {
	if v.MenuSel >= 0 && v.MenuSel < len(MenuItems) {
		e.SetWidth(MenuItems[v.MenuSel].Width)
	}
	v.CloseMenu()
}

func (c canvas) menu(e *calc.Engine, v *View) {
	r := MenuRect

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

	// Notch pointing at the WS key.
	ws := wsKey().Rect
	cx := (ws.Min.X + ws.Max.X) / 2
	for i := range menuNotch {
		y := r.Max.Y + i - 1
		hw := menuNotch - i
		c.hline(cx-hw, cx+hw+1, y, cBody)
		c.px(cx-hw-1, y, cOutline)
		c.px(cx+hw+1, y, cOutline)
	}
	c.px(cx, r.Max.Y+menuNotch-1, cOutline)

	// Title with an engraved rule underneath.
	title := "WORD"
	tx := r.Min.X + (r.Dx()-textWidth(title, 1))/2
	c.text(tx, r.Min.Y+menuPad, title, 1, cBodyLo2)
	ry := r.Min.Y + menuPad + glyphH + 2
	c.hline(r.Min.X+menuPad, r.Max.X-menuPad, ry, cBodyLo)
	c.hline(r.Min.X+menuPad, r.Max.X-menuPad, ry+1, cBodyHi)

	for i, it := range MenuItems {
		k := Key{Label: fmt.Sprintf("%d BIT", it.Width), Rect: it.Rect, Dark: i != v.MenuSel, Scale: 1}
		c.key(k, v.MenuDown == i, it.Width == e.Width, true)
	}
}
