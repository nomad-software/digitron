package calc

import (
	"math/big"
	"testing"
)

func TestRoundPlaces(t *testing.T) {
	for _, c := range []struct {
		in     string
		places int
		want   string
	}{
		{"1.25", 1, "1.3"},   // halves away from zero
		{"-1.25", 1, "-1.3"}, // on both sides
		{"1.24", 1, "1.2"},
		{"0.99999", 3, "1"},
		{"-0.00001", 3, "0"},
		{"2", 12, "2"},
	} {
		r, _ := new(big.Rat).SetString(c.in)
		if got := formatDecimal(roundPlaces(r, c.places)).String(); got != c.want {
			t.Errorf("roundPlaces(%s, %d) = %s, want %s", c.in, c.places, got, c.want)
		}
	}
}

func TestExactValuesNotRounded(t *testing.T) {
	// Only approximate values are rounded: an exact long decimal is shown
	// whole.
	run(t, 10, 64, "1/1024=", "0.0009765625")
	run(t, 10, 64, "1/3=", "0.3")
	run(t, 10, 64, "1/65536=", "0.0000152587890625")
}

// TestFitDisplay checks that every result fills the display: exactly where
// it fits, and otherwise rounded to as many places as the display holds,
// wherever the point is.
func TestFitDisplay(t *testing.T) {
	for _, c := range []struct {
		name, keys, want string
		rounded          bool
	}{
		{"fits exactly", "1/1024=", "0.0009765625", false},
		{"repeating fits", "1/7=", "0.142857", false},
		{"long period, small", "1/97=", "0.0103092783505154639175258", true},
		{"long period, large", "123456789/97=", "1272750.4020618556701030928", true},
		{"negative", "1n/97=", "-0.010309278350515463917526", true},
		{"big integer part", "9223372036854775807/7=", "1317624576693539401", false},
		{"big integer part, fraction", "9223372036854775806/97=", "95086309658296657.793814433", true},
		{"tiny", "1/97/1000000000=", "0.0000000000103092783505155", true},
		{"tiny, repeating", "1/3/1000000000/1000000000/1000=", "0.0000000000000000000003", false},
		// Too small for the digits after its zeros to show: in scientific
		// notation, rounded only if the mantissa is.
		{"too small", "1/97/1000000000/1000000000/1000000000=", "1.030927835051546391753E-29", true},
		{"too small, exact", "1/1000000000000000000000000000000=", "1E-30", false},
		{"too small, negative", "1n/3/1000000000000000000000000000000=", "-3.33333333333333333333E-31", true},
		{"six digits left: not yet", "1/97/1000000000000000000=", "0.0000000000000000000103093", true},
		{"five digits left: scientific", "1/7/100000000000000000000=", "1.428571428571428571429E-21", true},
		{"rounds up to the next power", "0.999999999999999999999999999999/1000000000000000000000000=", "1E-24", true},
		{"and keeps its sign", "0.999999999999999999999999999999n/1000000000000000000000000=", "-1E-24", true},
		{"carry", "0.99999999999999999999999999999+0=", "1", true},
		{"root of a big number", "9223372036854775807r", "3037000499.9760496922867524", true},
	} {
		e := New()
		press(e, c.keys)
		f := e.Display()
		if got := f.String(); got != c.want {
			t.Errorf("%s: %q = %q (%d cells), want %q", c.name, c.keys, got, len(f.Cells), c.want)
		}
		if len(f.Cells) > DisplayCells {
			t.Errorf("%s: %q is %d cells, more than %d", c.name, c.keys, len(f.Cells), DisplayCells)
		}
		if f.Rounded != c.rounded {
			t.Errorf("%s: %q rounded %v, want %v", c.name, c.keys, f.Rounded, c.rounded)
		}
	}
	// Copying an exact value gives it exactly, however it is shown.
	e := New()
	press(e, "1/97=")
	if got := e.CopyText(); got != "1/97" {
		t.Errorf("copy of 1/97 = %q", got)
	}
}

func TestHugeWholeNumberNotRounded(t *testing.T) {
	e := New()
	press(e, "1<100=")
	f := e.Display()
	if f.Rounded || f.String() != "1267650600228229401496703205376" {
		t.Errorf("2^100 shows %q, rounded %v; want it whole and exact", f, f.Rounded)
	}
}

func TestRepeatingOverline(t *testing.T) {
	e := New()
	press(e, "1/6=")
	f := e.Display()
	if f.String() != "0.16" || f.Cells[2].Over || !f.Cells[3].Over {
		t.Fatalf("1/6 cells: %+v", f.Cells)
	}
	press(e, "*6=")
	if e.Display().String() != "1" {
		t.Fatalf("1/6*6 = %s", e.Display())
	}
}
