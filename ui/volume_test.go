package ui

import (
	"image"
	"testing"
)

// TestVolumeAt checks that a press at each step's mark picks that step,
// that a press between marks picks the nearer, and that presses past the
// ends stop at them.
func TestVolumeAt(t *testing.T) {
	bottom := VolumeTrack.Max.Y - volKnobH/2
	for l := 0; l <= MaxVolume; l++ {
		y := bottom - l*volStep
		if got := VolumeAt(y); got != l {
			t.Errorf("VolumeAt(mark %d) = %d", l, got)
		}
		if got := VolumeAt(y - 2); got != l && l < MaxVolume {
			t.Errorf("VolumeAt(just above mark %d) = %d", l, got)
		}
	}
	if got := VolumeAt(VolumeTrack.Min.Y - 50); got != MaxVolume {
		t.Errorf("above the track: %d, want %d", got, MaxVolume)
	}
	if got := VolumeAt(VolumeTrack.Max.Y + 50); got != 0 {
		t.Errorf("below the track: %d, want 0", got)
	}
}

// TestPopupsInside checks that the pop-ups lie inside the calculator.
func TestPopupsInside(t *testing.T) {
	body := image.Rect(0, 0, W, H)
	for name, r := range map[string]image.Rectangle{"menu": MenuRect, "volume slider": VolumeRect} {
		if !r.In(body) {
			t.Errorf("the %s runs off the calculator: %v", name, r)
		}
	}
}
