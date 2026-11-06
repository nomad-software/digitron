package calc

// The tests in this file pin down answers that famous calculators and
// floating point got wrong, and problems made to break floating point.

import (
	"math/big"
	"strings"
	"testing"
)

// TestFamousBugs pins down answers that well-known calculators got wrong,
// so that they can never come back here. Digitron works in exact fractions
// and takes exact square roots of perfect squares, so every one of these
// must be exactly right, with no ≈.
func TestFamousBugs(t *testing.T) {
	for _, c := range []struct{ name, keys, want string }{
		// Windows calculator (Windows 7 to 10): √4 − 2 gave
		// -8.1648465955514287168521180122928e-39, because √4 was kept as
		// an approximation.
		{"windows sqrt(4)-2", "4r-2=", "0"},
		{"sqrt(9)-3", "9r-3=", "0"},
		{"sqrt(0.25)", "0.25r", "0.5"},
		{"sqrt(2.25)-1.5", "2.25r-1.5=", "0"},
		{"sqrt(4)*sqrt(4)", "4r*4r=", "4"},
		// Windows 3.1 calculator: 2.01 − 2 gave 0.00, and so did
		// 12.52 − 12.51.
		{"windows 3.1 2.01-2", "2.01-2=", "0.01"},
		{"windows 3.1 12.52-12.51", "12.52-12.51=", "0.01"},
		// The Pentium FDIV bug: 4195835 ÷ 3145727 was wrong in the fourth
		// significant digit, so this did not come back to 4195835.
		{"pentium fdiv", "4195835/3145727*3145727=", "4195835"},
		{"pentium fdiv check", "4195835-4195835/3145727*3145727=", "0"},
		// Windows 3.1 again: 3.11 − 3.1 gave 0.00.
		{"windows 3.1 3.11-3.1", "3.11-3.1=", "0.01"},
		// Excel 2007 showed 100000 for any formula whose result was the
		// double nearest 65535, such as 850 × 77.1, or a few below it.
		{"excel 2007 850*77.1", "850*77.1=", "65535"},
		{"excel 2007 5.1*12850", "5.1*12850=", "65535"},
		{"excel 2007 10.2*6425", "10.2*6425=", "65535"},
		// Excel: 1 × (0.5 − 0.4 − 0.1) gives -2.77555756156289E-17.
		{"excel 0.5-0.4-0.1", "0.5-0.4-0.1=", "0"},
		// The Patriot missile at Dhahran (1991) counted time in tenths of a
		// second with 0.1 cut to 24 bits; after 100 hours its clock was a
		// third of a second out. 3,600,000 tenths are exactly 360,000 s.
		{"patriot 100 hours", "0.1*3600000=", "360000"},
		// Above 2^53 a double cannot hold every whole number, so
		// JavaScript's 9007199254740992 + 1 is 9007199254740992.
		{"2^53+1", "9007199254740992+1=", "9007199254740993"},
		// Absorption: 1 vanishes into 10^16 in a double, so this is 0.
		{"1e16+1-1e16", "10000000000000000+1-10000000000000000=", "1"},
		// Catastrophic cancellation: 1 + 10^-20 is 1 in a double.
		{"(1+1e-20)-1", "1.00000000000000000001-1=", "0.00000000000000000001"},
		// Shop arithmetic that floats get wrong: 4.35 × 100 is 434.99999…,
		// 1.015 × 100 is 101.49999…, 0.7 + 0.1 is 0.79999….
		{"4.35*100", "4.35*100=", "435"},
		{"1.015*100", "1.015*100=", "101.5"},
		{"0.7+0.1", "0.7+0.1=", "0.8"},
		{"1.1+2.2", "1.1+2.2=", "3.3"},
		// The usual floating-point traps.
		{"0.1+0.2", "0.1+0.2=", "0.3"},
		{"1-0.9", "1-0.9=", "0.1"},
		{"3*0.1", "3*0.1=", "0.3"},
		{"1.1*1.1", "1.1*1.1=", "1.21"},
		{"thirds", "1/3+1/3+1/3=", "1"},
		{"division round trip", "10/3*3=", "10"},
	} {
		e := New()
		press(e, c.keys)
		if e.Err() != nil {
			t.Errorf("%s: %q gave error %v", c.name, c.keys, e.Err())
			continue
		}
		if got := e.Display().String(); got != c.want {
			t.Errorf("%s: %q = %q, want %q", c.name, c.keys, got, c.want)
		}
		if e.Approx() {
			t.Errorf("%s: %q is marked approximate, want exact", c.name, c.keys)
		}
	}
}

// TestRump checks Siegfried Rump's polynomial (1988), made to show that
// working at higher precision can agree with itself and still be wrong:
//
//	333.75·b⁶ + a²(11a²b² − b⁶ − 121b⁴ − 2) + 5.5·b⁸ + a/(2b)
//
// with a = 77617 and b = 33096. Single, double and extended precision all
// give about 1.1726, and quad precision another wrong answer; it is
// exactly −54767/66192, −0.8273960599….
func TestRump(t *testing.T) {
	pow := func(x string, n int) string { return x + strings.Repeat("*"+x, n-1) }
	a, b := "77617", "33096"
	keys := "333.75*" + pow(b, 6) +
		"+" + pow(a, 2) + "*(11*" + pow(a, 2) + "*" + pow(b, 2) + "-" + pow(b, 6) + "-121*" + pow(b, 4) + "-2)" +
		"+5.5*" + pow(b, 8) +
		"+" + a + "/(2*" + b + ")="
	e := New()
	press(e, keys)
	if e.Err() != nil {
		t.Fatalf("error %v", e.Err())
	}
	if want := big.NewRat(-54767, 66192); e.Value().R.Cmp(want) != 0 || e.Approx() {
		t.Errorf("Rump's polynomial = %s (approx %v), want exactly %s", e.Value().R.RatString(), e.Approx(), want.RatString())
	}
	if got := e.Display().String(); !strings.HasPrefix(got, "-0.82739605994682136814116") {
		t.Errorf("shown as %q", got)
	}
}

// TestMuller checks Jean-Michel Muller's recurrence
//
//	x(n+1) = 111 − 1130/x(n) + 3000/(x(n)·x(n−1)),  x0 = 11/2, x1 = 61/11
//
// which is exactly (6^(n+1) + 5^(n+1)) / (6^n + 5^n) and so tends to 6. In
// floating point the rounding errors feed a term that tends to 100, and by
// x(30) any float gives 100. Each step is keyed in, its operands pasted.
func TestMuller(t *testing.T) {
	exact := func(n int64) *big.Rat {
		p := func(b, e int64) *big.Int { return new(big.Int).Exp(big.NewInt(b), big.NewInt(e), nil) }
		return new(big.Rat).SetFrac(new(big.Int).Add(p(6, n+1), p(5, n+1)), new(big.Int).Add(p(6, n), p(5, n)))
	}
	prev, cur := big.NewRat(11, 2), big.NewRat(61, 11)
	for n := int64(2); n <= 30; n++ {
		e := New()
		paste := func(r *big.Rat) {
			if !e.Paste(r.RatString()) {
				t.Fatalf("x%d: %s was not pasted", n, r.RatString())
			}
		}
		press(e, "111-1130/")
		paste(cur)
		press(e, "+3000/(")
		paste(cur)
		press(e, "*")
		paste(prev)
		press(e, ")=")
		if e.Err() != nil {
			t.Fatalf("x%d: error %v", n, e.Err())
		}
		next := e.Value().R
		if want := exact(n); next.Cmp(want) != 0 {
			t.Fatalf("x%d = %s, want %s", n, next.RatString(), want.RatString())
		}
		prev, cur = cur, new(big.Rat).Set(next)
	}
	// It nears 6 from below, by about 5/6 of the way each step.
	if lo, hi := big.NewRat(599, 100), big.NewRat(6, 1); cur.Cmp(lo) <= 0 || cur.Cmp(hi) >= 0 {
		t.Errorf("x30 = %s, want it just below 6 (floats give 100)", cur.FloatString(6))
	}
}
