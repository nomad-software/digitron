// Digitron: an exact, arbitrary-precision programmer's calculator with a
// pixel-art face. The calculator is rendered as low-resolution pixel art
// (package ui) and blitted with QPainter using nearest-neighbour scaling.
package main

import (
	"fmt"
	"image"
	"math"
	"os"
	"unsafe"

	qt "github.com/mappu/miqt/qt6"

	"digitron/calc"
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

	// click plays the click of a key; nil when sound is off or unavailable.
	click *sound.Player

	mouseKey string
	swallow  bool // ignore the release of the click that closed the menu
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
	qt.QCoreApplication_SetApplicationName("digitron")

	a := &app{
		eng:    calc.New(),
		frame:  image.NewRGBA(image.Rect(0, 0, ui.W, ui.H)),
		config: settings.Load(path),
		path:   path,
	}
	a.eng.SetWidth(a.config.WordSize)
	if a.config.Sound {
		if a.click, err = sound.New(); err != nil {
			fmt.Fprintln(os.Stderr, "digitron: the keys will make no sound:", err)
		}
		defer a.click.Close()
	}
	a.w = qt.NewQWidget2()
	a.w.SetWindowTitle("Digitron")
	a.w.SetFocusPolicy(qt.StrongFocus)
	a.w.SetMouseTracking(true) // for menu hover
	a.w.SetMinimumSize2(ui.W, ui.H)
	a.w.Resize(a.config.Width, a.config.Height)

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

// upload enlarges the Go frame k times into a QImage, repeating each art
// pixel as a k×k block.
func (a *app) upload(k int) *qt.QImage {
	if a.up == nil || a.upK != k {
		if a.up != nil {
			a.up.Delete()
		}
		a.up, a.upK = qt.NewQImage3(ui.W*k, ui.H*k, qt.QImage__Format_RGBA8888), k
	}
	bpl := int(a.up.BytesPerLine())
	dst := unsafe.Slice(a.up.Bits(), bpl*ui.H*k)
	for y := range ui.H {
		src := a.frame.Pix[y*a.frame.Stride : y*a.frame.Stride+ui.W*4]
		row := dst[y*k*bpl : y*k*bpl+ui.W*k*4]
		for x := range ui.W {
			px := src[x*4 : x*4+4]
			for i := range k {
				copy(row[(x*k+i)*4:], px)
			}
		}
		for r := 1; r < k; r++ {
			copy(dst[(y*k+r)*bpl:], row)
		}
	}
	return a.up
}

// paint stretches the calculator to fill the whole window, scaling each
// axis independently. To keep the pixel art sharp at any size it uses "sharp bilinear" scaling:
// a nearest-neighbour enlargement to the next whole multiple of the device
// size, then a smooth shrink to the exact size, so only the seams between
// art pixels are blended, never the pixels themselves.
func (a *app) paint() {
	ui.Render(a.frame, a.eng, &a.view)

	ww, wh := a.w.Width(), a.w.Height()
	a.sx, a.sy = float64(ww)/ui.W, float64(wh)/ui.H

	// Enlarge enough for the larger of the two axes, so neither is ever
	// stretched by the smooth pass.
	dpr := a.w.DevicePixelRatioF()
	k := max(1, int(math.Ceil(max(a.sx, a.sy)*dpr-1e-9)))
	img := a.upload(k)

	p := qt.NewQPainter2(a.w.QPaintDevice)
	defer p.Delete()
	p.SetRenderHint(qt.QPainter__SmoothPixmapTransform)
	target := qt.NewQRect4(0, 0, ww, wh)
	defer target.Delete()
	p.DrawImage6(target, img)
	p.End()
}

func (a *app) toVirtual(x, y float64) image.Point {
	return image.Pt(int(x/a.sx), int(y/a.sy))
}

func (a *app) eventPoint(ev *qt.QSinglePointEvent) image.Point {
	pos := ev.Position()
	defer pos.Delete()
	return a.toVirtual(pos.X(), pos.Y())
}

func (a *app) mousePress(super func(*qt.QMouseEvent), ev *qt.QMouseEvent) {
	pt := a.eventPoint(ev.QSinglePointEvent)
	if a.view.Menu {
		// While the menu is open a click either picks an item or, anywhere
		// else, dismisses the menu without reaching the keys underneath.
		if i := ui.MenuItemAt(pt); i >= 0 {
			a.view.MenuSel, a.view.MenuDown = i, i
			a.click.Tick()
		} else if !pt.In(ui.MenuRect) {
			a.view.CloseMenu()
			a.swallow = true
		}
		a.w.Update()
		return
	}
	if k, ok := ui.KeyAt(pt); ok {
		a.mouseKey = k.ID
		a.view.Pressed = k.ID
		a.click.Tick()
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

func (a *app) mouseRelease(super func(*qt.QMouseEvent), ev *qt.QMouseEvent) {
	pt := a.eventPoint(ev.QSinglePointEvent)
	if a.swallow {
		a.swallow = false
		return
	}
	if a.view.Menu {
		if d := a.view.MenuDown; d >= 0 {
			a.view.MenuDown = -1
			if ui.MenuItemAt(pt) == d {
				a.chooseWidth()
			}
			a.w.Update()
		}
		return
	}
	if a.mouseKey == "" {
		return
	}
	if k, ok := ui.KeyAt(pt); ok && k.ID == a.mouseKey {
		a.press(k.ID)
	}
	a.mouseKey = ""
	a.view.Pressed = ""
	a.w.Update()
}

// mouseMove highlights the menu item under the pointer.
func (a *app) mouseMove(super func(*qt.QMouseEvent), ev *qt.QMouseEvent) {
	if !a.view.Menu {
		return
	}
	if i := ui.MenuItemAt(a.eventPoint(ev.QSinglePointEvent)); i >= 0 && i != a.view.MenuSel {
		a.view.MenuSel = i
		a.w.Update()
	}
}

// press performs a key's action. WS opens the word-size menu instead of
// acting on the engine directly.
func (a *app) press(id string) {
	if id == "ws" {
		a.view.OpenMenu(a.eng)
		return
	}
	ui.Press(a.eng, id)
}

func (a *app) wheel(super func(*qt.QWheelEvent), ev *qt.QWheelEvent) {
	d := ev.AngleDelta()
	dy := d.Y()
	d.Delete()
	step := 0
	switch {
	case dy > 0:
		step = -1
	case dy < 0:
		step = 1
	}
	if a.view.Menu {
		a.view.MenuMove(step)
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
	a.click.Tick()
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
	case qt.Key_F8:
		a.tap("bin")
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

// chooseWidth applies the highlighted word size and saves it straight away,
// so it survives even if the app is not closed cleanly.
func (a *app) chooseWidth() {
	a.view.MenuChoose(a.eng)
	a.saveSettings()
}

// menuKey handles the keyboard while the word-size menu is open: arrows
// move, Enter or Space picks, Escape or W closes, and typing a size's first
// digit (8, 1, 3, 6) picks it directly.
func (a *app) menuKey(key qt.Key, text string) {
	switch key {
	case qt.Key_Up:
		a.view.MenuMove(-1)
	case qt.Key_Down:
		a.view.MenuMove(1)
	case qt.Key_Home:
		a.view.MenuMove(-len(ui.MenuItems))
	case qt.Key_End:
		a.view.MenuMove(len(ui.MenuItems))
	case qt.Key_Return, qt.Key_Enter, qt.Key_Space:
		a.chooseWidth()
	case qt.Key_Escape, qt.Key_W:
		a.view.CloseMenu()
	default:
		for i, it := range ui.MenuItems {
			if text != "" && fmt.Sprint(it.Width)[:1] == text {
				a.view.MenuSel = i
				a.chooseWidth()
				break
			}
		}
	}
	a.w.Update()
}

// setIcon uses the rendered calculator face as the window icon.
func (a *app) setIcon() {
	ui.Render(a.frame, a.eng, &a.view)
	pm := qt.QPixmap_FromImage(a.upload(1))
	defer pm.Delete()
	icon := qt.NewQIcon2(pm)
	defer icon.Delete()
	a.w.SetWindowIcon(icon)
}
