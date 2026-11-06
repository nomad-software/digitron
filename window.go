package main

// Keeping the window the shape of the calculator, so that the frame is
// scaled by the same amount on both axes and every art pixel is square.
//
// Wayland has no way for a program to keep its window to a shape while it
// is dragged, and correcting the size at every step of a drag fights the
// compositor and flickers. So the window is left alone while it is being
// resized, and put back in shape once the resizing has stopped.

import (
	qt "github.com/mappu/miqt/qt6"

	"digitron/ui"
)

// reshapeDelay is how long, in milliseconds, the window must go without
// being resized before it is put back in shape.
const reshapeDelay = 300

// startReshaping readies the timer that puts the window back in shape, and
// notes the window's size, already in shape, as the one a resize starts
// from.
func (a *app) startReshaping(w, h int) {
	a.shapedW, a.shapedH = w, h
	a.reshape = qt.NewQTimer()
	a.reshape.SetSingleShot(true)
	a.reshape.OnTimeout(a.snapToShape)
}

// resized waits for the resizing to stop before putting the window in
// shape: each resize starts the wait again. A size already in shape needs
// nothing, and becomes the size the next resize starts from.
func (a *app) resized(super func(*qt.QResizeEvent), ev *qt.QResizeEvent) {
	super(ev)
	if a.w.IsMaximized() || a.w.IsFullScreen() {
		a.reshape.Stop()
		return
	}
	size := ev.Size() // belongs to the event and is freed with it
	if ui.InShape(size.Width(), size.Height()) {
		a.shapedW, a.shapedH = size.Width(), size.Height()
		a.reshape.Stop()
		return
	}
	a.reshape.Start(reshapeDelay)
}

// snapToShape puts the window in the calculator's shape: the side that was
// dragged further, since it was last in shape, keeps its length, and the
// other follows it. A maximised or full-screen window takes the size the
// desktop gives it, and is left alone.
func (a *app) snapToShape() {
	if a.w.IsMaximized() || a.w.IsFullScreen() {
		return
	}
	w, h := ui.WindowSize(a.w.Width(), a.w.Height(), a.shapedW, a.shapedH)
	a.shapedW, a.shapedH = w, h
	if w != a.w.Width() || h != a.w.Height() {
		a.w.Resize(w, h)
	}
}
