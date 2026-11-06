package calc

// The tests in this file check copying the exact value and pasting a
// number from text.

import (
	"strings"
	"testing"
)

func TestPaste(t *testing.T) {
	e := New()
	e.Paste("0xff")
	if e.Display().String() != "255" {
		t.Fatalf("got %s", e.Display())
	}
	e.Paste("1/3")
	press(e, "*3=")
	if e.Display().String() != "1" {
		t.Fatalf("got %s", e.Display())
	}
	if e.Paste("hello") {
		t.Fatal("bad paste accepted")
	}
	e.Paste("1,000,000")
	if e.CopyText() != "1000000" {
		t.Fatal(e.CopyText())
	}
	press(e, "/3=")
	if e.CopyText() != "1000000/3" {
		t.Fatal(e.CopyText())
	}
}

func TestCopyText(t *testing.T) {
	e := New()
	press(e, "12.")
	if got := e.CopyText(); got != "12." {
		t.Errorf("copy while typing = %q", got)
	}
	press(e, "5=")
	if got := e.CopyText(); got != "12.5" {
		t.Errorf("copy of 12.5 = %q", got)
	}
	// A decimal too long to show whole is copied as its fraction.
	e = New()
	press(e, "1/1019=")
	if !e.Display().Rounded {
		t.Error("1/1019 has a period of 1018 digits and should be rounded to fit")
	}
	if got := e.CopyText(); got != "1/1019" {
		t.Errorf("copy of 1/1019 = %q", got)
	}
	e = New()
	e.SetBase(16)
	press(e, "FF")
	e.Equals()
	if got := e.CopyText(); got != "FF" {
		t.Errorf("copy in HEX = %q", got)
	}
}

func TestPasteCases(t *testing.T) {
	for _, c := range []struct {
		base int
		in   string
		ok   bool
		want string
	}{
		{10, "0b101", true, "5"},
		{10, "0o17", true, "15"},
		{10, "-5", true, "-5"},
		{10, " 1_000 ", true, "1000"},
		{16, "ff", true, "FF"},
		{16, "0xzz", false, "0"},
		{16, "1.5", false, "0"},                                   // no fractions outside DEC
		{16, "0x1ffffffffffffffff", false, "0"},                   // 65 bits, more than the word
		{10, "0x1ffffffffffffffff", true, "36893488147419103231"}, // DEC has no word size
		{10, "", false, "0"},
		// Scientific notation, as an approximate tiny number is copied.
		{10, "1.5e-30", true, "1.5E-30"},
		{10, "1E3", true, "1000"},
		// Too big for MaxBits: refused before it is read.
		{10, "1e1000000000", false, "0"},
		{10, "1e-1000000000", false, "0"},
		{10, "1e300000", true, "1" + strings.Repeat("0", 300000)},
		{10, strings.Repeat("9", maxPaste+1), false, "0"},
	} {
		e := New()
		e.SetBase(c.base)
		ok := e.Paste(c.in)
		if ok != c.ok || e.Display().String() != c.want {
			t.Errorf("Paste(%q) in base %d = %v, %q; want %v, %q", c.in, c.base, ok, e.Display(), c.ok, c.want)
		}
	}
}
