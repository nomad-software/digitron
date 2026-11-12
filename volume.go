package main

// Handling of the volume slider, which package ui draws and lays out.

import (
	"image"

	"github.com/nomad-software/qute/qtcore"

	"digitron/ui"
)

// moveVolume moves the slider's knob to level without changing the volume,
// and reports whether it moved.
func (a *app) moveVolume(level int) bool {
	level = max(0, min(ui.MaxVolume, level))
	if level == a.view.Volume {
		return false
	}
	a.view.Volume = level
	return true
}

// setVolume moves the slider to level and, if that changed it, sets the
// volume and plays the tick at it, so that what is heard is what was
// chosen; at 0 nothing is. The keys and the wheel use it, as each of their
// steps is a choice made.
func (a *app) setVolume(level int) {
	if a.moveVolume(level) {
		a.applyVolume()
		a.sounds.Tick()
	}
}

// applyVolume sets the sounds to the slider's level.
func (a *app) applyVolume() {
	a.sounds.SetLoudness(float64(a.view.Volume) / ui.MaxVolume)
}

// endVolumeDrag ends a drag of the knob. Only the knob moved during it; the
// volume is set now that the level is known, and the tick played at it.
func (a *app) endVolumeDrag() {
	if !a.dragVolume {
		return
	}
	a.dragVolume = false
	a.applyVolume()
	a.sounds.Tick()
}

// closeVolume hides the slider and saves the level it was left at.
func (a *app) closeVolume() {
	a.endVolumeDrag()
	a.view.CloseVolume()
	a.saveSettings()
}

// volumePress handles a mouse press while the slider is open: on the track
// it moves the knob there and starts a drag, which sets the volume when the
// mouse is released; anywhere off the panel it closes the slider without
// reaching the keys underneath.
func (a *app) volumePress(pt image.Point) {
	switch {
	case pt.In(ui.VolumeTrack):
		a.dragVolume = true
		a.moveVolume(ui.VolumeAt(pt.Y))
	case !pt.In(ui.VolumeRect):
		a.closeVolume()
	}
}

// volumeKey handles the keyboard while the slider is open: Up and Right
// are louder, Down and Left quieter, Home the loudest, End mute, a digit
// that level, and Return, Space, Escape or S close it.
func (a *app) volumeKey(key qtcore.Key, text string) {
	switch key {
	case qtcore.Key_Up, qtcore.Key_Right:
		a.setVolume(a.view.Volume + 1)
	case qtcore.Key_Down, qtcore.Key_Left:
		a.setVolume(a.view.Volume - 1)
	case qtcore.Key_Home, qtcore.Key_PageUp:
		a.setVolume(ui.MaxVolume)
	case qtcore.Key_End, qtcore.Key_PageDown:
		a.setVolume(0)
	case qtcore.Key_Return, qtcore.Key_Enter, qtcore.Key_Space, qtcore.Key_Escape, qtcore.Key_S:
		a.closeVolume()
	default:
		if len(text) == 1 && text[0] >= '0' && text[0] <= '9' {
			a.setVolume(int(text[0] - '0'))
		}
	}
	a.w.Update()
}
