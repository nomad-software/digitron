package calc

import (
	"math/big"
	"testing"
)

// TestRepeatEquals checks that = repeats the last operation, as on a desk
// calculator.
func TestRepeatEquals(t *testing.T) {
	for _, c := range []struct{ name, keys, want string }{
		{"add", "2+3===", "11"},
		{"subtract", "10-2===", "4"},
		{"multiply", "2*3==", "18"},
		{"divide", "81/3==", "9"},
		{"the last operation only", "1+2*3==", "21"},
		{"a group on the right", "(1+2)*(3+4)==", "147"},
		{"a new number", "2+3=7=", "10"},
		{"a new number, again", "2+3=7==", "13"},
		{"after a unary key", "2+3=n=", "-2"},
		{"after C", "2+3=C=", "0"},
		{"no operation", "5==", "5"},
		{"shift", "1<1===", "8"},
	} {
		e := New()
		for _, k := range c.keys {
			if k == 'C' {
				e.ClearEntry()
			} else {
				press(e, string(k))
			}
		}
		if got := e.Display().String(); got != c.want {
			t.Errorf("%s: %q = %q, want %q", c.name, c.keys, got, c.want)
		}
	}

	e := New()
	press(e, "2+3==")
	if got := e.Trail(); got != "5 + 3 =" {
		t.Errorf("trail of the repeat = %q", got)
	}
	// AC forgets the operation.
	e.AllClear()
	press(e, "4=")
	if got := e.Display().String(); got != "4" {
		t.Errorf("= after AC repeated: %q", got)
	}

	// M+ neither repeats the operation nor sets one up.
	e = New()
	press(e, "2+3")
	e.MemAdd()
	press(e, "=")
	if got := e.Display().String(); got != "5" {
		t.Errorf("= after M+ of 2+3 gave %q, want 5", got)
	}
	e = New()
	press(e, "2+3=")
	e.MemAdd()
	if got := e.Display().String(); got != "5" || e.mem.R.Cmp(big.NewRat(5, 1)) != 0 {
		t.Errorf("M+ after = shows %q with memory %v, want 5 and 5", got, e.mem.R)
	}
}

func TestRepeatFails(t *testing.T) {
	// A repeated operation can fail like any other: here, ÷ 0.
	e := New()
	press(e, "1+1=")
	e.repeatOp, e.repeatRHS = OpDiv, zero()
	press(e, "=")
	if e.Err() != ErrDivZero {
		t.Errorf("a repeated ÷ 0 gave %v", e.Err())
	}
}
