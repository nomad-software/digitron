package gui

// Handling of the word-size menu, which package ui draws and lays out.

import (
	"fmt"

	"github.com/nomad-software/qute/qtcore"

	"digitron/ui"
)

// chooseWidth applies the highlighted word size and saves it straight away,
// so it survives even if the app is not closed cleanly.
func (a *App) chooseWidth() {
	a.view.MenuChoose(a.eng)
	a.saveSettings()
}

// menuMove moves the menu's highlight by d items, ticking if it moved.
func (a *App) menuMove(d int) {
	before := a.view.MenuSel
	a.view.MenuMove(d)
	if a.view.MenuSel != before {
		a.sounds.Tick()
	}
}

// menuKey handles the keyboard while the word-size menu is open: arrows
// move, Enter or Space picks, Escape or W closes, and typing a size's first
// digit (8, 1, 3, 6) picks it directly.
func (a *App) menuKey(key qtcore.Key, text string) {
	switch key {
	case qtcore.Key_Up:
		a.menuMove(-1)
	case qtcore.Key_Down:
		a.menuMove(1)
	case qtcore.Key_Home:
		a.menuMove(-len(ui.MenuItems))
	case qtcore.Key_End:
		a.menuMove(len(ui.MenuItems))
	case qtcore.Key_Return, qtcore.Key_Enter, qtcore.Key_Space:
		a.sounds.Key()
		a.chooseWidth()
	case qtcore.Key_Escape, qtcore.Key_W:
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
