package calc

import (
	"fmt"
	"math/rand"
	"testing"
)

// The tests in this file check that OCT and HEX work as a programmer's
// fixed-width integers do: wrapping in two's complement, dividing towards
// zero, shifting arithmetically, with C's precedence.

// TestAgainstGoIntegers checks every operator in HEX, at every word size,
// against Go's own signed integers of that size, which wrap, divide and
// shift as C's do on a two's-complement machine. The operands are random,
// with the edges of each range (0, ±1, the least and greatest) mixed in.
func TestAgainstGoIntegers(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, width := range Widths {
		mask := uint64(1)<<width - 1
		if width == 64 {
			mask = ^uint64(0)
		}
		// sext reads the low width bits of u as a signed number.
		sext := func(u uint64) int64 { return int64(u<<(64-width)) >> (64 - width) }
		minV, maxV := sext(1<<(width-1)), sext(1<<(width-1)-1)
		edges := []int64{0, 1, -1, 2, -2, minV, maxV, minV + 1, maxV - 1}
		operand := func() int64 {
			if rng.Intn(3) == 0 {
				return edges[rng.Intn(len(edges))]
			}
			return sext(rng.Uint64())
		}
		hex := func(x int64) string { return fmt.Sprintf("%X", uint64(x)&mask) }

		for range 400 {
			a, b := operand(), operand()
			s := uint(rng.Intn(width + 2)) // shift counts past the width too
			// Go's operators at width bits: the sum and the rest are worked
			// out in int64 and cut to width, which is how they wrap.
			cases := []struct {
				op   string
				want int64
				skip bool
			}{
				{"+", a + b, false},
				{"-", a - b, false},
				{"*", a * b, false},
				{"/", 0, b == 0},
				{"&", a & b, false},
				{"|", a | b, false},
				{"^", a ^ b, false},
				{"<", 0, false},
				{">", a >> s, false},
			}
			if b != 0 {
				// Division of the least by -1 overflows; Go defines it as
				// the least, which is what wrapping gives.
				cases[3].want = sext(uint64(a/b) & mask)
				if a == minV && b == -1 {
					cases[3].want = minV
				}
			}
			if s < 64 {
				cases[7].want = a << s
			}
			for _, c := range cases {
				if c.skip {
					continue
				}
				rhs := hex(b)
				if c.op == "<" || c.op == ">" {
					rhs = fmt.Sprintf("%X", s)
				}
				keys := hex(a) + c.op + rhs + "="
				e := New()
				e.SetBase(16)
				e.SetWidth(width)
				press(e, keys)
				if e.Err() != nil {
					t.Fatalf("%d bits: %s gave error %v", width, keys, e.Err())
				}
				if got, want := e.Display().String(), hex(sext(uint64(c.want)&mask)); got != want {
					t.Fatalf("%d bits: %s = %s, want %s (%d %s %d)", width, keys, got, want, a, c.op, b)
				}
				if !e.FitsWord() {
					t.Fatalf("%d bits: %s does not fit the word", width, keys)
				}
			}
		}
	}
}

// TestLikeC pins down the cases C programmers trip over, and a few famous
// overflows.
func TestLikeC(t *testing.T) {
	for _, c := range []struct {
		name        string
		base, width int
		keys, want  string
	}{
		// INT_MIN / -1 overflows; on x86 it traps and kills the program.
		// Here it wraps, as everything else does.
		{"INT8_MIN / -1", 16, 8, "80/1n=", "80"},
		{"INT64_MIN / -1", 16, 64, "8000000000000000/1n=", "8000000000000000"},
		{"-INT8_MIN", 16, 8, "80n=", "80"},
		// Division truncates towards zero, as in C99, not down.
		{"-7 / 2", 16, 8, "7n/2=", "FD"},
		{"7 / -2", 16, 8, "7/2n=", "FD"},
		{"-7 / -2", 16, 8, "7n/2n=", "3"},
		{"7 / 2 in OCT", 8, 8, "7/2=", "3"},
		// >> keeps the sign; shifting a word out leaves its sign.
		{"0x80 >> 7", 16, 8, "80>7=", "FF"},
		{"0x80 >> 8", 16, 8, "80>8=", "FF"},
		{"0x40 >> 8", 16, 8, "40>8=", "0"},
		{"1 << 7", 16, 8, "1<7=", "80"},
		{"1 << 8", 16, 8, "1<8=", "0"},
		{"1 << 63", 16, 64, "1<3F=", "8000000000000000"},
		{"1 << 64", 16, 64, "1<40=", "0"},
		// A negative count shifts the other way.
		{"1 << -1", 16, 8, "1<1n=", "0"},
		{"1 >> -1", 16, 8, "1>1n=", "2"},
		// Precedence: shifts are below + and −, and & ^ | are below them,
		// in that order.
		{"2 + 3 << 1", 10, 64, "2+3<1=", "10"},
		{"1 << 2 + 1", 10, 64, "1<2+1=", "8"},
		{"1 | 2 & 3", 10, 64, "1|2&3=", "3"},
		{"1 | 2 ^ 3 & 4", 10, 64, "1|2^3&4=", "3"},
		{"6 ^ 3 | 8", 10, 64, "6^3|8=", "13"},
		{"8 & 4 + 4", 10, 64, "8&4+4=", "8"},
		{"1 + 2 * 3 << 1", 10, 64, "1+2*3<1=", "14"},
		// ~x is −x − 1, and bitwise operators see a negative number as
		// two's complement with as many sign bits as it needs.
		{"~5", 10, 64, "5~", "-6"},
		{"~-5", 10, 64, "5n~", "4"},
		{"-7 & 3", 10, 64, "7n&3=", "1"},
		{"-1 ^ 5", 10, 64, "1n^5=", "-6"},
		{"~0 in HEX", 16, 16, "0~", "FFFF"},
		// Unsigned-looking wrap.
		{"0xFF + 1", 16, 8, "FF+1=", "0"},
		{"0377 + 1", 8, 8, "377+1=", "0"},
		{"0xFFFF * 0xFFFF", 16, 16, "FFFF*FFFF=", "1"},
		// "Nuclear Gandhi": 1 − 2 in an unsigned byte is 255.
		{"1 - 2 in 8 bits", 16, 8, "1-2=", "FF"},
		// The Year 2038 problem and Gangnam Style's view counter: one past
		// the greatest 32-bit signed number is the least.
		{"INT32_MAX + 1", 16, 32, "7FFFFFFF+1=", "80000000"},
		// DEC has no word size, so it simply carries on.
		{"2147483647 + 1 in DEC", 10, 32, "2147483647+1=", "2147483648"},
		{"2^64 in DEC", 10, 64, "1<64=", "18446744073709551616"},
	} {
		e := New()
		e.SetBase(c.base)
		e.SetWidth(c.width)
		press(e, c.keys)
		got := e.Display().String()
		if e.Err() != nil {
			got = "ERR:" + e.Err().Error()
		}
		if got != c.want {
			t.Errorf("%s: base %d, %d bits, %q = %q, want %q", c.name, c.base, c.width, c.keys, got, c.want)
		}
	}
}

// TestAriane5 checks the conversion that destroyed Ariane 5's first flight
// (1996): a 64-bit floating-point horizontal velocity, too big for the
// 16-bit signed integer it was converted to. Here a value too big for the
// word wraps into it when leaving DEC, as a C cast does, rather than
// stopping the calculator.
func TestAriane5(t *testing.T) {
	e := New()
	e.SetWidth(16)
	press(e, "40000")
	e.SetBase(16)
	if got := e.Display().String(); got != "9C40" || !e.FitsWord() {
		t.Errorf("40000 in 16 bits = %q, fits %v, want 9C40", got, e.FitsWord())
	}
	if e.Value().R.Num().Int64() != 40000-65536 {
		t.Errorf("40000 in 16 bits is %s, want -25536", e.Value().R.RatString())
	}
}
