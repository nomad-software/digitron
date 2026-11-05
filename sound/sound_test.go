package sound

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestEmbeddedTick(t *testing.T) {
	s, rate, err := parseWAV(tickWAV)
	if err != nil {
		t.Fatal(err)
	}
	if rate != 22050 || len(s) < 800 {
		t.Errorf("tick.wav is %d samples at %d Hz", len(s), rate)
	}
}

// TestTickUpToDate checks that the embedded tick.wav is what the generator
// makes, so a change to it is not forgotten. Run go generate to remake it.
func TestTickUpToDate(t *testing.T) {
	out := filepath.Join(t.TempDir(), "tick.wav")
	cmd := exec.Command("go", "run", "./gen", out)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go run ./gen: %v\n%s", err, b)
	}
	want, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(tickWAV, want) {
		t.Error("tick.wav is out of date: run go generate ./sound")
	}
}

func TestParseWAVRejects(t *testing.T) {
	for name, b := range map[string][]byte{
		"empty":     nil,
		"not riff":  []byte("RIFX\x00\x00\x00\x00WAVE"),
		"no chunks": []byte("RIFF\x04\x00\x00\x00WAVE"),
	} {
		if _, _, err := parseWAV(b); err == nil {
			t.Errorf("%s: parsed", name)
		}
	}
}

func TestResample(t *testing.T) {
	in := []int16{0, 100, 200, 300, 400, 500, 600, 700}
	if out := resample(in, 1); len(out) != len(in) || out[3] != 300 {
		t.Errorf("resample at pitch 1 = %v", out)
	}
	if out := resample(in, 2); len(out) != 4 || out[1] != 200 {
		t.Errorf("resample at pitch 2 = %v", out)
	}
	if out := resample(in, 0.5); len(out) != 16 || out[1] != 50 {
		t.Errorf("resample at pitch 0.5 = %v", out)
	}
}

func TestPCMBytes(t *testing.T) {
	if b := pcmBytes([]int16{1, -1}); !bytes.Equal(b, []byte{1, 0, 0xff, 0xff}) {
		t.Errorf("pcmBytes = %v", b)
	}
}
