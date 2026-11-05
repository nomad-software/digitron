// Digitron is an exact programmer's calculator with a pixel-art face. The
// calculator is drawn as low-resolution pixel art by package ui and scaled
// to the window with QPainter (see draw.go).
package main

import (
	"fmt"
	"image"
	"os"
	"runtime"

	qt "github.com/mappu/miqt/qt6"

	"digitron/calc"
	"digitron/desktop"
	"digitron/settings"
	"digitron/sound"
	"digitron/ui"
)

type app struct {
	w     *qt.QWidget
	eng   *calc.Engine
	view  ui.View
	frame *image.RGBA

	// up is the frame enlarged by a whole-number factor upK with
	// nearest-neighbour sampling, ready to be shrunk to the window.
	up  *qt.QImage
	upK int

	// Current horizontal and vertical blit scale, for mapping mouse
	// coordinates back.
	sx, sy float64

	// config is the saved settings, and path the file they are saved in.
	config settings.Settings
	path   string

	// sounds plays the chirp of a key and the tick of the menu; nil when
	// sound is off or unavailable, and until it has been started.
	sounds *sound.Player
	// soundStart starts the sound once the window is on screen, because
	// Qt Multimedia takes about a second to start; soundPending is whether
	// it is still to be started.
	soundStart   *qt.QTimer
	soundPending bool

	mouseKey string
	flash    *qt.QTimer
}

func main() {
	// The settings are kept where the system keeps the configuration of
	// programs, or in the current directory if it has no such place.
	path, err := settings.Prepared()
	if err != nil {
		fmt.Fprintln(os.Stderr, "digitron: the settings are kept in the current directory:", err)
		path = "settings.json"
	}

	qt.NewQApplication(os.Args)
	qt.QCoreApplication_SetApplicationName(desktop.Name)
	qt.QGuiApplication_SetApplicationDisplayName("Digitron")
	// Under Wayland this is the name a window goes by, which a desktop
	// matches to the desktop entry of that name for its icon and its place
	// in the task bar.
	qt.QGuiApplication_SetDesktopFileName(desktop.Name)

	// On Linux the desktop is given an entry for the program, with the logo
	// as its icon, which starts the copy that is running now, wherever it is
	// and whatever it is called, so that it can be started from the
	// desktop's own menus and kept in a dock or sidebar.
	if runtime.GOOS == "linux" {
		if err := desktop.Install(ui.IconPNG(8)); err != nil {
			fmt.Fprintln(os.Stderr, "digitron: the desktop entry could not be written:", err)
		}
	}

	a := &app{
		eng:    calc.New(),
		frame:  image.NewRGBA(image.Rect(0, 0, ui.W, ui.H)),
		config: settings.Load(path),
		path:   path,
	}
	a.eng.SetWidth(a.config.WordSize)
	defer a.sounds.Close()
	a.w = qt.NewQWidget2()
	a.w.SetWindowTitle("Digitron")
	a.w.SetFocusPolicy(qt.StrongFocus)
	a.w.SetMouseTracking(true) // for menu hover
	a.w.SetMinimumSize2(ui.W, ui.H)
	a.w.Resize(a.config.Width, a.config.Height)

	if a.config.Sound {
		a.soundPending = true
		a.soundStart = qt.NewQTimer()
		a.soundStart.SetSingleShot(true)
		a.soundStart.OnTimeout(a.startSound)
	}

	a.flash = qt.NewQTimer()
	a.flash.SetSingleShot(true)
	a.flash.OnTimeout(func() {
		if a.mouseKey == "" {
			a.view.Pressed = ""
			a.w.Update()
		}
	})

	a.w.OnPaintEvent(func(super func(*qt.QPaintEvent), ev *qt.QPaintEvent) { a.paint() })
	a.w.OnMousePressEvent(a.mousePress)
	a.w.OnMouseReleaseEvent(a.mouseRelease)
	a.w.OnMouseMoveEvent(a.mouseMove)
	a.w.OnWheelEvent(a.wheel)
	a.w.OnKeyPressEvent(a.keyPress)
	a.w.OnCloseEvent(func(super func(*qt.QCloseEvent), ev *qt.QCloseEvent) {
		a.saveSettings()
		super(ev)
	})

	a.setIcon()
	if a.config.Maximized {
		a.w.ShowMaximized()
	} else {
		a.w.Show()
	}
	qt.QApplication_Exec()
}

// afterFirstPaint starts the sound once the first frame has been drawn: the
// timer fires after the paint has finished and reached the screen.
func (a *app) afterFirstPaint() {
	if a.soundPending {
		a.soundPending = false
		a.soundStart.Start(0)
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
}

// saveSettings saves the settings, with the word size and the size of the
// window. A maximised window keeps the size it had before.
func (a *app) saveSettings() {
	a.config.WordSize = a.eng.Width
	a.config.Maximized = a.w.IsMaximized()
	if !a.config.Maximized {
		a.config.Width, a.config.Height = a.w.Width(), a.w.Height()
	}
	if err := a.config.Save(a.path); err != nil {
		fmt.Fprintln(os.Stderr, "digitron: the settings could not be saved:", err)
	}
}
