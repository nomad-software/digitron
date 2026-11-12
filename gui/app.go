// Package gui is the calculator's window: the Qt layer, which hands mouse
// and keyboard events to the engine and the views of package ui, and draws
// the frame package ui renders, scaled to the window.
package gui

import (
	"fmt"
	"image"
	"os"

	"github.com/nomad-software/qute/qtcore"
	"github.com/nomad-software/qute/qtgui"
	"github.com/nomad-software/qute/qtwidgets"

	"digitron/calc"
	"digitron/settings"
	"digitron/sound"
	"digitron/ui"
)

// App is the calculator: its window, its engine and what it shows.
type App struct {
	w     *qtwidgets.QWidget
	eng   *calc.Engine
	view  ui.View
	frame *image.RGBA

	// up is the frame enlarged by a whole-number factor upK with
	// nearest-neighbour sampling, ready to be shrunk to the window.
	up  *qtgui.QImage
	upK int

	// Current horizontal and vertical blit scale, for mapping mouse
	// coordinates back.
	sx, sy float64

	// config is the saved settings, and path the file they are saved in.
	config settings.Settings
	path   string

	// sounds plays the chirp of a key and the tick of the menu; nil when
	// sound is unavailable, and until it has been started.
	sounds *sound.Player
	// soundStart starts the sound once the window is on screen, because
	// Qt Multimedia takes about a second to start; soundPending is whether
	// it is still to be started.
	soundStart   *qtcore.QTimer
	soundPending bool

	mouseKey string
	flash    *qtcore.QTimer

	// dragVolume is whether the mouse is dragging the volume slider's knob.
	dragVolume bool

	// reshape puts the window back in the calculator's shape once it has
	// stopped being resized; shapedW×shapedH is its size when last in shape.
	reshape          *qtcore.QTimer
	shapedW, shapedH int
}

// Open makes the calculator's window from the settings saved in path, and
// shows it. It must be called inside qute.Main.
func Open(config settings.Settings, path string) *App {
	a := newApp(config, path)
	a.show()
	return a
}

// Close releases the sound. It is called after qute.Main returns; a nil App,
// one never opened, has nothing to release.
func (a *App) Close() {
	if a != nil {
		a.sounds.Close()
	}
}

// newApp makes the calculator's window, not yet shown, from the settings
// saved in path. The Qt application must have been made.
func newApp(config settings.Settings, path string) *App {
	a := &App{
		eng:    calc.New(),
		frame:  image.NewRGBA(image.Rect(0, 0, ui.W, ui.H)),
		config: config,
		path:   path,
	}
	a.eng.SetWidth(a.config.WordSize)
	a.view.Volume = a.config.Volume
	a.w = newWindow(a)
	a.w.SetWindowTitle("Digitron")
	a.w.SetFocusPolicy(qtcore.StrongFocus)
	a.w.SetMouseTracking(true) // for menu hover
	a.w.SetMinimumSize(ui.W, ui.H)
	// A saved size out of shape is put in shape, keeping its width.
	w, h := ui.WindowSize(a.config.Width, a.config.Height, a.config.Width, a.config.Height)
	a.w.Resize(w, h)
	a.startReshaping(w, h)

	// The sound is started even when muted, so that turning it up later
	// does not stall the window while Qt Multimedia starts.
	a.soundPending = true
	a.soundStart = qtcore.NewQTimer(a.w)
	a.soundStart.SetSingleShot(true)
	a.soundStart.OnTimeout(a.startSound)

	a.flash = qtcore.NewQTimer(a.w)
	a.flash.SetSingleShot(true)
	a.flash.OnTimeout(func() {
		if a.mouseKey == "" {
			a.view.Pressed = ""
			a.w.Update()
		}
	})

	a.setIcon()
	return a
}

// show shows the window, maximised if it was when last closed.
func (a *App) show() {
	if a.config.Maximized {
		a.w.ShowMaximized()
	} else {
		a.w.Show()
	}
}

// afterFirstPaint starts the sound once the first frame has been drawn: the
// timer fires after the paint has finished and reached the screen.
func (a *App) afterFirstPaint() {
	if a.soundPending {
		a.soundPending = false
		a.soundStart.StartWithInterval(0)
	}
}

// startSound readies the sounds. Until it has run, keys make no sound.
func (a *App) startSound() {
	s, err := sound.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, "digitron: there will be no sound:", err)
		return
	}
	a.sounds = s
	a.applyVolume()
}

// saveSettings saves the settings, with the word size, the volume and the
// size of the window. A maximised window keeps the size it had before.
func (a *App) saveSettings() {
	a.config.WordSize = a.eng.Width
	a.config.Volume = a.view.Volume
	a.config.Maximized = a.w.IsMaximized()
	if !a.config.Maximized {
		a.config.Width, a.config.Height = a.w.Width(), a.w.Height()
	}
	if err := a.config.Save(a.path); err != nil {
		fmt.Fprintln(os.Stderr, "digitron: the settings could not be saved:", err)
	}
}
