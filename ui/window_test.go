package ui

import "testing"

func TestWindowSize(t *testing.T) {
	for _, c := range []struct {
		name             string
		w, h, oldW, oldH int
		wantW, wantH     int
	}{
		{"already in shape", 2 * W, 2 * H, W, H, 2 * W, 2 * H},
		{"dragged wider", 3 * W, 2 * H, 2 * W, 2 * H, 3 * W, 3 * H},
		{"dragged taller", 2 * W, 3 * H, 2 * W, 2 * H, 3 * W, 3 * H},
		{"dragged narrower", W, 2 * H, 2 * W, 2 * H, W, H},
		{"dragged by a corner, mostly wider", 3 * W, 2*H + 10, 2 * W, 2 * H, 3 * W, 3 * H},
		{"dragged by a corner, mostly taller", 2*W + 10, 3 * H, 2 * W, 2 * H, 3 * W, 3 * H},
		{"keeping the width", 1000, 600, 1000, 600, 1000, (1000*H + W/2) / W},
		{"rounded to the nearest pixel", 500, 0, 0, 0, 500, (500*H + W/2) / W},
	} {
		w, h := WindowSize(c.w, c.h, c.oldW, c.oldH)
		if w != c.wantW || h != c.wantH {
			t.Errorf("%s: WindowSize(%d, %d, %d, %d) = %d×%d, want %d×%d", c.name, c.w, c.h, c.oldW, c.oldH, w, h, c.wantW, c.wantH)
		}
	}
}

// TestInShape checks that every size WindowSize makes is in shape, and that
// a size a few pixels out is not.
func TestInShape(t *testing.T) {
	for w := W; w < 4*W; w++ {
		if sw, sh := WindowSize(w, 0, 0, 0); !InShape(sw, sh) {
			t.Fatalf("%d×%d, from WindowSize, is not in shape", sw, sh)
		}
	}
	for h := H; h < 4*H; h++ {
		if sw, sh := WindowSize(0, h, 0, 0); !InShape(sw, sh) {
			t.Fatalf("%d×%d, from WindowSize, is not in shape", sw, sh)
		}
	}
	if InShape(2*W, 2*H+3) || InShape(2*W+3, 2*H) {
		t.Error("a size 3 pixels out is in shape")
	}
}
