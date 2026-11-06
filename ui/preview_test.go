package ui

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"digitron/calc"
)

// TestPreview renders sample frames as PNGs when PREVIEW_DIR is set.
func TestPreview(t *testing.T) {
	dir := os.Getenv("PREVIEW_DIR")
	if dir == "" {
		t.Skip("set PREVIEW_DIR to write preview PNGs")
	}
	scenes := map[string]func(e *calc.Engine, v *View){
		"basic": func(e *calc.Engine, v *View) {
			for _, k := range []string{"1", "2", "add", "(", "3", "mul", "7"} {
				Press(e, k)
			}
			v.Pressed = "7"
		},
		"repeat": func(e *calc.Engine, v *View) {
			for _, k := range []string{"1", "div", "7", "eq", "m+"} {
				Press(e, k)
			}
		},
		"hex64": func(e *calc.Engine, v *View) {
			e.SetWidth(64)
			Press(e, "hex")
			for _, k := range []string{"hD", "hE", "hA", "hD", "hB", "hE", "hE", "hF", "neg"} {
				Press(e, k)
			}
		},
		"big": func(e *calc.Engine, v *View) {
			for _, k := range []string{"2", "sqrt"} {
				Press(e, k)
			}
		},
		"bin32": func(e *calc.Engine, v *View) {
			e.SetWidth(32)
			Press(e, "hex")
			for _, k := range []string{"hC", "hA", "hF", "hE", "hB", "hA", "hB", "hE"} {
				Press(e, k)
			}
		},
		"screenshot": func(e *calc.Engine, v *View) {
			Press(e, "hex")
			for _, k := range []string{"hD", "hE", "hA", "hD", "hB", "hE", "hE", "hF", "xor", "hC", "hA", "hF", "hE", "eq"} {
				Press(e, k)
			}
		},
		"float64": func(e *calc.Engine, v *View) {
			for _, k := range []string{"1", "div", "1", "0", "eq"} {
				Press(e, k)
			}
		},
		"float16": func(e *calc.Engine, v *View) {
			e.SetWidth(16)
			for _, k := range []string{"1", "div", "3", "eq"} {
				Press(e, k)
			}
		},
		"beyond": func(e *calc.Engine, v *View) {
			for _, k := range []string{"1", "shl", "1", "0", "0", "eq"} {
				Press(e, k)
			}
		},
		"wsmenu": func(e *calc.Engine, v *View) {
			for _, k := range []string{"4", "2"} {
				Press(e, k)
			}
			v.OpenMenu(e)
			v.MenuMove(-1)
		},
		"volume": func(e *calc.Engine, v *View) {
			for _, k := range []string{"4", "2"} {
				Press(e, k)
			}
			v.Volume = 7
			v.OpenVolume()
		},
		"mute": func(e *calc.Engine, v *View) {
			v.OpenVolume()
		},
	}
	for name, f := range scenes {
		e := calc.New()
		v := &View{}
		f(e, v)
		img := image.NewRGBA(image.Rect(0, 0, W, H))
		Render(img, e, v)
		writePNG(t, filepath.Join(dir, name+".png"), img, 3)
	}
	if err := os.WriteFile(filepath.Join(dir, "logo.png"), IconPNG(8), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writePNG writes img enlarged s times with nearest-neighbour sampling.
func writePNG(t *testing.T, path string, img *image.RGBA, s int) {
	b := img.Bounds()
	big := image.NewRGBA(image.Rect(0, 0, b.Dx()*s, b.Dy()*s))
	for y := range b.Dy() * s {
		for x := range b.Dx() * s {
			big.Set(x, y, img.At(x/s, y/s))
		}
	}
	fh, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()
	if err := png.Encode(fh, big); err != nil {
		t.Fatal(err)
	}
}

// TestDisplayCellsFit checks that the main display has room in its big
// digits for all the characters the engine fits a decimal to.
func TestDisplayCellsFit(t *testing.T) {
	if bigFit < calc.DisplayCells {
		t.Errorf("the main display holds %d big characters, the engine fits decimals to %d", bigFit, calc.DisplayCells)
	}
}
