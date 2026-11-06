package main

// Mouse and keyboard input.

import (
	"image"

	qt "github.com/mappu/miqt/qt6"

	"digitron/ui"
)

// eventPoint returns where a mouse event happened on the virtual screen.
// Position returns a copy with a finalizer, so it is not deleted here.
func (a *app) eventPoint(ev *qt.QSinglePointEvent) image.Point {
	pos := ev.Position()
	return a.toVirtual(pos.X(), pos.Y())
}

func (a *app) mousePress(super func(*qt.QMouseEvent), ev *qt.QMouseEvent) {
	pt := a.eventPoint(ev.QSinglePointEvent)
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
func (a *app) mouseRelease(super func(*qt.QMouseEvent), ev *qt.QMouseEvent) {
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
func (a *app) mouseMove(super func(*qt.QMouseEvent), ev *qt.QMouseEvent) {
	if a.dragVolume {
		a.moveVolume(ui.VolumeAt(a.eventPoint(ev.QSinglePointEvent).Y))
		a.w.Update()
		return
	}
	if !a.view.Menu {
		return
	}
	if i := ui.MenuItemAt(a.eventPoint(ev.QSinglePointEvent)); i >= 0 && i != a.view.MenuSel {
		a.view.MenuSel = i
		a.sounds.Tick()
		a.w.Update()
	}
}

// press performs a key's action. WRD opens the word-size menu and the sound
// key the volume slider instead of acting on the engine.
func (a *app) press(id string) {
	switch id {
	case "ws":
		a.view.OpenMenu(a.eng)
	case "snd":
		a.view.OpenVolume()
	default:
		ui.Press(a.eng, id)
	}
}

func (a *app) wheel(super func(*qt.QWheelEvent), ev *qt.QWheelEvent) {
	dy := ev.AngleDelta().Y() // a copy with a finalizer, not deleted here
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
	if !a.eventPoint(ev.QSinglePointEvent).In(ui.MainGlass) {
		return
	}
	a.view.MainScroll += step
	a.w.Update()
}

// tap presses a key from the keyboard and briefly shows it held down.
func (a *app) tap(id string) {
	a.sounds.Key()
	a.press(id)
	a.view.Pressed = id
	a.view.MainScroll = 0
	a.flash.Start(110)
	a.w.Update()
}

func (a *app) keyPress(super func(*qt.QKeyEvent), ev *qt.QKeyEvent) {
	key := qt.Key(ev.Key())
	if ev.Modifiers()&qt.ControlModifier != 0 {
		switch key {
		case qt.Key_Q:
			a.w.Close()
		case qt.Key_C:
			qt.QGuiApplication_Clipboard().SetText(a.eng.CopyText())
		case qt.Key_V:
			a.eng.Paste(qt.QGuiApplication_Clipboard().Text())
			a.w.Update()
		}
		return
	}
	if a.view.Menu {
		a.menuKey(key, ev.Text())
		return
	}
	if a.view.VolumeOpen {
		a.volumeKey(key, ev.Text())
		return
	}
	switch key {
	case qt.Key_Backspace:
		a.tap("bs")
	case qt.Key_Escape:
		a.tap("ac")
	case qt.Key_Delete:
		a.tap("c")
	case qt.Key_Return, qt.Key_Enter:
		a.tap("eq")
	case qt.Key_F5:
		a.tap("hex")
	case qt.Key_F6:
		a.tap("dec")
	case qt.Key_F7:
		a.tap("oct")
	case qt.Key_F9:
		a.tap("neg")
	default:
		if id := ui.KeyForText(ev.Text()); id != "" {
			a.tap(id)
		} else {
			super(ev)
		}
	}
}
