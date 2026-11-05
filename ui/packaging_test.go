package ui

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// The macOS app's icon is the logo enlarged with sharp pixels, centred in a
// 1024-pixel square within the margin Apple's grid gives icons (about a
// tenth of the side each way).
const (
	macIconPath  = "macos/digitron.png"
	macIconSize  = 1024
	macIconScale = 25 // 32 art pixels make 800, leaving a margin of 112
)

func macIcon() *image.RGBA {
	logo := IconAt(macIconScale)
	img := image.NewRGBA(image.Rect(0, 0, macIconSize, macIconSize))
	off := (macIconSize - logo.Bounds().Dx()) / 2
	draw.Draw(img, logo.Bounds().Add(image.Pt(off, off)), logo, image.Point{}, draw.Src)
	return img
}

func encodePNG(t *testing.T, img image.Image) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestWriteIcons is make icons: it draws the macOS app's icon into the
// packaging folder named by DIGITRON_ICONS.
func TestWriteIcons(t *testing.T) {
	dir := os.Getenv("DIGITRON_ICONS")
	if dir == "" {
		t.Skip("set DIGITRON_ICONS to the packaging folder to write the icons")
	}
	path := filepath.Join(dir, macIconPath)
	if err := os.WriteFile(path, encodePNG(t, macIcon()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestPackagedIconsAreCurrent says when the logo has changed without make
// icons being run.
func TestPackagedIconsAreCurrent(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "packaging", macIconPath))
	if err != nil {
		t.Fatalf("%v: run make icons", err)
	}
	defer f.Close()
	got, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	want := macIcon()
	if got.Bounds() != want.Bounds() {
		t.Fatalf("%s is %v, not %v: run make icons", macIconPath, got.Bounds(), want.Bounds())
	}
	for y := range macIconSize {
		for x := range macIconSize {
			r1, g1, b1, a1 := got.At(x, y).RGBA()
			r2, g2, b2, a2 := want.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				t.Fatalf("%s differs from the logo at %d,%d: run make icons", macIconPath, x, y)
			}
		}
	}
}
