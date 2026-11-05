// Package settings stores the settings of the program, which are kept
// between runs in a JSON file, in a folder for the program in the place the
// system has for the configuration of programs.
// It has no dependency on the UI.
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"

	"digitron/calc"
)

// The default size of the window, twice the size of the calculator's art,
// and the least size it can be, the size of the art.
const (
	defaultWidth  = 744
	defaultHeight = 824
	minWidth      = 372
	minHeight     = 412
)

// defaultWordSize is the word size, in bits, the calculator starts with.
const defaultWordSize = 64

// Settings is the state that is saved to disk.
type Settings struct {
	// Width and Height are the size of the window when it is not maximised.
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximized bool `json:"maximized"`

	// WordSize is the calculator's word size in bits, one of calc.Widths.
	WordSize int `json:"wordSize"`

	// Sound is whether the keys click when they are pressed.
	Sound bool `json:"sound"`
}

// Defaults returns the settings the program starts with.
func Defaults() Settings {
	return Settings{
		Width:    defaultWidth,
		Height:   defaultHeight,
		WordSize: defaultWordSize,
		Sound:    true,
	}
}

// appName is the name of the folder the settings are kept in.
const appName = "digitron"

// fileName is the name of the settings file.
const fileName = "settings.json"

// configPath returns where the settings file is kept: in a folder for the
// program in the place the system has for the configuration of programs, which
// is ~/.config on Linux (or $XDG_CONFIG_HOME) and ~/Library/Application Support
// on macOS.
func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appName, fileName), nil
}

// Prepared returns where the settings file is kept, as configPath does, with
// the folder it is kept in made.
func Prepared() (string, error) {
	path, err := configPath()
	if err != nil {
		return "", err
	}
	if err := prepare(path); err != nil {
		return "", err
	}
	return path, nil
}

// prepare makes the folder the settings file at path is kept in.
func prepare(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o700)
}

// Load reads the settings from the JSON file at path. Defaults are used if
// the file is missing or unreadable, and for any value that is unusable.
func Load(path string) Settings {
	s := Defaults()
	if data, err := os.ReadFile(path); err == nil {
		if json.Unmarshal(data, &s) != nil {
			s = Defaults()
		}
	}
	s.normalise()
	return s
}

// normalise puts a default in place of each value that is unusable.
func (s *Settings) normalise() {
	d := Defaults()
	if s.Width < minWidth || s.Height < minHeight {
		s.Width, s.Height = d.Width, d.Height
	}
	if !slices.Contains(calc.Widths, s.WordSize) {
		s.WordSize = d.WordSize
	}
}

// Save writes the settings to the JSON file at path, replacing it
// atomically, so that a failure leaves the old file whole.
func (s Settings) Save(path string) error {
	// The settings are only numbers and booleans, which JSON always writes.
	data, _ := json.MarshalIndent(s, "", "  ")
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
