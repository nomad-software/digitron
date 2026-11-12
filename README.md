# <img src="docs/previews/logo.png" alt="" width="40" height="40"> Digitron

A retro calculator for programmers. It looks like a pocket calculator from
the eighties, drawn pixel by pixel, and it does the arithmetic
correctly. It is written in Go with a Qt 6 interface using the
[qute](https://github.com/nomad-software/qute) bindings, for Linux.

![Digitron in hexadecimal at 64 bits, showing DEADBEEF XOR CAFE = DEAD7411 on its main display and the bits of the result on the binary display below](docs/previews/screenshot.png)

## Why Digitron

- **Exact, not nearly.** Digitron works in exact fractions, not floating
  point, so `0.1 + 0.2` is `0.3` and `1 ÷ 3 × 3` is `1`. Decimals that repeat
  are shown with a bar over the part that repeats, so `1 ÷ 7` is
  0.1̅4̅2̅8̅5̅7̅, not 0.14285714285714285.
- **Made for programmers.** Octal, decimal and hexadecimal are one key
  away, and binary is on show all the time. In OCT and HEX, word sizes run
  from 8 to 64 bits in two's complement, and values wrap at the word size
  the way they do in your code. DEC has no word size: `2⁶³ + 1` and
  `1 << 100` come out exactly, as do numbers of up to about 315,000 digits.
- **Every bit on show.** The second display shows the bits of the current
  value, and clicking a bit flips it. A fraction is shown as the IEEE 754
  floating-point number your code would hold, sign, exponent and mantissa
  set apart.
- **A calculator you want to look at.** Digitron is real low-resolution
  pixel art, scaled to sharp square pixels at any size, with keys that move
  and chirp when you press them.
- **Native and small.** It is a single compiled program, not a web page in a
  window. It starts at once and needs only Qt installed.
- **Nothing phones home.** No account, no telemetry and no network at all.

## Features

- **Bases:** OCT, DEC and HEX. Switching base keeps the value, except
  that OCT and HEX work in whole numbers, so a fraction is dropped. There is
  no binary base: the binary display always shows the bits, and a binary
  number is entered by clicking them, or pasted with `0b` in front. Keys
  that mean nothing in the current base are greyed out.
- **Word sizes:** 8, 16, 32 and 64 bits, chosen from a menu on the WRD key.
  The status line shows the base and the word size. OCT and HEX keep to
  it, and switching to them from DEC drops a fraction and wraps the value into
  the word. DEC has no word size: it works with whole numbers and fractions of
  any size, exactly, and the word size says how the binary display shows
  them. Choosing a word size with a fraction showing makes it the nearest
  number that size's floating-point format holds, marked `≈` if that changed
  it: `0.1` at 16 bits becomes `0.0999755859375`. A fraction too big for the
  format, such as 70000.5 at 16 bits, is an error. DEC still has one limit,
  so that a long shift or squaring a number again and again cannot freeze
  the calculator: a result of more than about 315,000 digits (2²⁰ bits, top
  or bottom of a fraction), such as `1 << 1048576`, is the error `TOO BIG`,
  and a digit typed or a number pasted past it is not taken.
- **The binary display** shows a whole number's bits in two's complement,
  grouped in fours. When a DEC result has a fractional part it switches to
  floating point, labelled `FP64`, `FP32`, `FP16` or `FP8` for the word
  size: IEEE 754 double, single or half precision, or for 8 bits FP8 E5M2.
  The bits are split into sign, exponent and mantissa, and are the nearest
  float to the exact value, rounding as IEEE 754 does, down to subnormals and
  up to infinity, which is labelled `+INF` or `-INF`. There is no NaN: what
  would make one in your code, such as `0 ÷ 0` or the square root of a
  negative number, is an error, shown in the dim green of the status line.
  Its bits cannot be clicked. A whole number too big for the word size has
  no bits worth showing, and the display says `BEYOND 64 BITS`
  (or the size chosen) until it fits again; the status line then shows only
  the base.
- **Bitwise operators:** AND, OR, XOR, NOT and shifts. `>>` is an arithmetic
  shift, so it keeps the sign.
- **Precedence like C:** `|` is lower than `^`, which is lower than `&`, then
  shifts, then `+` and `−`, then `×` and `÷`. `1 + 2 << 3` is 24.
- **Parentheses**, with the number still open shown on the display.
- **Square roots** worked out to 64 decimal places, so that what is made
  from them stays accurate, and marked with `≈` because they are not
  exact.
- **Percent** like a desk calculator: `200 + 10%` is 220, `50 × 10%` is 5.
- **Memory:** MC, MR, M+ and M−.
- **The expression** being worked out is shown above the result, and stays
  there after `=`.
- **Repeated `=`** does the last operation again, as on a desk calculator:
  `2 + 3 =` gives 5, and each `=` after it adds 3 again (8, 11, …). Type a
  new number and press `=` to do it to that instead: `2 + 3 = 7 =` is 10. It
  is the last operation outside any brackets, so `1 + 2 × 3 = =` multiplies
  by 3 again. AC forgets it.
- **Repeating decimals** are shown exactly, with a bar over the digits that
  repeat for ever: `1 ÷ 3` is `0.3̅`, `1 ÷ 7` is `0.1̅4̅2̅8̅5̅7̅`. Some repeat in
  long blocks, and then the bar runs over all of them: `45 ÷ 85` (9/17)
  repeats 16 digits, `5294117647058823`, so all 16 are barred. Ctrl+C copies
  such a number as its fraction.
- **The whole display is used:** a result is shown exactly when it fits the
  27 characters of the main display, and otherwise rounded to as many
  decimal places as fill it, wherever the point is, with `≈` shown. So
  `1 ÷ 97` is `0.0103092783505154639175258` and `123456789 ÷ 97` is
  `1272750.4020618556701030928`. A number so small that fewer than six of its
  digits would be left after the zeros is shown in scientific notation
  instead, again filling the display: `1 ÷ 97 ÷ 10²⁷` is
  `1.030927835051546391753E-29`, and `1 ÷ 10³⁰`, exactly, `1E-30`. A whole
  number is never rounded: a long one, as DEC can make, or a long number
  being typed, wraps onto smaller lines, which you scroll with the mouse
  wheel.
- **Copy and paste:** Ctrl+C copies the exact value, as a fraction such as
  `1/7` or `1/97` if its decimal repeats or is rounded on the display; a
  square root is copied as it is shown. Ctrl+V takes numbers with `0x`,
  `0o` or `0b` in front, with `_`, `,` or spaces between digits, and in
  decimal fractions such as `22/7` and scientific notation such as
  `1.5e-30`.
- **Sound:** every key press makes a short retro chirp, like an old
  mainframe terminal, and moving through the word-size menu or the volume
  slider makes a soft, dull tick. Each plays at a very slightly different pitch every time, so
  it never sounds mechanical. Both sounds are built into the program. The
  VOL key opens a slider, styled like the word-size menu, that sets their
  volume in ten steps; at the bottom, MUTE, there is no sound at all. The
  volume is remembered between runs. The sound starts a moment
  after the window opens, so that the window can open at once; a key
  pressed in that first second makes no sound.
- **Resizing:** the window keeps the calculator's shape, so every pixel
  stays square. While it is being dragged it can take any shape; a moment
  after it stops, the side dragged further keeps its length and the other
  follows it. A maximised window fills the screen and is stretched to fit it.
- **Remembers** the window size, the word size and the volume between runs. See
  [Settings](#settings).

## Keyboard shortcuts

Almost everything can be done from the keyboard.

| Keys | Does |
|---|---|
| 0–9, A–F | Type a digit (A–F in HEX only). |
| `.` or `,` | Decimal point (DEC only). |
| `+` `-` `*` or `x` `/` | Add, subtract, multiply, divide. |
| `=` or Return | Equals. |
| `&` `\|` `^` `~` | AND, OR, XOR, NOT. |
| `<` `>` | Shift left, shift right. |
| `(` `)` | Open and close parentheses. |
| `%` | Percent. |
| `r` | Square root. |
| `n` or F9 | Change sign (+/−). As in the Windows calculator, 0 has no sign: type the digits first, then change it. |
| Backspace | Delete the last digit typed. |
| Delete | Clear the entry (C). |
| Escape | Clear everything (AC). Memory, base and word size are kept. |
| F5, F6, F7 | HEX, DEC, OCT. |
| `w` | Open the word-size menu. |
| `s` | Open the volume slider. |
| Ctrl+C | Copy the exact value. |
| Ctrl+V | Paste a number. |
| Ctrl+Q | Quit. |

**In the word-size menu**

| Keys | Does |
|---|---|
| Up, Down, Home, End | Move the highlight. |
| Return, Enter or Space | Choose the highlighted size. |
| `8`, `1`, `3`, `6` | Choose 8, 16, 32 or 64 bits straight away. |
| Escape or `w` | Close the menu without changing anything. |

**On the volume slider**

| Keys | Does |
|---|---|
| Up or Right, Down or Left | One step louder or quieter. |
| Home or Page Up, End or Page Down | The loudest, or mute. |
| `0`–`9` | That step straight away (0 is mute). |
| Return, Space, Escape or `s` | Close the slider. |

With the mouse, click or drag along the slider; the volume changes when
the button is let go, with a tick at the new level. The wheel over it, like
the keys, changes it a step at a time.

The mouse works too: click the keys, click a bit on the binary display to
flip it, and scroll the main display when a result is too long to show at
once.

## Requirements

Digitron is for Linux only.

Building needs:

- Go 1.27 or later
- A C++17 compiler (GCC or Clang)
- `pkg-config`
- Qt 6.12 or later: the base development files (Qt 6 Widgets) and the
  Multimedia development files, for the sounds. The qute bindings are made
  from Qt 6.12 and do not compile with an older Qt.

Install them with your distribution's package manager:

```sh
# Arch, CachyOS, Manjaro
sudo pacman -S go gcc pkgconf qt6-base qt6-multimedia qt6-multimedia-ffmpeg

# Debian, Ubuntu
sudo apt install golang-go build-essential pkg-config qt6-base-dev qt6-multimedia-dev

# Fedora
sudo dnf install golang gcc-c++ pkgconf-pkg-config qt6-qtbase-devel qt6-qtmultimedia-devel

# openSUSE
sudo zypper install go gcc-c++ pkgconf-pkg-config qt6-base-devel qt6-multimedia-devel
```

If your distribution's Go is older than 1.27, install Go from
[go.dev/dl](https://go.dev/dl/) instead. If its Qt is older than 6.12, as
it is on stable releases of most distributions for now, install Qt 6.12
with [Qt's online installer](https://www.qt.io/download-qt-installer) or
[aqtinstall](https://github.com/miurahr/aqtinstall), as the AppImage
workflow does, and point `pkg-config` at it with
`PKG_CONFIG_PATH=<Qt>/lib/pkgconfig`; or use the AppImage.

Running a built binary on another machine only needs the Qt 6 (6.12 or later) runtime
libraries (`qt6-base` and `qt6-multimedia` on Arch, `libqt6widgets6` and
`libqt6multimedia6` on Debian and Ubuntu, `qt6-qtbase` and
`qt6-qtmultimedia` on Fedora, `libQt6Widgets6` and `libQt6Multimedia6` on
openSUSE).

### Checking

Check that Qt can be found with:

```sh
pkg-config --modversion Qt6Widgets Qt6Multimedia
```

This should print two versions, 6.12 or later.

## Building

```sh
make build
```

This is the same as:

```sh
go build -ldflags "-s -w" -o digitron .
```

The `-s -w` flags build without the symbol table and debug information, which
roughly halves the size of the binary, to about 19 MB. It is the only
file a build leaves in the project's folder (and `.gitignore` leaves it
out); Go keeps what it compiles in its own build cache.

The sounds, `sound/key.wav` and `sound/tick.wav`, are committed and
embedded in the binary. The chirp is cut from a recording; the ffmpeg
command that cut it is in `sound/sound.go`. The tick is made by a small
synthesiser in `sound/gen`. After changing the synthesiser, remake the tick
with:

```sh
go generate ./sound
```

A test fails if the committed tick no longer matches what the synthesiser
makes.

The first build compiles the C++ part of the Qt bindings and takes a couple
of minutes. Later builds use the Go build cache and take a few seconds.

## AppImage

Releases for Linux are an [AppImage](https://appimage.org): one file with
Digitron and the Qt libraries it needs inside it, which runs without
installing anything.

```sh
chmod +x Digitron-*-x86_64.AppImage
./Digitron-*-x86_64.AppImage
```

It is built by a GitHub workflow (`.github/workflows/appimage.yml`), which runs
`packaging/appimage.sh` on Ubuntu 22.04 with Qt's own build of Qt 6.12:

- Pushing a tag that starts with `v` builds it and attaches it to a release of
  that name: `git tag v1.0.0 && git push origin v1.0.0`.
- It can also be run by hand from the Actions tab of the repository (AppImage >
  Run workflow), which leaves the AppImage as an artifact of the run to
  download.

It runs under both X11 and Wayland, using Wayland where the session is one.
To use the other, set `QT_QPA_PLATFORM` to `xcb` or `wayland` when starting
it. Qt Multimedia, which plays the sounds, and its FFmpeg backend are in it;
the script will not make an AppImage without them, nor one that does not
start.

The C library of the system (glibc) is not put in the AppImage, as it cannot
be reliably: it has to match the system's graphics drivers. It is built on an
old distribution instead, and runs on any with glibc 2.35 or later (Ubuntu
22.04, Debian 12, Fedora 36 and anything newer, or rolling).

The script builds in `/tmp/digitron-build` (or the folder `BUILD` names),
and leaves only the AppImage, `Digitron-<version>-x86_64.AppImage`, in the
project's folder, where `.gitignore` leaves it out. The tools it downloads
are kept there between runs, and `make clean` removes it all.

`make appimage` runs the same script locally, but an AppImage built on a newer
system only runs on systems at least as new. On a distribution whose Qt
Multimedia uses the system's FFmpeg, such as Arch, it is not made at all:
that FFmpeg brings a large tree of libraries with it, one of which does not
survive being packed, and the AppImage crashes as it starts, which the
script's check catches. Use the workflow, with Qt's own build, for releases.

## Running

```sh
./digitron
```

or build and run in one step with:

```sh
make run
```

Every time it starts Digitron writes a desktop entry for itself,
`~/.local/share/applications/digitron.desktop` (or under `$XDG_DATA_HOME`),
with its logo as the icon in
`~/.local/share/icons/hicolor/256x256/apps/digitron.png`. That puts it among
the desktop's applications, to be started from the menu and kept in a dock
or sidebar. The entry starts the copy that was run last, from wherever it was
and whatever it is called: the AppImage, if it was run as one, and otherwise
the program itself. The files are only rewritten when that changes. Delete
the two files to take it out of the menu again; they come back the next time
Digitron is run.

### Settings

The window size, whether the window was maximised, the word size and the
volume are saved when Digitron closes; the word size also as soon as it is
chosen, and the volume when the slider is closed.
They are kept in a JSON file, `digitron/settings.json`, in the place the
system has for the configuration of programs: `~/.config` (or
`$XDG_CONFIG_HOME`). If there is
no such place, the file is kept in the current directory.

```json
{
  "width": 744,
  "height": 796,
  "maximized": false,
  "wordSize": 64,
  "volume": 10
}
```

`volume` runs from `0`, no sound, to `10`, the loudest.

A value that is missing or cannot be used is replaced by its default, so the
file can be edited by hand or deleted to start afresh.

## Testing

```sh
make test
```

This runs every test with Go's race detector. Among them:

- **Famous bugs**, kept from ever coming back: the Windows calculator's
  `√4 − 2`, Windows 3.1's `2.01 − 2`, the Pentium FDIV bug, Excel 2007's
  `850 × 77.1`, the Patriot missile's clock, and the usual floating-point
  traps such as `0.1 + 0.2`.
- **Problems made to break floating point**, which exact fractions get
  exactly right: Rump's polynomial and Muller's recurrence.
- **Programmer's integers** in OCT and HEX checked against Go's own 8-, 16-,
  32- and 64-bit integers on thousands of random operands, and C's traps:
  `INT_MIN ÷ −1`, division towards zero, arithmetic shifts, precedence, the
  Ariane 5 conversion and the 2038 overflow.
- **Limits:** the size limit in DEC, reached by shifting and by squaring
  through memory, and pastes too big to read, such as `1e1000000000`.
- **Random key presses**, thousands of sequences, checked after every key
  for a state the calculator must never be in. To search for longer:

```sh
make fuzz               # a minute
make fuzz FUZZTIME=10m
```

The screenshot and logo in this file, and pictures of a few other states of
the calculator, are drawn by a test into `docs/previews`, and are made again
with:

```sh
make screenshot
```

The logo is also the window icon.

The window itself, painted through Qt, has benchmarks of its own, which run
headlessly on Qt's offscreen platform: painting at 1× and 3× and while being
resized, the frame's drawing alone, which is pure Go, and a key and a click
sent to the window. They measure what the Qt bindings cost, to compare one
version of them, or one binding, with another:

```sh
make bench
```

`docs/concept.png` is the concept the calculator's face was drawn from.

## License

Digitron is free software, licensed under the
[GNU General Public License, version 3](LICENSE).
