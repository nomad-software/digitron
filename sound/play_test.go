// Under the race detector a program that has played sound through Qt
// Multimedia crashes as it exits (in runtime.racefini, with Qt's audio
// threads still running; miqt did the same), so this test is left out of
// race builds, and make test runs it by itself without -race.

//go:build !race

package sound

import (
	"testing"

	"github.com/nomad-software/qute/qutetest"
)

// TestMain starts Qt, on its offscreen platform, for TestPlay.
func TestMain(m *testing.M) { qutetest.TestMain(m) }

// TestPlay makes the sounds through Qt Multimedia, plays every voice of each
// more than once round, changes the loudness, mutes them and closes them: a
// mistake in what the bindings own panics here. Where there is no audio
// device nothing is heard, but the same calls are made.
func TestPlay(t *testing.T) {
	qutetest.Run(t, func() {
		p, err := New()
		if err != nil {
			t.Fatal(err)
		}
		for range 2 * voices {
			p.Key()
			p.Tick()
		}
		p.SetLoudness(0.5)
		p.Key()
		p.SetLoudness(0)
		p.Key()
		p.Tick()
		p.Close()
		p.Close() // twice is harmless
	})
	var p *Player // nil plays nothing
	p.Key()
	p.Tick()
	p.SetLoudness(1)
	p.Close()
}
