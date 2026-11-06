package ui

import (
	"fmt"
	"image"

	"digitron/calc"
)

// The word-size menu pops up above the WRD key: a pop-up holding one small
// dark key per size.

const (
	menuItemW   = 56
	menuItemH   = 16
	menuItemGap = 3
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

func buildMenu() (image.Rectangle, []MenuItem) {
	n := len(calc.Widths)
	w := menuItemW + 2*popupPad
	h := popupPad + popupTitleH + n*menuItemH + (n-1)*menuItemGap + popupPad
	panel := popupRect(keyByID("ws"), w, h)
	var items []MenuItem
	y := popupTop(panel)
	for _, wd := range calc.Widths {
		x := panel.Min.X + popupPad
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
	c.popup(MenuRect, keyByID("ws"), "WORD")
	// The highlighted item is a light key with dark text, whether or not it
	// is the current size; the current size is lit green only when it is
	// not highlighted.
	for i, it := range MenuItems {
		hl := i == v.MenuSel
		k := Key{Label: fmt.Sprintf("%d BIT", it.Width), Rect: it.Rect, Dark: !hl, Scale: 1}
		c.key(k, false, it.Width == e.Width && !hl, true)
	}
}
