# <img src="docs/previews/logo.png" alt="" width="40" height="40"> Digitron

A retro calculator for programmers. It looks like a pocket calculator from
the eighties, drawn pixel by pixel, and it does the arithmetic
correctly. It is written in Go with a Qt 6 interface using the
[miqt](https://github.com/mappu/miqt) bindings.

![Digitron in hexadecimal at 64 bits, showing DEADBEEF XOR CAFE = DEAD7411 on its main display and the bits of the result on the binary display below](docs/previews/screenshot.png)

## Why Digitron

- **Exact, not nearly.** Digitron works in exact fractions, not floating
  point, so `0.1 + 0.2` is `0.3` and `1 ÷ 3 × 3` is `1`. Decimals that repeat
  are shown with a bar over the part that repeats, so `1 ÷ 7` is
  0.1̅4̅2̅8̅5̅7̅, not 0.14285714285714285.
- **Made for programmers.** Binary, octal, decimal and hexadecimal are one
  key away. Word sizes run from 8 to 64 bits in two's complement, and values
  wrap at the word size the way they do in your code.
- **Every bit on show.** The second display always shows the bits of the
  current value, and clicking a bit flips it.
- **A calculator you want to look at.** Digitron is real low-resolution
  pixel art, scaled to sharp square pixels at any size, with keys that move
  and chirp when you press them.
- **Native and small.** It is a single compiled program, not a web page in a
  window. It starts at once and needs only Qt installed.
- **Nothing phones home.** No account, no telemetry and no network at all.

## Features

- **Bases:** BIN, OCT, DEC and HEX. Switching base keeps the value, except
  that BIN, OCT and HEX work in whole numbers, so a fraction is dropped. Keys
  that mean nothing in the current base are greyed out.
- **Word sizes:** 8, 16, 32 and 64 bits, chosen from a menu on the WS key.
  The status line shows the base and the word size.
- **Bitwise operators:** AND, OR, XOR, NOT and shifts. `>>` is an arithmetic
  shift, so it keeps the sign.
- **Precedence like C:** `|` is lower than `^`, which is lower than `&`, then
  shifts, then `+` and `−`, then `×` and `÷`. `1 + 2 << 3` is 24.
- **Parentheses**, with the number still open shown on the display.
- **Square roots** to 64 decimal places, marked with `≈` because they are
  not exact.
- **Percent** like a desk calculator: `200 + 10%` is 220, `50 × 10%` is 5.
- **Memory:** MC, MR, M+ and M−.
- **The expression** being worked out is shown above the result, and stays
  there after `=`.
- **Long results** wrap onto smaller lines in the main display, which you
  scroll with the mouse wheel.
- **Copy and paste:** Ctrl+C copies the exact value, as a fraction such as
  `1/7` if its decimal repeats. Ctrl+V takes numbers with `0x`, `0o` or `0b`
  in front, with `_`, `,` or spaces between digits, and fractions such as
  `22/7` in decimal.
- **Sound:** every key press makes a short retro chirp, like an old
  mainframe terminal. Moving through the word-size menu makes a soft, dull
  tick. Each plays at a slightly different pitch every time, so it never
  sounds mechanical. Both sounds are built into the program. It can be turned off in the
  [settings](#settings).
- **Resizing:** the calculator stretches to fill the window.
- **Remembers** the window size and the word size between runs. See
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
| `n` or F9 | Change sign (+/−). |
| Backspace | Delete the last digit typed. |
| Delete | Clear the entry (C). |
| Escape | Clear everything (AC). Memory, base and word size are kept. |
| F5, F6, F7, F8 | HEX, DEC, OCT, BIN. |
| `w` | Open the word-size menu. |
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

The mouse works too: click the keys, click a bit on the binary display to
flip it, and scroll the main display when a result is too long to show at
once.

## Requirements

Digitron is developed and tested on Linux.

Building needs:

- Go 1.27 or later
- A C++ compiler (GCC or Clang)
- `pkg-config`
- The Qt 6 base development files (Qt 6 Widgets)
- The Qt 6 Multimedia development files, for the sounds

### Linux

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
[go.dev/dl](https://go.dev/dl/) instead.

Running a built binary on another machine only needs the Qt 6 runtime
libraries (`qt6-base` and `qt6-multimedia` on Arch, `libqt6widgets6` and
`libqt6multimedia6` on Debian and Ubuntu, `qt6-qtbase` and
`qt6-qtmultimedia` on Fedora, `libQt6Widgets6` and `libQt6Multimedia6` on
openSUSE).

### macOS

Digitron has not been tried on macOS, but it uses nothing specific to Linux.
Install the Xcode command line tools, which provide the Clang compiler, and
then the rest with [Homebrew](https://brew.sh):

```sh
xcode-select --install
brew install go pkgconf qt
```

The `qt` formula is Qt 6, Multimedia included. If `pkg-config` cannot find it afterwards, tell it
where Homebrew put Qt:

```sh
export PKG_CONFIG_PATH="$(brew --prefix qt)/lib/pkgconfig:$PKG_CONFIG_PATH"
```

### Checking

On either system, check that Qt can be found with:

```sh
pkg-config --modversion Qt6Widgets Qt6MultimediaWidgets
```

This should print two versions starting with 6.

## Building

```sh
make build
```

This is the same as:

```sh
go build -ldflags "-s -w" -o digitron .
```

The `-s -w` flags build without the symbol table and debug information, which
roughly halves the size of the binary, to about 33 MB.

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

The first build compiles the whole Qt binding and takes several minutes,
printing a lot of compiler warnings from the Qt headers, which can be
ignored. Later builds use the Go build cache and take a few seconds.

## Running

```sh
./digitron
```

or build and run in one step with:

```sh
make run
```

### Settings

The window size, whether the window was maximised, and the word size are
saved when Digitron closes, and the word size also as soon as it is chosen.
They are kept in a JSON file, `digitron/settings.json`, in the place the
system has for the configuration of programs: `~/.config` on Linux (or
`$XDG_CONFIG_HOME`) and `~/Library/Application Support` on macOS. If there is
no such place, the file is kept in the current directory.

```json
{
  "width": 744,
  "height": 824,
  "maximized": false,
  "wordSize": 64,
  "sound": true
}
```

Set `sound` to `false` to turn the sounds off.

A value that is missing or cannot be used is replaced by its default, so the
file can be edited by hand or deleted to start afresh.

## Testing

```sh
make test
```

This runs every test with Go's race detector.

The screenshot and logo in this file, and pictures of a few other states of
the calculator, are drawn by a test into `docs/previews`, and are made again
with:

```sh
make screenshot
```

## License

Digitron is free software, licensed under the
[GNU General Public License, version 3](LICENSE).
