package calc

import (
	"math/big"
	"testing"
	"time"
)

// TestBeyondTheWord checks DEC, which has no word size, against OCT and HEX,
// which keep to it.
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
	// DEC shifts as far as MaxBits: 1 << 1048575 has exactly MaxBits
	// bits, and one more is too big.
	e := New()
	press(e, "1<1048575=")
	if e.Err() != nil || e.Value().R.Num().BitLen() != MaxBits {
		t.Errorf("1 << 1048575: %v, %d bits", e.Err(), e.Value().R.Num().BitLen())
	}
	run(t, 10, 64, "1<1048576=", "ERR:TOO BIG")
	run(t, 10, 64, "3<1048575=", "ERR:TOO BIG") // two bits shifted that far
	run(t, 10, 64, "0<1000000000=", "0")        // nothing shifted any way is 0
	run(t, 10, 64, "3>1000000=", "0")           // right past every bit
	run(t, 10, 64, "3n>1000000=", "-1")         // and a negative floors
	run(t, 16, 8, "1<1000=", "0")               // a word shifts out
}

// TestEverythingFitsTheWord checks that leaving DEC, or changing the word
// size in OCT and HEX, fits all the engine holds to the word, not only
// what is shown: operands waiting in the expression, memory, and what =
// repeats. They used to keep their fractions, or their size, and bring them
// back into a result. Found by TestKeysAtRandom.
func TestEverythingFitsTheWord(t *testing.T) {
	for _, c := range []struct {
		name   string
		before string          // keys pressed first, in DEC at 64 bits
		change func(e *Engine) // then this
		after  string          // then these
		want   string
	}{
		{"a waiting fraction", "0.5+0.5*", func(e *Engine) { e.SetBase(16) }, "2=", "0"},
		{"a fraction in memory", "0.5", func(e *Engine) { e.MemAdd(); e.SetBase(16); e.MemRecall() }, "", "0"},
		{"memory too big for the word", "300", func(e *Engine) { e.MemAdd(); e.SetWidth(8); e.SetBase(16); e.MemRecall() }, "", "2C"},
		{"a fraction = repeats", "1+1.5=", func(e *Engine) { e.SetBase(16) }, "=", "3"},
		{"a waiting operand too big for a smaller word", "", func(e *Engine) {
			e.SetBase(16)
			e.SetWidth(16)
			press(e, "7F00>")
			e.SetWidth(8)
		}, "8=", "0"},
	} {
		e := New()
		press(e, c.before)
		c.change(e)
		press(e, c.after)
		if got := e.Display().String(); got != c.want || !e.Value().IsInt() {
			t.Errorf("%s: shows %q, holds %s, want %q", c.name, got, e.Value().R.RatString(), c.want)
		}
	}
}

// pow2InMemory leaves 2^n in memory, for n a power of 2 from 65536, by
// squaring 2^65536 through memory, as someone at the keys could: MR × MR =,
// MC, M+.
func pow2InMemory(t *testing.T, e *Engine, n int) {
	t.Helper()
	press(e, "1<65536=")
	e.MemAdd()
	for p := 65536; p < n; p *= 2 {
		e.MemRecall()
		press(e, "*")
		e.MemRecall()
		press(e, "=")
		e.MemClear()
		e.MemAdd()
	}
	if e.Err() != nil || e.mem.R.Num().BitLen() != n+1 {
		t.Fatalf("2^%d in memory: error %v, %d bits", n, e.Err(), e.mem.R.Num().BitLen())
	}
}

// TestTooBig checks that DEC, which has no word size, still has a limit,
// MaxBits, so that squaring a number again and again cannot ask for
// billions of digits and stop the calculator. Every way to pass it is an
// error, and quick.
func TestTooBig(t *testing.T) {
	start := time.Now()

	// Squaring: 2^524288 is within the limit, 2^1048576 one bit beyond it.
	// (Shifting is in TestShiftsWithoutAWord.)
	e := New()
	pow2InMemory(t, e, 524288)
	e.MemRecall()
	press(e, "*")
	e.MemRecall()
	press(e, "=")
	if e.Err() != ErrTooBig {
		t.Errorf("2^1048576: error %v, want ErrTooBig", e.Err())
	}

	// Just within: 2^1048575 has exactly MaxBits bits.
	e = New()
	pow2InMemory(t, e, 524288)
	e.MemRecall()
	press(e, "/2*")
	e.MemRecall()
	press(e, "=")
	if e.Err() != nil || e.Value().R.Num().BitLen() != MaxBits {
		t.Fatalf("2^1048575: error %v, %d bits", e.Err(), e.Value().R.Num().BitLen())
	}

	// M+: memory doubled past the limit is refused and left as it was.
	e.MemClear()
	e.MemAdd()
	before := new(big.Rat).Set(e.mem.R)
	e.MemAdd()
	if e.Err() != ErrTooBig || e.mem.R.Cmp(before) != 0 {
		t.Errorf("M+ past the limit: error %v, memory changed %v", e.Err(), e.mem.R.Cmp(before) != 0)
	}

	// %: 4x + x% is 4x × x / 100, x² / 25, whose top is 2^1048576.
	e = New()
	pow2InMemory(t, e, 524288)
	e.MemRecall()
	press(e, "*4+")
	e.MemRecall()
	press(e, "%")
	if e.Err() != ErrTooBig {
		t.Errorf("4x + x%%: error %v, want ErrTooBig", e.Err())
	}

	// A fraction counts its bottom too: 1 / 2^1048576.
	e = New()
	pow2InMemory(t, e, 524288)
	press(e, "1/")
	e.MemRecall()
	press(e, "/")
	e.MemRecall()
	press(e, "=")
	if e.Err() != ErrTooBig {
		t.Errorf("1 / 2^1048576: error %v, want ErrTooBig", e.Err())
	}

	if d := time.Since(start); d > 10*time.Second {
		t.Errorf("reaching the limit took %v", d)
	}
}
