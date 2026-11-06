package calc

// The tests in this file check typing a number: digits, the point, +/-,
// backspace, and what the status line shows of the expression.

import (
	"math/big"
	"testing"
)

func TestEntry(t *testing.T) {
	for _, c := range []struct{ keys, want string }{
		{"007", "7"},        // leading zeros are dropped
		{"1+.5=", "1.5"},    // a point starts a new number at 0.
		{"5b", "0"},         // backspace to nothing leaves 0
		{"5nb", "0"},        // and so does backspacing a lone sign
		{"12b3", "13"},      // backspace mid number
		{"5=b", "5"},        // backspace does nothing to a result
		{"5nn", "5"},        // +/- twice while typing
		{"(2)3=", "6"},      // a number after ) multiplies
		{"(2)(3)=", "6"},    // and so does (
		{"2(3)=", "6"},      // and a number before (
		{"1.2.3", "1.23"},   // a second point is ignored
		{"2=(3)=", "3"},     // ( after = starts afresh
		{"5+*3=", "15"},     // a second operator replaces the first
		{"(+2)=", "2"},      // an operator straight after ( is ignored
		{"+3=", "3"},        // an operator first works on the 0 shown
		{"(2+)3)=", "5"},    // ) straight after an operator is ignored
		{"()=", "0"},        // and straight after (
		{"2+3==", "8"},      // = again does + 3 again
		{"=", "0"},          // = on nothing
		{"(2+3", "3"},       // an open group shows what is typed
		{"(2+3=", "5"},      // = closes open groups
		{"((1+2)*3", "3"},   // nesting
		{"((1+2)*3)=", "9"}, // closed
	} {
		run(t, 10, 64, c.keys, c.want)
	}
	run(t, 16, 64, "1.2", "12") // no point outside DEC
}

func TestNegateTyping(t *testing.T) {
	// DEC has no word size, so +/- always works, even where the result
	// would not fit it: -128 to 128 in 8 bits.
	e := New()
	e.SetWidth(8)
	press(e, "12n8n")
	if got := e.Display().String(); got != "128" || e.Value().R.Cmp(big.NewRat(128, 1)) != 0 {
		t.Fatalf("+/- on -128 in 8 bits shows %q, value %v; want 128", got, e.Value().R)
	}
}

// TestNegateZero checks that 0 has no sign, as in the Windows calculator:
// +/- on a 0 being typed does nothing, so a negative number is typed as
// its digits and then +/-, and backspacing a negative number down to 0
// drops the sign. Nothing ever shows -0.
func TestNegateZero(t *testing.T) {
	for _, c := range []struct{ keys, want string }{
		{"n", "0"},
		{"0n", "0"},
		{"0n5", "5"},  // the sign is not kept for the digit after
		{"0.n", "0."}, // nor for a point
		{"0.0n", "0.0"},
		{"0.5n", "-0.5"}, // but a number below 1 has one
		{"5n", "-5"},
		{"5nb", "0"},
		{"0.05nb", "0.0"}, // −0.05 backspaced to −0.0 is 0.0
		{"0.05nbb", "0."}, // and on to 0.
		{"5n0", "-50"},
		{"0=n", "0"},
		{"1-1=n", "0"},
	} {
		run(t, 10, 64, c.keys, c.want)
	}
}

func TestTrailAndParens(t *testing.T) {
	e := New()
	press(e, "((")
	if e.OpenParens() != 2 {
		t.Fatalf("OpenParens = %d, want 2", e.OpenParens())
	}
	press(e, "1)")
	if e.OpenParens() != 1 {
		t.Fatalf("OpenParens = %d, want 1", e.OpenParens())
	}

	e = New()
	press(e, "1+2")
	if got := e.Trail(); got != "1 +" {
		t.Errorf("Trail while typing = %q, want %q", got, "1 +")
	}
	press(e, "*(3")
	if got := e.Trail(); got != "1 + 2 × (" {
		t.Errorf("Trail in a group = %q", got)
	}
	press(e, "=")
	if got := e.Trail(); got != "1 + 2 × (3) =" {
		t.Errorf("Trail after = is %q", got)
	}
}
