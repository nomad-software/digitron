package gui

// Mouse and keyboard input.

import (
	"image"

	"github.com/nomad-software/qute/qtcore"
	"github.com/nomad-software/qute/qtgui"

	"digitron/ui"
)

// eventPoint returns where a mouse event happened on the virtual screen.
func (a *App) eventPoint(ev *qtgui.QSinglePointEvent) image.Point {
	pos := ev.Position()
	return a.toVirtual(pos.X, pos.Y)
}

func (a *App) mousePress(ev *qtgui.QMouseEvent) {
	pt := a.eventPoint(&ev.QSinglePointEvent)
	if a.view.Menu {
		// While the menu is open a click either picks an item or, anywhere
		// else, dismisses the menu without reaching the keys underneath.
		if i := ui.MenuItemAt(pt); i >= 0 {
			a.view.MenuSel = i
			a.sounds.Key()
			a.chooseWidth()
		} else if !pt.In(ui.MenuRect) {
			a.view.CloseMenu()
		}
		a.w.Update()
		return
	}
	if a.view.VolumeOpen {
		a.volumePress(pt)
		a.w.Update()
		return
	}
	// A key acts as it goes down, as on a real calculator, so that what is
	// heard and what happens always go together; it stays down until the
	// button is let go.
	if k, ok := ui.KeyAt(pt); ok {
		a.mouseKey = k.ID
		a.view.Pressed = k.ID
		a.sounds.Key()
		a.press(k.ID)
		a.w.Update()
		return
	}
	for _, b := range a.view.Bits {
		if pt.In(b.Rect) {
			a.eng.ToggleBit(b.Index)
			a.w.Update()
			return
		}
	}
}

// mouseRelease lets a key held down by the mouse back up, and ends a drag
// of the volume slider's knob, setting the volume it was left at.
func (a *App) mouseRelease() {
	a.endVolumeDrag()
	if a.mouseKey == "" {
		return
	}
	a.mouseKey = ""
	a.view.Pressed = ""
	a.w.Update()
}

// mouseMove drags the volume slider's knob, which only moves it, or
// highlights the menu item under the pointer.
func (a *App) mouseMove(ev *qtgui.QMouseEvent) {
	if a.dragVolume {
		a.moveVolume(ui.VolumeAt(a.eventPoint(&ev.QSinglePointEvent).Y))
		a.w.Update()
		return
	}
	if !a.view.Menu {
		return
	}
	if i := ui.MenuItemAt(a.eventPoint(&ev.QSinglePointEvent)); i >= 0 && i != a.view.MenuSel {
		a.view.MenuSel = i
		a.sounds.Tick()
		a.w.Update()
	}
}

// press performs a key's action, mouse or keyboard, and brings a long
// number back to its first line. WRD opens the word-size menu and the sound
// key the volume slider instead of acting on the engine.
func (a *App) press(id string) {
	a.view.MainScroll = 0
	switch id {
	case "ws":
		a.view.OpenMenu(a.eng)
	case "snd":
		a.view.OpenVolume()
	default:
		ui.Press(a.eng, id)
	}
}

func (a *App) wheel(ev *qtgui.QWheelEvent) {
	dy := ev.AngleDelta().Y
	step := 0
	switch {
	case dy > 0:
		step = -1
	case dy < 0:
		step = 1
	}
	if a.view.Menu {
		a.menuMove(step)
		a.w.Update()
		return
	}
	if a.view.VolumeOpen {
		a.setVolume(a.view.Volume - step) // up is louder
		a.w.Update()
		return
	}
	if !a.eventPoint(&ev.QSinglePointEvent).In(ui.MainGlass) {
		return
	}
	a.view.MainScroll += step
	a.w.Update()
}

// tap presses a key from the keyboard and briefly shows it held down.
func (a *App) tap(id string) {
	a.sounds.Key()
	a.press(id)
	a.view.Pressed = id
	a.flash.StartWithInterval(110)
	a.w.Update()
}

// keyPress handles a key, and reports whether it was one the calculator
// uses.
func (a *App) keyPress(ev *qtgui.QKeyEvent) bool {
	key := qtcore.Key(ev.Key())
	if ev.Modifiers()&qtcore.ControlModifier != 0 {
		switch key {
		case qtcore.Key_Q:
			a.w.Close()
		case qtcore.Key_C:
			qtgui.QGuiApplicationClipboard().SetText(a.eng.CopyText())
		case qtcore.Key_V:
			if a.eng.Paste(qtgui.QGuiApplicationClipboard().Text()) {
				a.view.MainScroll = 0
			}
			a.w.Update()
		}
		return true
	}
	if a.view.Menu {
		a.menuKey(key, ev.Text())
		return true
	}
	if a.view.VolumeOpen {
		a.volumeKey(key, ev.Text())
		return true
	}
	switch key {
	case qtcore.Key_Backspace:
		a.tap("bs")
	case qtcore.Key_Escape:
		a.tap("ac")
	case qtcore.Key_Delete:
		a.tap("c")
	case qtcore.Key_Return, qtcore.Key_Enter:
		a.tap("eq")
	case qtcore.Key_F5:
		a.tap("hex")
	case qtcore.Key_F6:
		a.tap("dec")
	case qtcore.Key_F7:
		a.tap("oct")
	case qtcore.Key_F9:
		a.tap("neg")
	default:
		id := ui.KeyForText(ev.Text())
		if id == "" {
			return false
		}
		a.tap(id)
	}
	return true
}
