package calc

import (
	"math/big"
	"math/rand"
	"slices"
	"strings"
	"testing"
)

// The tests in this file press keys at random, and check what must always
// hold rather than particular answers. go test runs FuzzKeys on its seeds
// only; make fuzz searches further.

// maxFuzzKeys is how many keys a fuzzed sequence presses at most. MaxBits
// keeps every number finite, but checking one at the limit takes about a
// third of a second, mostly copying and pasting back its 315,000 digits,
// and a long sequence could spend minutes there.
const maxFuzzKeys = 64

// fuzzKey presses the key that b stands for: 0 to 15 are digits, and the
// rest every other key, the base and word size, a bit and a paste. Some
// keys need an argument, which is next, the byte after b; it is pressed as
// a key itself after.
func fuzzKey(e *Engine, b, next byte) {
	switch k := int(b) % 50; k {
	case 16:
		e.Point()
	case 17, 18, 19, 20, 21, 22, 23, 24, 25:
		e.Operator([]Op{OpAdd, OpSub, OpMul, OpDiv, OpAnd, OpOr, OpXor, OpShl, OpShr}[k-17])
	case 26:
		e.LParen()
	case 27:
		e.RParen()
	case 28, 29, 30: // = is pressed often, so that results are worked out
		e.Equals()
	case 31:
		e.Percent()
	case 32:
		e.Not()
	case 33:
		e.Negate()
	case 34:
		e.Sqrt()
	case 35:
		e.Backspace()
	case 36:
		e.ClearEntry()
	case 37:
		e.AllClear()
	case 38:
		e.MemClear()
	case 39:
		e.MemRecall()
	case 40:
		e.MemAdd()
	case 41:
		e.MemSub()
	case 42:
		e.SetBase(8)
	case 43:
		e.SetBase(10)
	case 44:
		e.SetBase(16)
	case 45:
		e.SetWidth(Widths[int(next)%len(Widths)])
	case 46:
		e.ToggleBit(int(next) % e.Width)
	case 47:
		e.Paste(e.CopyText())
	case 48:
		e.Paste(string(rune(next)))
	case 49:
		e.Paste("-1/3")
	default:
		e.Digit(k)
	}
}

// pressKeys presses the keys that keys stand for, at most maxFuzzKeys of
// them, and checks the engine after each.
func pressKeys(t *testing.T, keys []byte) {
	t.Helper()
	keys = keys[:min(len(keys), maxFuzzKeys)]
	e := New()
	for i, b := range keys {
		next := byte(0)
		if i+1 < len(keys) {
			next = keys[i+1]
		}
		fuzzKey(e, b, next)
		checkInvariants(t, e, keys[:i+1])
	}
}

// checkInvariants fails if the engine is in a state it must never be in.
func checkInvariants(t *testing.T, e *Engine, keys []byte) {
	t.Helper()
	fail := func(format string, args ...any) {
		t.Helper()
		t.Fatalf("after keys %v: "+format, append([]any{keys}, args...)...)
	}
	if e.Base != 8 && e.Base != 10 && e.Base != 16 {
		fail("base %d", e.Base)
	}
	if !slices.Contains(Widths, e.Width) {
		fail("word size %d", e.Width)
	}
	// Everything the UI asks for can be asked for.
	shown := e.Display().String()
	_ = e.Trail()
	text := e.CopyText()
	if shown == "" {
		fail("an empty display")
	}
	if e.Err() != nil {
		return
	}
	v := e.Value()
	if e.Base != 10 {
		// OCT and HEX hold whole numbers in the word, always.
		if !v.IsInt() {
			fail("a fraction, %s, in base %d", v.R.RatString(), e.Base)
		}
		if !e.FitsWord() {
			fail("%s does not fit %d bits in base %d", v.R.RatString(), e.Width, e.Base)
		}
	}
	// Nothing is ever shown as -0, typed or worked out: zero has no sign.
	if strings.HasPrefix(shown, "-") && v.R.Sign() >= 0 {
		fail("%s shown as %q", v.R.RatString(), shown)
	}
	// The binary display can always be drawn.
	switch {
	case e.IsFloat():
		if n := len(FloatBitsOf(v, e.Width)); n != e.Width {
			fail("%d float bits at %d", n, e.Width)
		}
	case e.FitsWord():
		if n := len(BitsOf(v, e.Width)); n != e.Width {
			fail("%d bits at %d", n, e.Width)
		}
	}
	// What is copied pastes back as the same value, unless it was an
	// approximation, which is copied as it is shown.
	if !v.Approx && !e.typing {
		p := New()
		p.Base, p.Width = e.Base, e.Width
		if !p.Paste(text) {
			fail("the copy %q does not paste", text)
		}
		if p.Value().R.Cmp(v.R) != 0 {
			fail("the copy %q pastes as %s, not %s", text, p.Value().R.RatString(), v.R.RatString())
		}
	}
}

// FuzzKeys presses keys at random and checks, after each one, that nothing
// panics and the engine stays in a state it can be in.
func FuzzKeys(f *testing.F) {
	for _, seed := range [][]byte{
		{},
		{1, 17, 2, 28},             // 1 + 2 =
		{1, 20, 0, 28, 17, 5},      // 1 ÷ 0 = + 5: after an error
		{7, 20, 3, 28, 44, 28},     // 7 ÷ 3 =, to HEX, =
		{26, 26, 26, 1, 17, 2},     // ((( 1 + 2, left open
		{27, 27, 28},               // ) ) =, with nothing open
		{1, 24, 6, 5, 5, 3, 6, 28}, // 1 << 65536
		{1, 24, 6, 5, 5, 3, 6, 28, 40, 39, 19, 39, 28, 38, 40, 39, 19, 39, 28, 38, 40, 39, 19, 39, 28, 38, 40, 39, 19, 39, 28, 38, 40, 39, 19, 39, 28}, // then squared through memory, past MaxBits
		{0, 33, 5, 16, 33, 35, 35},  // 0 ±, 5 . ±, backspaces: no −0
		{45, 0, 46, 7, 33, 28},      // 8 bits, 0, flip bit 7, 7, ±, =
		{49, 47, 44, 39},            // paste −1/3, copy and paste, to HEX, MR
		{1, 34, 34, 34, 44, 19, 28}, // √√√1, to HEX, × =
		// The sequences that found fractions hidden in HEX: from %, and
		// from an operand waiting when the base changed.
		{211, 70, 102, 104, 244, 70, 145, 23, 141, 151, 231},
		{151, 246, 50, 168, 90, 252, 97, 68, 214, 81, 191, 69, 212, 62, 2, 136, 194, 148, 45, 130},
	} {
		f.Add(seed)
	}
	f.Fuzz(pressKeys)
}

// TestKeysAtRandom presses a few thousand random sequences of keys, so
// that go test covers more than FuzzKeys's seeds.
func TestKeysAtRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	for range 3000 {
		keys := make([]byte, 1+rng.Intn(maxFuzzKeys/2))
		rng.Read(keys)
		pressKeys(t, keys)
	}
}

// TestAgainstBigRat checks DEC's arithmetic, typed in key by key, against
// math/big on random decimals: a op b, and a op b op c with C's
// precedence, which puts × and ÷ above + and −.
func TestAgainstBigRat(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	// decimal returns a random decimal of up to 12 digits either side of
	// the point, as typed and as an exact number.
	decimal := func() (string, *big.Rat) {
		var sb strings.Builder
		for range 1 + rng.Intn(12) {
			sb.WriteByte(byte('0' + rng.Intn(10)))
		}
		if rng.Intn(2) == 0 {
			sb.WriteByte('.')
			for range 1 + rng.Intn(12) {
				sb.WriteByte(byte('0' + rng.Intn(10)))
			}
		}
		s := sb.String()
		r, _ := new(big.Rat).SetString(s)
		if rng.Intn(3) == 0 {
			return s + "n", r.Neg(r)
		}
		return s, r
	}
	type op struct {
		key  string
		do   func(a, b *big.Rat) *big.Rat
		tier int // 1 for + −, 2 for × ÷
	}
	ops := []op{
		{"+", func(a, b *big.Rat) *big.Rat { return new(big.Rat).Add(a, b) }, 1},
		{"-", func(a, b *big.Rat) *big.Rat { return new(big.Rat).Sub(a, b) }, 1},
		{"*", func(a, b *big.Rat) *big.Rat { return new(big.Rat).Mul(a, b) }, 2},
		{"/", func(a, b *big.Rat) *big.Rat { return new(big.Rat).Quo(a, b) }, 2},
	}
	check := func(keys string, want *big.Rat) {
		t.Helper()
		e := New()
		press(e, keys)
		if e.Err() != nil {
			t.Fatalf("%s gave error %v", keys, e.Err())
		}
		if e.Value().R.Cmp(want) != 0 || e.Approx() {
			t.Fatalf("%s = %s (approx %v), want exactly %s", keys, e.Value().R.RatString(), e.Approx(), want.RatString())
		}
	}
	for range 2000 {
		as, a := decimal()
		bs, b := decimal()
		cs, c := decimal()
		o1, o2 := ops[rng.Intn(len(ops))], ops[rng.Intn(len(ops))]
		if (o1.key == "/" && b.Sign() == 0) || (o2.key == "/" && c.Sign() == 0) {
			continue
		}
		check(as+o1.key+bs+"=", o1.do(a, b))
		var want *big.Rat
		if o2.tier > o1.tier {
			want = o1.do(a, o2.do(b, c))
		} else {
			want = o2.do(o1.do(a, b), c)
		}
		check(as+o1.key+bs+o2.key+cs+"=", want)
	}
}
