// Package sound plays the sounds of the interface: the chirp of a key
// being pressed and the tick of moving through a menu. The sounds are
// embedded in the program and played from memory with Qt Multimedia, so no
// file is read or written. If there is no audio device they are silently
// skipped.
//
// key.wav is the opening chirp of a recording, chatter3.ogg, cut from 20 ms
// to 140 ms with a 2 ms fade in and a 10 ms fade out, and raised to peak at
// -1 dB:
//
//	ffmpeg -i chatter3.ogg -af "atrim=start=0.020:end=0.140,asetpts=PTS-STARTPTS,afade=t=in:d=0.002,afade=t=out:st=0.110:d=0.010,volume=16.4dB" -ac 1 -ar 32000 -c:a pcm_s16le -map_metadata -1 -fflags +bitexact -flags:a +bitexact key.wav
//
// tick.wav is synthesised by the program in gen.
package sound

//go:generate go run ./gen

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

var (
	//go:embed key.wav
	keyWAV []byte
	//go:embed tick.wav
	tickWAV []byte
)

const (
	keyVolume  = 0.18 // Volume of a key's chirp from 0 to 1.
	tickVolume = 0.12 // Volume of a menu's tick from 0 to 1.

	// Each sound is made at a few pitches around its own, up to its spread
	// above and below it, and one is picked at random each time, so that
	// typing doesn't sound mechanical. The key varies only a little, so it
	// stays recognisably the same chirp.
	variants   = 7
	keySpread  = 0.01
	tickSpread = 0.08

	// voices is how many of each sound can play at once, so that quick
	// presses don't cut each other off.
	voices = 3
)

// voice is one audio output and the buffer it plays from.
type voice struct {
	sink *multimedia.QAudioSink
	buf  *qt.QBuffer
}

// effect is one sound at each of its pitches, with audio outputs of its
// own, set to its sample rate.
type effect struct {
	pitches [][]byte // 16-bit little-endian PCM
	voices  []voice
	next    int // the voice to play it on next
	last    int // the pitch played last
}

// Player plays the sounds. A nil Player plays nothing.
type Player struct {
	key, tick *effect
}

// New readies the sounds to be played. It needs the Qt application to have
// been made.
func New() (*Player, error) {
	key, err := newEffect(keyWAV, keyVolume, keySpread)
	if err != nil {
		return nil, fmt.Errorf("key.wav: %w", err)
	}
	tick, err := newEffect(tickWAV, tickVolume, tickSpread)
	if err != nil {
		key.close()
		return nil, fmt.Errorf("tick.wav: %w", err)
	}
	return &Player{key: key, tick: tick}, nil
}

func newEffect(wav []byte, volume, spread float64) (*effect, error) {
	pcm, rate, err := parseWAV(wav)
	if err != nil {
		return nil, err
	}
	e := &effect{last: -1}
	for i := range variants {
		pitch := 1 - spread + 2*spread*float64(i)/float64(variants-1)
		e.pitches = append(e.pitches, pcmBytes(resample(pcm, pitch)))
	}

	format := multimedia.NewQAudioFormat()
	defer format.Delete()
	format.SetSampleRate(rate)
	format.SetChannelCount(1)
	format.SetSampleFormat(multimedia.QAudioFormat__Int16)
	for range voices {
		v := voice{sink: multimedia.NewQAudioSink3(format), buf: qt.NewQBuffer()}
		v.sink.SetVolume(volume)
		e.voices = append(e.voices, v)
	}
	return e, nil
}

// Key plays the chirp of a key being pressed.
func (p *Player) Key() {
	if p != nil {
		p.key.play()
	}
}

// Tick plays the tick of moving through a menu.
func (p *Player) Tick() {
	if p != nil {
		p.tick.play()
	}
}

// Close stops and releases the audio outputs.
func (p *Player) Close() {
	if p == nil {
		return
	}
	p.key.close()
	p.tick.close()
}

// play plays the sound at a pitch picked at random, but never the same one
// twice running, on its next voice.
func (e *effect) play() {
	if len(e.voices) == 0 {
		return
	}
	i := rand.Intn(len(e.pitches))
	if i == e.last && len(e.pitches) > 1 {
		i = (i + 1 + rand.Intn(len(e.pitches)-1)) % len(e.pitches)
	}
	e.last = i

	v := e.voices[e.next]
	e.next = (e.next + 1) % len(e.voices)
	v.sink.Stop()
	v.buf.Close()
	v.buf.SetData(e.pitches[i])
	if v.buf.Open(qt.QIODeviceBase__ReadOnly) {
		v.sink.Start(v.buf.QIODevice)
	}
}

func (e *effect) close() {
	for _, v := range e.voices {
		v.sink.Stop()
		v.sink.Delete()
		v.buf.Delete()
	}
	e.voices = nil
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
