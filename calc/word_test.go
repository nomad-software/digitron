package calc

import "testing"

// TestBeyondTheWord checks DEC, which has no word size, against BIN, OCT and
// HEX, which keep to it.
func TestBeyondTheWord(t *testing.T) {
	e := New()
	e.SetWidth(8)
	press(e, "127")
	if !e.FitsWord() {
		t.Error("127 fits 8 bits")
	}
	press(e, "+1=")
	if e.FitsWord() {
		t.Error("128 does not fit 8 bits, signed")
	}
	e.ToggleBit(0) // there are no bits to click
	if got := e.Display().String(); got != "128" {
		t.Errorf("a bit was flipped beyond the word: %q", got)
	}
	e.SetWidth(16) // a wider word, and it fits again, still 128
	if !e.FitsWord() || e.Display().String() != "128" {
		t.Errorf("128 in 16 bits: fits %v, shows %q", e.FitsWord(), e.Display())
	}
	press(e, "n")
	if !e.FitsWord() {
		t.Error("-128 fits")
	}

	// Leaving DEC wraps the value into the word, as it drops a fraction.
	e = New()
	e.SetWidth(8)
	press(e, "300")
	e.SetBase(16)
	if got := e.Display().String(); got != "2C" || !e.FitsWord() {
		t.Errorf("300 in 8-bit HEX shows %q (fits %v), want 2C", got, e.FitsWord())
	}

	// Memory in DEC keeps a value of any size.
	e = New()
	e.SetWidth(8)
	press(e, "1000")
	e.MemAdd()
	e.MemRecall()
	if got := e.Display().String(); got != "1000" {
		t.Errorf("memory gave back %q, want 1000", got)
	}
}

func TestShiftsWithoutAWord(t *testing.T) {
	run(t, 10, 64, "1<65537=", "ERR:SHIFT TOO BIG")
	run(t, 10, 64, "3>1000000=", "0")   // right past every bit
	run(t, 10, 64, "3n>1000000=", "-1") // and a negative floors
	run(t, 16, 8, "1<1000=", "0")       // a word shifts out
	e := New()
	press(e, "1<65535=")
	if e.Err() != nil || e.Value().R.Num().BitLen() != 65536 {
		t.Errorf("1 << 65535: %v, %d bits", e.Err(), e.Value().R.Num().BitLen())
	}
}

func TestFractionPendingIntoHex(t *testing.T) {
	// A fraction typed in DEC and left in the expression is shown by its
	// whole part once in HEX, where the result is worked out.
	e := New()
	press(e, "1.5+")
	e.SetBase(16)
	press(e, "1=")
	if got := e.Display().String(); got != "2" {
		t.Errorf("1.5 + 1 in HEX shows %q, want 2", got)
	}
}
