package ui

// WindowSize returns the window size nearest w×h that has the shape of the
// calculator, W×H, so that every art pixel is drawn square. The side that
// changed most from oldW×oldH, relative to its length, is kept and the
// other follows it; on a tie, or to keep the width outright (pass oldH = h),
// the width is kept. A size of at least W wide or H high gives at least
// W×H, the least the window may be.
func WindowSize(w, h, oldW, oldH int) (int, int) {
	dw := abs(w-oldW) * H // the changes, compared at the calculator's scale
	dh := abs(h-oldH) * W
	if dw >= dh {
		return w, (w*H + W/2) / W
	}
	return (h*W + H/2) / H, h
}

// InShape reports whether a window of w×h has the calculator's shape, to
// the nearest pixel, as WindowSize makes it.
func InShape(w, h int) bool {
	return abs(w*H-h*W) <= max(W, H)/2
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
