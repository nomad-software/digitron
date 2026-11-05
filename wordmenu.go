package main

// Handling of the word-size menu, which package ui draws and lays out.

import (
	"fmt"

	qt "github.com/mappu/miqt/qt6"

	"digitron/ui"
)

// chooseWidth applies the highlighted word size and saves it straight away,
// so it survives even if the app is not closed cleanly.
func (a *app) chooseWidth() {
	a.view.MenuChoose(a.eng)
	a.saveSettings()
}

// menuMove moves the menu's highlight by d items, ticking if it moved.
func (a *app) menuMove(d int) {
	before := a.view.MenuSel
	a.view.MenuMove(d)
	if a.view.MenuSel != before {
		a.sounds.Tick()
	}
}

// menuKey handles the keyboard while the word-size menu is open: arrows
// move, Enter or Space picks, Escape or W closes, and typing a size's first
// digit (8, 1, 3, 6) picks it directly.
func (a *app) menuKey(key qt.Key, text string) {
	switch key {
	case qt.Key_Up:
		a.menuMove(-1)
	case qt.Key_Down:
		a.menuMove(1)
	case qt.Key_Home:
		a.menuMove(-len(ui.MenuItems))
	case qt.Key_End:
		a.menuMove(len(ui.MenuItems))
	case qt.Key_Return, qt.Key_Enter, qt.Key_Space:
		a.sounds.Key()
		a.chooseWidth()
	case qt.Key_Escape, qt.Key_W:
		a.view.CloseMenu()
	default:
		for i, it := range ui.MenuItems {
			if text != "" && fmt.Sprint(it.Width)[:1] == text {
				a.view.MenuSel = i
				a.sounds.Key()
				a.chooseWidth()
				break
			}
		}
	}
	a.w.Update()
}
