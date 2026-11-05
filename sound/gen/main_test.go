package main

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestTickSamples(t *testing.T) {
	s := tickSamples()
	if want := int(math.Floor(sampleRate * tickLength)); len(s) < want {
		t.Fatalf("tick is %d samples, want at least %d", len(s), want)
	}
	if s[0] != 0 {
		t.Errorf("tick starts at %d, want 0 so it doesn't pop", s[0])
	}
}

func TestWAV(t *testing.T) {
	b := wav([]int16{1, -1, 300})
	le := binary.LittleEndian
	if string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" || string(b[36:40]) != "data" {
		t.Fatal("bad WAV header")
	}
	if len(b) != 44+6 || le.Uint32(b[4:]) != uint32(len(b)-8) || le.Uint32(b[40:]) != 6 {
		t.Errorf("bad WAV lengths")
	}
	if le.Uint32(b[24:]) != sampleRate || le.Uint16(b[34:]) != 16 {
		t.Errorf("bad WAV format")
	}
	if int16(le.Uint16(b[46:])) != -1 {
		t.Errorf("samples not written little-endian")
	}
}
