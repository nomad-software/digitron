// Package sound plays the click of a key being pressed. The click,
// tick.wav, is embedded in the program and played from memory with Qt
// Multimedia, so no file is read or written. If there is no audio device it
// is silently skipped.
package sound

//go:generate go run ./gen tick.wav

import (
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/rand"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/multimedia"
)

//go:embed tick.wav
var tickWAV []byte

const (
	tickVolume = 0.18 // Volume of the tick from 0 to 1.

	// The tick is made at a few pitches around its own, from minPitch to
	// maxPitch, and one is picked at random each time, so that typing
	// doesn't sound mechanical.
	variants = 7
	minPitch = 0.92
	maxPitch = 1.08

	// voices is how many ticks can sound at once, so that quick presses
	// don't cut each other off.
	voices = 3
)

// voice is one audio output and the buffer it plays from.
type voice struct {
	sink *multimedia.QAudioSink
	buf  *qt.QBuffer
}

// Player plays the tick. A nil Player plays nothing.
type Player struct {
	ticks  [][]byte // 16-bit little-endian PCM, one for each pitch
	voices []voice
	next   int // the voice to play the next tick
	last   int // the pitch played last
}

// New readies the tick to be played. It needs the Qt application to have
// been made.
func New() (*Player, error) {
	pcm, rate, err := parseWAV(tickWAV)
	if err != nil {
		return nil, fmt.Errorf("tick.wav: %w", err)
	}
	p := &Player{last: -1}
	for i := range variants {
		pitch := minPitch + (maxPitch-minPitch)*float64(i)/float64(variants-1)
		p.ticks = append(p.ticks, pcmBytes(resample(pcm, pitch)))
	}

	format := multimedia.NewQAudioFormat()
	defer format.Delete()
	format.SetSampleRate(rate)
	format.SetChannelCount(1)
	format.SetSampleFormat(multimedia.QAudioFormat__Int16)
	for range voices {
		v := voice{sink: multimedia.NewQAudioSink3(format), buf: qt.NewQBuffer()}
		v.sink.SetVolume(tickVolume)
		p.voices = append(p.voices, v)
	}
	return p, nil
}

// Tick plays the tick of a key, at a pitch picked at random, but never the
// same one twice running.
func (p *Player) Tick() {
	if p == nil || len(p.voices) == 0 {
		return
	}
	i := rand.Intn(len(p.ticks))
	if i == p.last && len(p.ticks) > 1 {
		i = (i + 1 + rand.Intn(len(p.ticks)-1)) % len(p.ticks)
	}
	p.last = i

	v := p.voices[p.next]
	p.next = (p.next + 1) % len(p.voices)
	v.sink.Stop()
	v.buf.Close()
	v.buf.SetData(p.ticks[i])
	if v.buf.Open(qt.QIODeviceBase__ReadOnly) {
		v.sink.Start(v.buf.QIODevice)
	}
}

// Close stops and releases the audio outputs.
func (p *Player) Close() {
	if p == nil {
		return
	}
	for _, v := range p.voices {
		v.sink.Stop()
		v.sink.Delete()
		v.buf.Delete()
	}
	p.voices = nil
}

// parseWAV returns the samples and sample rate of a mono 16-bit PCM WAV
// file.
func parseWAV(b []byte) ([]int16, int, error) {
	le := binary.LittleEndian
	if len(b) < 12 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return nil, 0, errors.New("not a WAV file")
	}
	var rate int
	var gotFmt bool
	for off := 12; off+8 <= len(b); {
		id, size := string(b[off:off+4]), int(le.Uint32(b[off+4:]))
		body := b[off+8:]
		if size > len(body) {
			return nil, 0, errors.New("truncated chunk " + id)
		}
		body = body[:size]
		switch id {
		case "fmt ":
			if size < 16 || le.Uint16(body[0:]) != 1 || le.Uint16(body[2:]) != 1 || le.Uint16(body[14:]) != 16 {
				return nil, 0, errors.New("not mono 16-bit PCM")
			}
			rate, gotFmt = int(le.Uint32(body[4:])), true
		case "data":
			if !gotFmt {
				return nil, 0, errors.New("data before format")
			}
			s := make([]int16, size/2)
			for i := range s {
				s[i] = int16(le.Uint16(body[i*2:]))
			}
			return s, rate, nil
		}
		off += 8 + size + size&1 // chunks are padded to an even length
	}
	return nil, 0, errors.New("no data")
}

// resample plays the samples back faster (pitch above 1) or slower, which
// raises or lowers the pitch and shortens or lengthens the sound, as a
// change of pitch does on a tape. It interpolates between samples.
func resample(in []int16, pitch float64) []int16 {
	n := int(float64(len(in)) / pitch)
	out := make([]int16, n)
	for i := range out {
		pos := float64(i) * pitch
		j := int(pos)
		if j >= len(in)-1 {
			out[i] = in[len(in)-1]
			continue
		}
		f := pos - float64(j)
		out[i] = int16(math.Round(float64(in[j])*(1-f) + float64(in[j+1])*f))
	}
	return out
}

// pcmBytes returns samples as 16-bit little-endian PCM.
func pcmBytes(s []int16) []byte {
	b := make([]byte, len(s)*2)
	for i, v := range s {
		binary.LittleEndian.PutUint16(b[i*2:], uint16(v))
	}
	return b
}
