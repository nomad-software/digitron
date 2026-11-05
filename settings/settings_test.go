package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"digitron/ui"
)

func TestPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/somewhere")
	t.Setenv("HOME", "/home/someone")
	path, err := configPath()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, filepath.Join("digitron", "settings.json")) {
		t.Errorf("configPath = %q", path)
	}
}

func TestPrepare(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "settings.json")
	if err := prepare(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() || info.Mode().Perm() != 0o700 {
		t.Errorf("folder is %v", info.Mode())
	}
}

func TestLoadMissing(t *testing.T) {
	s := Load(filepath.Join(t.TempDir(), "settings.json"))
	if !reflect.DeepEqual(s, Defaults()) {
		t.Errorf("Load of a missing file = %+v, want the defaults", s)
	}
}

func TestLoadUnreadable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if s := Load(path); !reflect.DeepEqual(s, Defaults()) {
		t.Errorf("Load of a broken file = %+v, want the defaults", s)
	}
}

func TestSaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	want := Settings{Width: 1000, Height: 600, Maximized: true, WordSize: 16, Sound: false}
	if err := want.Save(path); err != nil {
		t.Fatal(err)
	}
	if got := Load(path); got != want {
		t.Errorf("Load = %+v, want %+v", got, want)
	}
	// Nothing is left beside the file by the atomic replace.
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("folder holds %d files, want 1", len(entries))
	}
}

func TestNormalise(t *testing.T) {
	d := Defaults()
	for _, c := range []struct {
		name string
		in   Settings
		want Settings
	}{
		{"too narrow", Settings{Width: 100, Height: 900, WordSize: 32}, Settings{Width: d.Width, Height: d.Height, WordSize: 32}},
		{"too short", Settings{Width: 900, Height: 100, WordSize: 32}, Settings{Width: d.Width, Height: d.Height, WordSize: 32}},
		{"smallest", Settings{Width: 372, Height: 398, WordSize: 8}, Settings{Width: 372, Height: 398, WordSize: 8}},
		{"word size too big", Settings{Width: 900, Height: 900, WordSize: 128}, Settings{Width: 900, Height: 900, WordSize: d.WordSize}},
		{"unbounded word size", Settings{Width: 900, Height: 900, WordSize: 0}, Settings{Width: 900, Height: 900, WordSize: d.WordSize}},
		{"odd word size", Settings{Width: 900, Height: 900, WordSize: 12}, Settings{Width: 900, Height: 900, WordSize: d.WordSize}},
	} {
		s := c.in
		s.normalise()
		if s != c.want {
			t.Errorf("%s: normalise(%+v) = %+v, want %+v", c.name, c.in, s, c.want)
		}
	}
}

func TestLoadPartial(t *testing.T) {
	// A file from before a setting existed keeps its default.
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"width": 800, "height": 900}`), 0o600); err != nil {
		t.Fatal(err)
	}
	want := Settings{Width: 800, Height: 900, WordSize: defaultWordSize, Sound: true}
	if got := Load(path); got != want {
		t.Errorf("Load = %+v, want %+v", got, want)
	}
}

func TestPrepared(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path, err := Prepared()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "digitron", "settings.json"); path != want {
		t.Errorf("Prepared = %q, want %q", path, want)
	}
	if info, err := os.Stat(filepath.Dir(path)); err != nil || !info.IsDir() {
		t.Errorf("Prepared did not make the folder: %v", err)
	}
}

func TestPreparedWithoutConfigDir(t *testing.T) {
	// With neither $XDG_CONFIG_HOME nor $HOME there is no configuration
	// folder, and the program falls back to the current directory.
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	if _, err := Prepared(); err == nil {
		t.Error("Prepared found a folder with no $HOME")
	}
}

func TestPreparedCannotMakeFolder(t *testing.T) {
	// A file where the folder should be.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "digitron"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", dir)
	if _, err := Prepared(); err == nil {
		t.Error("Prepared made a folder over a file")
	}
}

func TestSaveFails(t *testing.T) {
	// No folder to write in.
	if err := Defaults().Save(filepath.Join(t.TempDir(), "missing", "settings.json")); err == nil {
		t.Error("Save into a missing folder succeeded")
	}
	// A folder, with something in it, where the file should be: the new
	// file cannot replace it, and nothing is left behind.
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.MkdirAll(filepath.Join(path, "x"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := Defaults().Save(path); err == nil {
		t.Error("Save over a folder succeeded")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("Save left %d files behind", len(entries)-1)
	}
}

// TestSizesMatchArt checks the window's default and least sizes against the
// size of the calculator's art, so that a change to the layout is not left
// behind here.
func TestSizesMatchArt(t *testing.T) {
	if minWidth != ui.W || minHeight != ui.H {
		t.Errorf("least size %d×%d, art %d×%d", minWidth, minHeight, ui.W, ui.H)
	}
	if defaultWidth != 2*ui.W || defaultHeight != 2*ui.H {
		t.Errorf("default size %d×%d, want twice the art, %d×%d", defaultWidth, defaultHeight, 2*ui.W, 2*ui.H)
	}
}
