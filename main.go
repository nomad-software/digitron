// Digitron is an exact programmer's calculator with a pixel-art face. The
// engine is package calc, the face is drawn by package ui, and package gui
// is the Qt window that shows it.
package main

import (
	"fmt"
	"os"

	"github.com/nomad-software/qute"
	"github.com/nomad-software/qute/qtcore"
	"github.com/nomad-software/qute/qtgui"

	"digitron/desktop"
	"digitron/gui"
	"digitron/settings"
	"digitron/ui"
)

func main() {
	// The settings are kept where the system keeps the configuration of
	// programs, or in the current directory if it has no such place.
	path, err := settings.Prepared()
	if err != nil {
		fmt.Fprintln(os.Stderr, "digitron: the settings are kept in the current directory:", err)
		path = "settings.json"
	}

	var a *gui.App
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

		a = gui.Open(settings.Load(path), path)
	})
	a.Close()
	os.Exit(code)
}
