package ui

import (
	"fmt"
	"image"
)

// The volume slider pops up above the sound key, in a pop-up like the
// word-size menu's: an upright groove of LCD glass, lit from the bottom up
// to the level, with a notch mark at each step, a small dark key as the
// knob, and the level, or MUTE, on a little LCD beneath.

// MaxVolume is the loudest step of the slider; 0 is silent.
const MaxVolume = 10

const (
	volW       = 48
	volStep    = 6 // pixels between steps
	volKnobW   = 22
	volKnobH   = 8
	volTrackH  = MaxVolume*volStep + volKnobH
	volGrooveW = 4
	volBezel   = 4 // the readout's frame: outline, two pixels of grey, shadow
	volLCDH    = glyphH + 4 + 2*volBezel
	volLCDGap  = 4
)

var (
	// VolumeRect is the slider's panel and VolumeTrack the part of it that
	// a press sets the level from.
	VolumeRect, VolumeTrack = buildVolume()
)

func buildVolume() (image.Rectangle, image.Rectangle) {
	h := popupPad + popupTitleH + volTrackH + volLCDGap + volLCDH + popupPad
	panel := popupRect(keyByID("snd"), volW, h)
	top := popupTop(panel)
	track := image.Rect(panel.Min.X+popupPad, top, panel.Max.X-popupPad, top+volTrackH)
	return panel, track
}

// VolumeAt returns the level whose knob position is nearest the height y,
// clamped to the slider's range.
func VolumeAt(y int) int {
	bottom := VolumeTrack.Max.Y - volKnobH/2 // the centre of the knob at 0
	return max(0, min(MaxVolume, (bottom-y+volStep/2)/volStep))
}

// OpenVolume shows the volume slider.
func (v *View) OpenVolume() { v.VolumeOpen = true }

// CloseVolume hides the volume slider.
func (v *View) CloseVolume() { v.VolumeOpen = false }

func (c canvas) volume(v *View) {
	r := VolumeRect
	c.popup(r, keyByID("snd"), "VOL")

	t := VolumeTrack
	cx := (t.Min.X + t.Max.X) / 2
	bottom := t.Max.Y - volKnobH/2
	level := max(0, min(MaxVolume, v.Volume))

	// A mark either side of the groove at each step, longer every five.
	for i := 0; i <= MaxVolume; i++ {
		y := bottom - i*volStep
		n := 2
		if i%5 == 0 {
			n = 4
		}
		x0 := cx - volGrooveW/2 - 2
		x1 := cx + volGrooveW/2 + 2
		c.hline(x0-n, x0, y, cBodyLo2)
		c.hline(x1, x1+n, y, cBodyLo2)
	}

	// The groove: sunken glass, lit from the bottom up to the level in the
	// dim green of the status line, so that it does not outshine a value.
	g := image.Rect(cx-volGrooveW/2, bottom-MaxVolume*volStep-1, cx+volGrooveW/2, bottom+2)
	c.fill(g.Inset(-1), cOutline)
	c.fill(g, cGlass)
	if level > 0 {
		c.fill(image.Rect(g.Min.X, bottom-level*volStep, g.Max.X, g.Max.Y), cLCDDim)
	}

	// The knob, a small dark key centred on the level.
	ky := bottom - level*volStep - volKnobH/2
	c.key(Key{Rect: image.Rect(cx-volKnobW/2, ky, cx+volKnobW/2, ky+volKnobH), Dark: true, Scale: 1}, false, false, true)

	// The readout, a little display framed like the main ones.
	l := image.Rect(r.Min.X+popupPad, t.Max.Y+volLCDGap, r.Max.X-popupPad, t.Max.Y+volLCDGap+volLCDH).Inset(volBezel)
	c.lcd(l, volBezel)
	msg := fmt.Sprint(level)
	if level == 0 {
		msg = "MUTE"
	}
	c.text(l.Min.X+(l.Dx()-textWidth(msg, 1)+1)/2, l.Min.Y+2, msg, 1, cLCD)
}
