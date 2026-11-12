// Digitron is an exact programmer's calculator with a pixel-art face. The
// calculator is drawn as low-resolution pixel art by package ui and scaled
// to the window with QPainter (see draw.go).
package main

import (
	"fmt"
	"image"
	"os"

	"github.com/nomad-software/qute"
	"github.com/nomad-software/qute/qtcore"
	"github.com/nomad-software/qute/qtgui"
	"github.com/nomad-software/qute/qtwidgets"

	"digitron/calc"
	"digitron/desktop"
	"digitron/settings"
	"digitron/sound"
	"digitron/ui"
)

type app struct {
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

func main() {
	// The settings are kept where the system keeps the configuration of
	// programs, or in the current directory if it has no such place.
	path, err := settings.Prepared()
	if err != nil {
		fmt.Fprintln(os.Stderr, "digitron: the settings are kept in the current directory:", err)
		path = "settings.json"
	}

	var a *app
	code := qute.Main(func() {
		qtcore.QCoreApplicationSetApplicationName(desktop.Name)
		qtgui.QGuiApplicationSetApplicationDisplayName("Digitron")
		// Under Wayland this is the name a window goes by, which a desktop
		// matches to the desktop entry of that name for its icon and its
		// place in the task bar.
		qtgui.QGuiApplicationSetDesktopFileName(desktop.Name)

		// The desktop is given an entry for the program, with the logo as
		// its icon, which starts the copy that is running now, wherever it
		// is and whatever it is called, so that it can be started from the
		// desktop's own menus and kept in a dock or sidebar.
		if err := desktop.Install(ui.IconPNG(8)); err != nil {
			fmt.Fprintln(os.Stderr, "digitron: the desktop entry could not be written:", err)
		}

		a = newApp(settings.Load(path), path)
		a.show()
	})
	a.sounds.Close()
	os.Exit(code)
}

// newApp makes the calculator's window, not yet shown, from the settings
// saved in path. The Qt application must have been made.
func newApp(config settings.Settings, path string) *app {
	a := &app{
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
func (a *app) show() {
	if a.config.Maximized {
		a.w.ShowMaximized()
	} else {
		a.w.Show()
	}
}

// afterFirstPaint starts the sound once the first frame has been drawn: the
// timer fires after the paint has finished and reached the screen.
func (a *app) afterFirstPaint() {
	if a.soundPending {
		a.soundPending = false
		a.soundStart.StartWithInterval(0)
	}
}

// startSound readies the sounds. Until it has run, keys make no sound.
func (a *app) startSound() {
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
func (a *app) saveSettings() {
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
