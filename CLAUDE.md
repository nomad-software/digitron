# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Digitron is a retro programmer's calculator: exact rational arithmetic, OCT/DEC/HEX (there is no BIN base: binary is entered by clicking bits on the binary display, or pasted with `0b`), two's-complement word sizes of 8 to 64 bits in OCT/HEX (DEC has no limit), a binary display that shows fractions as IEEE 754 floating point, and a face drawn as true low-resolution pixel art. It is written in Go with a Qt 6 GUI via the [qute](https://github.com/nomad-software/qute) bindings (the user's own project, checked out beside this one in `../qute`), and is for Linux only. `README.md` is the user-facing description.

## Layout

- `calc/` — the engine. No dependency on Qt or the UI.
  - Values are exact `big.Rat`s. Precedence follows C. `Width` is always one of `calc.Widths` (8, 16, 32, 64).
  - OCT/HEX keep to the word size (`bound` → `wrap`, which also cuts a fraction towards zero, as a C cast does, so ÷, % and √ need no truncation of their own). `toWord` fits everything the engine holds (shown, operand, waiting operands in `tokens`, memory, `repeatRHS`) when leaving DEC or changing the word size, so no fraction or oversized value hides and comes back in a result. `FitsWord` says whether a value fits the word's signed range; the binary display and `ToggleBit` need it (`ToggleBit` also leaves a DEC fraction alone, as its bits are floating point).
  - DEC has no word size, but one limit, `MaxBits` (2²⁰ bits, top or bottom of the fraction): `tooBig` checks it in `apply`, `Percent`, `memAdd` and `Paste`, giving `ErrTooBig`, and in `entryFits`, which refuses a digit typed past it. A shift count is clamped to just past it before shifting, so a shift is too big exactly when its result is. `digitsOfBits` (log₁₀2 as 30103/100000) turns bits into decimal digits, for `maxDigits` and `decimalExponent`'s estimate.
  - 0 has no sign, as in the Windows calculator: `Negate` does nothing to a 0 being typed (`entryIsZero`) and `Backspace` drops the sign when the entry becomes 0, so nothing ever shows -0 (`checkInvariants` checks it).
  - `paste.go` is copy and paste: `CopyText` gives the exact value; `Paste` reads one `0x`/`0o`/`0b` prefix (`0x0b1` is hex), separators, fractions and scientific notation, and refuses text longer than `maxPaste` or an exponent past `maxDigits` before reading it.
  - `float.go` encodes a fraction in IEEE 754 at the word size (`FloatFormats`: double, single, half, FP8 E5M2) from the exact value with round-to-nearest-even; `TestFloatAgainstGo` checks it against Go's own conversions. `IsFloat` (DEC and a fraction) switches the binary display to it. `SetWidth` in DEC makes a fraction the nearest value of the new format (`toFloat`, decoding with `floatValue`), so the value and its bits agree; `TestFloatValueRoundTrip` decodes and re-encodes every FP16 and FP8 pattern.
  - `Format` fits a decimal to `DisplayCells` (the big digits of the main display; a ui test checks they agree): exact if it fits (a repeating decimal with its repeating block overlined, `Cell.Over`) or if it is a whole number, otherwise rounded to fill it, setting `Formatted.Rounded`, which shows ≈. A small number that would keep fewer than `MinSignificant` (6) digits after its zeros is shown in scientific notation instead (`scientific`, `decimalExponent`: `1.03…E-29`), rounded only if its mantissa is; Paste reads that form back. Square roots are kept to `SqrtDigits` places but always shown rounded.
  - `Equals` remembers the last top-level operation of what it worked out (`remember`) and repeats it when there is no expression; `pending`/`commit` are shared with M+ and M−, which neither repeat nor set up a repeat.
- `ui/` — draws the whole calculator into an `image.RGBA` "virtual screen" of `ui.W`×`ui.H` (372×398) art pixels (`render.go`: the palette, `View`, the case, the keys and `Render`; `canvas.go`: the drawing primitives; `display.go`: the main and binary displays), and owns the layout (`layout.go`: `Keys`, `MainGlass`, `BinGlass`, `KeyAt`), the keys' actions and keyboard map (`actions.go`), the pop-ups (`popup.go`: the panel, notch and title shared by the word-size menu, `menu.go`, and the volume slider, `volume.go`, whose `MaxVolume` a settings test keeps equal to `settings.MaxVolume`), the logo (`icon.go`: `Icon`, and `IconAt`/`IconPNG` for every enlarged copy — the window icon, the desktop entry's, the AppImage's and the README's all come from it), the window's shape (`window.go`: `WindowSize` picks the size in the art's shape nearest a resize, keeping the side dragged further, and `InShape` says whether a size already is) and both fonts (`font.go`). No Qt here either. `View` holds UI state that is not engine state. The binary display (`binDisplay`) has three modes: an integer's bits in fours (clickable), a fraction as floating point (labelled `FP64` etc., and `+INF`/`-INF` at the right of the top row when `calc.FloatInfinity` says it is too big for the format; sign/exponent/mantissa set apart, not clickable; NaN is never made, as everything that would make one is an engine error), and `BEYOND n BITS` for a whole number too big for the word.
- `settings/` — the JSON settings file, in Prism's and ApeMail's pattern: `Defaults`, `Load` (falls back to defaults, then `normalise` replaces any unusable value), atomic `Save`, `Prepared` for the path (`os.UserConfigDir()/digitron/settings.json`).
- `desktop/` — writes the desktop entry and the 256×256 logo into `$XDG_DATA_HOME` (`~/.local/share`) at every start, pointing at the running copy or at `$APPIMAGE`, as Delta does. No dependency on the UI; it is given the icon as PNG data (`ui.IconPNG`). `TestPackagedEntryAgrees` keeps it in step with `packaging/digitron.desktop`. `desktop.Name` is also Qt's application and desktop file name, which Wayland matches windows by.
- `atomicfile/` — replaces a file whole through a temporary file beside it; used by `settings` and `desktop`.
- `sound/` — the key chirp and menu tick, embedded with `//go:embed` and played from memory through Qt Multimedia `QAudioSink`s (one pool of voices per sound, each at its own sample rate). `sound/gen` synthesises `tick.wav`; `key.wav` is cut from a recording with the ffmpeg command in the `sound.go` package comment.
- `main.go` — only startup: the settings path, then inside `qute.Main` Qt's application and desktop file names, the desktop entry and `gui.Open`; `App.Close` releases the sound after `qute.Main` returns.
- `gui/` — the Qt layer: `app.go` (the `App` struct, `Open`, and `newApp`, which makes the window without showing it, settings, starting the sound), `widget.go` (`window`, the Go subclass of `QWidget` whose event overrides hand each event to the `App`), `draw.go` (scaling the frame to the window, the icon), `window.go` (keeping the window the calculator's shape with `ui.WindowSize` and `ui.InShape`: a resize only starts the `reshape` timer, and the window is put in shape `reshapeDelay` after the resizing stops, because correcting it at every step of a drag fights the Wayland compositor and flickers; do not go back to that. Maximised and full-screen windows are left as the desktop sizes them), `input.go` (mouse and keyboard), `wordmenu.go` (the menu's handlers), `volume.go` (the slider's handlers: click, drag, wheel, keys; a drag only moves the knob, `moveVolume`, and the volume is set on release, `endVolumeDrag`, while keys and wheel set it at each step, `setVolume`; it saves the level when closed). `bench_test.go` (`make bench`) times the layer headlessly (painting, `ui.Render` alone, a key and a click sent to the window, with the sound never started), to compare versions of the bindings; switching from miqt to qute left painting unchanged.
- `packaging/`, `.github/workflows/` — the AppImage, ported from Delta's: `appimage.sh` (linuxdeploy, run by the workflow on Ubuntu 22.04 with Qt's own build of Qt 6.12, installed with `aqt`), the desktop entry and `AppRun`. The script builds in `/tmp/digitron-build` (`BUILD`), never in the project, and leaves only `Digitron-<version>-x86_64.AppImage` in the project's root, which `.gitignore` leaves out; `make build` likewise leaves only `digitron` there. It refuses to finish without a Qt Multimedia backend plugin (FFmpeg), and checks that the result starts. `make appimage` fails on Arch: its Qt Multimedia uses the system FFmpeg, whose dependency tree includes a library that crashes the loader once linuxdeploy has patched it. That is expected; releases come from the workflow.
- `docs/previews/` — images drawn by `ui/preview_test.go`; the README's screenshot and logo come from here. `docs/concept.png` is the design concept the face was drawn from.

## Technical debt

Pay technical debt down as part of every feature and fix; do not let cruft build up.

- When a feature is removed or changed, remove what it leaves unused: glyphs in `font.go`, fields of `View`, engine methods, settings, tests of old behaviour. Keep `go vet`, `staticcheck ./...` and `deadcode -test ./...` clean.
- Keep files to one concern; split one that grows a second.
- Keep this file, the README and the previews (`make screenshot`) in step with the code in the same change.
- Anything made to help with the work that is worth running again (a generator, a sound cut) goes in the repository with a make target, a `go:generate` line or a note here, never only in a scratch folder.
- Refactoring changes no behaviour; check with the tests and that `make screenshot` leaves the previews byte-identical.
- Keep test files to one topic too. In `calc`:
  - `engine_test.go`: the operations (arithmetic, √, %, memory, base and word size, bits, C, errors), and `press`/`run`, which every test uses.
  - `entry_test.go`: typing a number (digits, point, +/-, backspace) and the status line's expression.
  - `famous_test.go`: famous calculator and floating-point bugs, Rump's polynomial, Muller's recurrence.
  - `display_test.go`: fitting the display, repeating decimals, scientific notation.
  - `paste_test.go`: copy and paste.
  - `repeat_test.go`: repeated =.
  - `word_test.go`: DEC without a word size (shifts, `MaxBits`) and fitting everything to the word.
  - `float_test.go`: IEEE 754.
  - `c_test.go`: OCT/HEX against Go's fixed-width integers, and C's traps.
  - `fuzz_test.go`: random keys checked against invariants (`checkInvariants`, `FuzzKeys`, `TestKeysAtRandom`), and DEC against `math/big`.
- When fuzzing finds a bug, add the failing case as a named regression test in the file for its topic, and its key sequence to `FuzzKeys`'s seeds.

## Drawing

- Everything visible is drawn in `ui` at art-pixel resolution with `canvas` primitives (`fill`, `bevel`, `text`, `tiny`, `key`, `lcd`, `popup`) and the palette at the top of `render.go`. Never draw with Qt directly; new UI elements (like the menu) are pixel art in `ui`, styled like the keys.
- `lcd` draws a display's glass in a frame of a given width: `displayBezel` (6) for the main and binary displays, `volBezel` (4) for the volume slider's readout. Below 4 no grey shows.
- 5x7 glyphs are 5 wide except a few symbols drawn wider (`⌫` is 9); `text` and `textWidth` lay them out by `glyphAdvance`. The main display's digit cells still assume `advance`.
- Keys come in three shades, `lightKey`, `darkKey` and `deepKey` (`Key.Dark`, `Key.Deep`); the keys that open a pop-up, WRD and VOL, are deep.
- In a pop-up the highlighted item is a light key with dark text, even when it is the current choice; the current choice is lit green only when not highlighted.
- Bright `cLCD` is for values (the main display's number, set bits, the volume readout's level); labels, indicators and messages (the status line, bit indices, `+INF`, the volume slider's lit groove, errors through `cLCDErr`) are `cLCDDim`.
- The window keeps the art's shape (`window.go`), so the stretch below is the same on both axes and the pixels are square, except when maximised or full screen.
- `draw.go` stretches the frame to fill the window (each axis independently) using "sharp bilinear": a nearest-neighbour enlargement to a whole multiple, then one smooth shrink. Mouse positions map back with `toVirtual`.
- Hit-testing uses the same rectangles that drawing uses (`Keys`, `MenuItems`, `View.Bits`).

## Input

- A key acts as it goes down, mouse or keyboard, with its chirp at the same moment; releasing the mouse only lets the key back up. Do not move the action to the release: dragging off a key would then play the chirp and do nothing. Word-size menu items are picked on press too.

## Sound

- Qt Multimedia takes about a second to start (its first `QAudioSink`). It is started from a single-shot timer after the first paint so the window opens at once; until then `app.sounds` is nil and every `Player` method is nil-safe. Do not move sound start back before `Show()`.
- `QSoundEffect` plays only from files; it was replaced because nothing may be written to disk. Do not reintroduce files in caches or temp folders.
- After changing `sound/gen`, run `go generate ./sound`; `TestUpToDate` fails if `tick.wav` drifts from the generator.
- The volume (`settings.Volume`, 0 to 10) reaches the sound through `Player.SetLoudness`, which cubes it for even-sounding steps; at 0 `Key` and `Tick` play nothing. The sound is started even when muted, so turning it up does not stall the window.
- Pitch variation is per sound (`keySpread`, `tickSpread`); the user wants the key chirp almost constant.

## Commands

```sh
make build        # go build -ldflags "-s -w" -o digitron .
make run
make test         # go test -race -timeout 5m ./..., then sound.TestPlay without -race (Qt Multimedia crashes the race detector at exit)
make vet          # go vet ./...
make fuzz         # FuzzKeys for FUZZTIME (default 60s)
make bench        # the Qt layer's benchmarks (gui/bench_test.go)
make screenshot   # redraw docs/previews
make appimage     # packaging/appimage.sh, built in /tmp (releases: the AppImage workflow)
make clean        # the binary, the AppImage and /tmp/digitron-build
go generate ./sound
```

A PostToolUse hook in `.claude/settings.json` runs `make build` after every edit to a `.go` file, `go.mod`, `go.sum` or the `Makefile`, and reports build failures.

## Building with qute

- qute is cgo, one package per Qt module, each found through `pkg-config`: Digitron uses `qtcore`, `qtgui`, `qtwidgets` and `qtmultimedia` (`Qt6Core`, `Qt6Gui`, `Qt6Widgets`, `Qt6Multimedia`). It needs Qt 6.12 or later, as its bindings are generated from 6.12, and a C++17 compiler. `README.md` lists the packages per distribution; keep it, the workflow (the `aqt` version and modules) and the packaging script's checks in step if the Qt modules used change.
- Use a released version of qute (`go get github.com/nomad-software/qute@vX.Y.Z`), never a `replace` pointing at `../qute`, so the workflow builds the same thing.
- The first build compiles qute's C++ shims and takes a couple of minutes. Avoid `go clean -cache`.
- Linux is the only platform. There is no macOS or Windows support; do not add any.

## qute conventions

- Qt calls happen on the GUI thread, inside `qute.Main` (a call from another goroutine panics; use `qute.Post`). Tests start Qt with `qutetest.TestMain` and wrap Qt calls in `qutetest.Run`, which runs a real event loop: timers fire there, which is why the benchmarks clear `soundPending`.
- Names follow Qt's: `qtcore.QTimer`, `qtgui.QImageFormat_RGBA8888`, `qtcore.Key_F5`. An overload has a descriptive name (`StartWithInterval`, `DrawImageIn`, `NewQImageWithSize`, `NewQAudioSinkWithAudioFormat`); read the generated `*_gen.go` in the module cache, or `../qute`, rather than guessing. `QPoint`, `QPointF`, `QSize` and `QRect` are plain Go structs.
- Virtuals are overridden by a Go subclass: `window` embeds `qtwidgets.QWidgetBase`, is made with `qtwidgets.NewQWidgetWith`, and calls the base with `w.Super()`. Event arguments are dead once the handler returns; copy out what is needed.
- Ownership: objects are freed by the GC unless parented, so the timers are parented to the window. `Delete()` of a QObject inside a handler panics (use `DeleteLater`); non-QObjects (painters, images, icons) may be deleted there. `QAudioSink.StartWithDevice` keeps its buffer alive.
- Run `go run github.com/nomad-software/qute/cmd/qute@<version in go.mod> vet ./...` after changing the Qt layer.
