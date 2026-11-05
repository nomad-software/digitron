// Command gen makes tick.wav, the click of a key, which the sound package
// embeds. The click is generated rather than recorded: a soft low thump
// with a muffled knock and only a hint of a click, the same tick as the
// typewriter of the "The Thing" screensaver. Run it with go generate in the
// sound package after changing it.
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"os"
)

const (
	sampleRate = 22050 // Samples per second of the generated sound.
	tickLength = 0.04  // Length of the tick in seconds.
)

func main() {
	out := "tick.wav"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	if err := os.WriteFile(out, wav(tickSamples()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

// tickSamples generates the tick. It's mostly a soft low thump with a
// muffled burst of noise and only a hint of a click, so it sounds dull.
// The noise is seeded, so the tick is the same every time it is made.
func tickSamples() []int16 {
	noise := rand.New(rand.NewSource(1))
	samples := make([]float64, int(math.Round(sampleRate*tickLength+0.5)))

	low, lower := 0.0, 0.0
	peak := 0.0
	for i := range samples {
		t := float64(i) / sampleRate

		thump := math.Sin(2*math.Pi*220*t) * math.Exp(-t/0.007) * 0.4
		click := math.Sin(2*math.Pi*1000*t) * math.Exp(-t/0.002) * 0.2

		// Low pass the noise twice so it's a muffled knock rather than a
		// hiss.
		low += (noise.Float64()*2 - 1 - low) * 0.3
		lower += (low - lower) * 0.3
		knock := lower * math.Exp(-t/0.0025) * 2.5

		// Ease in over half a millisecond to avoid a pop.
		attack := math.Min(1, t/0.0005)

		samples[i] = (thump + click + knock) * attack
		peak = math.Max(peak, math.Abs(samples[i]))
	}

	out := make([]int16, len(samples))
	for i, s := range samples {
		out[i] = int16(s / peak * math.MaxInt16 * 0.9)
	}
	return out
}

// wav returns the samples as a mono 16-bit PCM WAV file.
func wav(samples []int16) []byte {
	const channels, bits = 1, 16
	dataLen := len(samples) * 2
	b := make([]byte, 44+dataLen)
	le := binary.LittleEndian
	copy(b[0:], "RIFF")
	le.PutUint32(b[4:], uint32(36+dataLen))
	copy(b[8:], "WAVE")
	copy(b[12:], "fmt ")
	le.PutUint32(b[16:], 16) // size of the fmt chunk
	le.PutUint16(b[20:], 1)  // PCM
	le.PutUint16(b[22:], channels)
	le.PutUint32(b[24:], sampleRate)
	le.PutUint32(b[28:], sampleRate*channels*bits/8) // bytes a second
	le.PutUint16(b[32:], channels*bits/8)            // bytes a frame
	le.PutUint16(b[34:], bits)
	copy(b[36:], "data")
	le.PutUint32(b[40:], uint32(dataLen))
	for i, s := range samples {
		le.PutUint16(b[44+i*2:], uint16(s))
	}
	return b
}
