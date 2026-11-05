# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Digitron is a retro programmer's calculator: exact rational arithmetic, BIN/OCT/DEC/HEX, two's-complement word sizes of 8 to 64 bits in BIN/OCT/HEX (DEC has no limit), a binary display that shows fractions as IEEE 754 floating point, and a face drawn as true low-resolution pixel art. It is written in Go with a Qt 6 GUI via the [miqt](https://github.com/mappu/miqt) bindings. `README.md` is the user-facing description.

## Layout

- `calc/` — the engine. No dependency on Qt or the UI.
  - Values are exact `big.Rat`s. Precedence follows C. `Width` is always one of `calc.Widths` (8, 16, 32, 64).
  - BIN/OCT/HEX keep to the word size (`bound` → `wrap`); DEC has none and keeps values of any size (left shifts there are limited by `MaxShift`). `FitsWord` says whether a value fits the word's signed range; the binary display and `ToggleBit` need it.
  - `float.go` encodes a fraction in IEEE 754 at the word size (`FloatFormats`: double, single, half, FP8 E5M2) from the exact value with round-to-nearest-even; `TestFloatAgainstGo` checks it against Go's own conversions. `IsFloat` (DEC and a fraction) switches the binary display to it. `SetWidth` in DEC makes a fraction the nearest value of the new format (`toFloat`, decoding with `floatValue`), so the value and its bits agree; `TestFloatValueRoundTrip` decodes and re-encodes every FP16 and FP8 pattern.
  - `Format` fits a decimal to `DisplayCells` (the big digits of the main display; a ui test checks they agree): exact if it fits (a repeating decimal with its repeating block overlined, `Cell.Over`) or if it is a whole number, otherwise rounded to fill it, setting `Formatted.Rounded`, which shows ≈. Square roots are kept to `SqrtDigits` places but always shown rounded.
  - `Equals` remembers the last top-level operation of what it worked out (`remember`) and repeats it when there is no expression; `pending`/`commit` are shared with M+ and M−, which neither repeat nor set up a repeat.
- `ui/` — draws the whole calculator into an `image.RGBA` "virtual screen" of `ui.W`×`ui.H` (372×398) art pixels, and owns the layout: `Keys`, `MainGlass`, `BinGlass`, the word-size menu (`menu.go`), the logo (`icon.go`: `Icon`, and `IconAt`/`IconPNG` for every enlarged copy — the window icon, the desktop entry's, the AppImage's, the macOS app's and the README's all come from it) and both fonts (`font.go`). No Qt here either. `View` holds UI state that is not engine state. The binary display (`binDisplay`) has three modes: an integer's bits in fours (clickable), a fraction as floating point (labelled `FP64` etc., sign/exponent/mantissa set apart, not clickable), and `BEYOND n BITS` for a whole number too big for the word.
- `settings/` — the JSON settings file, in Prism's and ApeMail's pattern: `Defaults`, `Load` (falls back to defaults, then `normalise` replaces any unusable value), atomic `Save`, `Prepared` for the path (`os.UserConfigDir()/digitron/settings.json`).
- `desktop/` — on Linux, writes the desktop entry and the 256×256 logo into `$XDG_DATA_HOME` (`~/.local/share`) at every start, pointing at the running copy or at `$APPIMAGE`, as Delta does. No dependency on the UI; it is given the icon as PNG data (`ui.IconPNG`). `TestPackagedEntryAgrees` keeps it in step with `packaging/digitron.desktop`. `desktop.Name` is also Qt's application and desktop file name, which Wayland matches windows by.
- `atomicfile/` — replaces a file whole through a temporary file beside it; used by `settings` and `desktop`.
- `sound/` — the key chirp and menu tick, embedded with `//go:embed` and played from memory through Qt Multimedia `QAudioSink`s (one pool of voices per sound, each at its own sample rate). `sound/gen` synthesises `tick.wav`; `key.wav` is cut from a recording with the ffmpeg command in the `sound.go` package comment.
- `main.go` (startup, the `app` struct, Qt's application and desktop file names, the desktop entry, settings, starting the sound), `draw.go` (scaling the frame to the window, the icon), `input.go` (mouse and keyboard), `wordmenu.go` (the menu's handlers) — the Qt layer.
- `packaging/`, `.github/workflows/` — the AppImage and macOS app, ported from Delta's: `appimage.sh` (linuxdeploy, run by the workflow on Ubuntu 22.04 with Qt's own build of Qt 6.9), `macos.sh` (macdeployqt, Homebrew's `qtbase` and `qtmultimedia`), the desktop entry, `AppRun`, and `macos/Info.plist` and `macos/digitron.png` (drawn by `make icons`, checked by `TestPackagedIconsAreCurrent`). Both scripts refuse to finish without a Qt Multimedia backend plugin (FFmpeg in the AppImage; Apple's on macOS, as Homebrew builds Qt Multimedia there without FFmpeg), and check that the result starts. `make appimage` fails on Arch: its Qt Multimedia uses the system FFmpeg, whose dependency tree includes a library that crashes the loader once linuxdeploy has patched it. That is expected; releases come from the workflow.
- `docs/previews/` — images drawn by `ui/preview_test.go`; the README's screenshot and logo come from here. `docs/concept.png` is the design concept the face was drawn from.

## Technical debt

Pay technical debt down as part of every feature and fix; do not let cruft build up.

- When a feature is removed or changed, remove what it leaves unused: glyphs in `font.go`, fields of `View`, engine methods, settings, tests of old behaviour. Keep `go vet`, `staticcheck ./...` and `deadcode -test ./...` clean.
- Keep files to one concern; split one that grows a second.
- Keep this file, the README and the previews (`make screenshot`) in step with the code in the same change.
- Anything made to help with the work that is worth running again (a generator, a sound cut) goes in the repository with a make target, a `go:generate` line or a note here, never only in a scratch folder.
- Refactoring changes no behaviour; check with the tests and that `make screenshot` leaves the previews byte-identical.
- Keep test files to one topic too (in `calc`: `exact_test.go` for the famous bugs and entry, `display_test.go` for fitting the display, `repeat_test.go` for repeated =, `word_test.go` for DEC without a word size, `float_test.go` for IEEE 754).

## Drawing

- Everything visible is drawn in `ui` at art-pixel resolution with `canvas` primitives (`fill`, `bevel`, `text`, `tiny`, `key`) and the palette at the top of `render.go`. Never draw with Qt directly; new UI elements (like the menu) are pixel art in `ui`, styled like the keys.
- `draw.go` stretches the frame to fill the window (each axis independently) using "sharp bilinear": a nearest-neighbour enlargement to a whole multiple, then one smooth shrink. Mouse positions map back with `toVirtual`.
- Hit-testing uses the same rectangles that drawing uses (`Keys`, `MenuItems`, `View.Bits`).

## Input

- A key acts as it goes down, mouse or keyboard, with its chirp at the same moment; releasing the mouse only lets the key back up. Do not move the action to the release: dragging off a key would then play the chirp and do nothing. Word-size menu items are picked on press too.

## Sound

- Qt Multimedia takes about a second to start (its first `QAudioSink`). It is started from a single-shot timer after the first paint so the window opens at once; until then `app.sounds` is nil and every `Player` method is nil-safe. Do not move sound start back before `Show()`.
- `QSoundEffect` plays only from files; it was replaced because nothing may be written to disk. Do not reintroduce files in caches or temp folders.
- After changing `sound/gen`, run `go generate ./sound`; `TestUpToDate` fails if `tick.wav` drifts from the generator.
- Pitch variation is per sound (`keySpread`, `tickSpread`); the user wants the key chirp almost constant.

## Commands

```sh
make build        # go build -ldflags "-s -w" -o digitron .
make run
make test         # go test -race ./...
make screenshot   # redraw docs/previews
make icons        # redraw packaging/macos/digitron.png from the logo
make appimage     # packaging/appimage.sh (releases: the AppImage workflow)
make macos        # packaging/macos.sh, on a Mac with Apple Silicon
go generate ./sound
```

A PostToolUse hook in `.claude/settings.json` runs `make build` after every edit to a `.go` file, `go.mod`, `go.sum` or the `Makefile`, and reports build failures.

## Building with miqt

- miqt is cgo: building needs a C++ compiler and the Qt 6 development headers for `Qt6Widgets` and `Qt6MultimediaWidgets`, found via `pkg-config`. `README.md` lists the packages per distribution; keep it, the workflows (the `aqt` modules, the Homebrew formulae) and the packaging scripts' checks in step if the Qt modules used change.
- macOS builds turn off Go's asynchronous preemption (see `packaging/macos.sh`): its signal crashed Qt programs there.
- The first build of the binding takes several minutes and prints many warnings from the Qt headers; search the output for `error`. Avoid `go clean -cache`.
- Linux is the only platform tested. Avoid Linux-only assumptions where a portable Go or Qt API exists.

## miqt conventions

- Import as `qt "github.com/mappu/miqt/qt6"` and `github.com/mappu/miqt/qt6/multimedia`. Never Qt 5's `github.com/mappu/miqt/qt`, which `goimports` picks for a bare `qt.` (and which then compiles a whole second binding); write Qt imports by hand.
- Overloaded C++ constructors and methods become numbered Go functions (`NewQAudioSink3`, `NewQRect4`); check the generated source for the right number rather than guessing.
- Memory: objects from `NewQ...` constructors are never freed by Go, so short-lived ones need `Delete()`. Values returned by methods and static functions (`ev.Position()`, `ev.AngleDelta()`, `QPixmap_FromImage(...)`) already have a finalizer (`GoGC()` in the generated code); calling `Delete()` on them is a double free that crashes at a later garbage collection. Check the generated function before deleting anything.
- Do not delete a Qt object from inside one of its own signal handlers (for example a `QTimer` in its `OnTimeout`).
- Qt objects must only be touched from the GUI thread.
