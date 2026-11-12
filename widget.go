package main

// The window: a QWidget whose events are handed to the app.

import (
	"github.com/nomad-software/qute/qtgui"
	"github.com/nomad-software/qute/qtwidgets"
)

// window is the calculator's window, a Go subclass of QWidget.
type window struct {
	qtwidgets.QWidgetBase
	a *app
}

// newWindow makes the window for a, not yet shown.
func newWindow(a *app) *qtwidgets.QWidget {
	w := &window{a: a}
	qtwidgets.NewQWidgetWith(w, nil)
	return w.QWidget
}

func (w *window) PaintEvent(*qtgui.QPaintEvent)           { w.a.paint() }
func (w *window) MousePressEvent(ev *qtgui.QMouseEvent)   { w.a.mousePress(ev) }
func (w *window) MouseReleaseEvent(ev *qtgui.QMouseEvent) { w.a.mouseRelease() }
func (w *window) MouseMoveEvent(ev *qtgui.QMouseEvent)    { w.a.mouseMove(ev) }
func (w *window) WheelEvent(ev *qtgui.QWheelEvent)        { w.a.wheel(ev) }

// KeyPressEvent hands a key the calculator does not use to QWidget.
func (w *window) KeyPressEvent(ev *qtgui.QKeyEvent) {
	if !w.a.keyPress(ev) {
		w.Super().KeyPressEvent(ev)
	}
}

func (w *window) ResizeEvent(ev *qtgui.QResizeEvent) {
	w.Super().ResizeEvent(ev)
	w.a.resized(ev.Size())
}

// CloseEvent saves the settings as the window closes.
func (w *window) CloseEvent(ev *qtgui.QCloseEvent) {
	w.a.saveSettings()
	w.Super().CloseEvent(ev)
}
