package calc

import (
	"math/big"
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

func TestSqrtIrrationalIsMarked(t *testing.T) {
	e := New()
	press(e, "2r")
	if !e.Approx() {
		t.Fatal("√2 should be marked approximate")
	}
	// It is shown, and copied, rounded to fill the display.
	if got := e.Display().String(); got != "1.4142135623730950488016887" {
		t.Errorf("√2 shows %q", got)
	}
	if got := e.CopyText(); got != "1.4142135623730950488016887" {
		t.Errorf("√2 copies as %q", got)
	}
	// Approximation carries through what is worked out from it. Kept to 64
	// places, √2 × √2 rounds to 2 as it is shown.
	press(e, "*2r=")
	if !e.Approx() {
		t.Fatal("√2 × √2 should be marked approximate")
	}
	if got := e.Display().String(); got != "2" {
		t.Errorf("√2 × √2 shows %q, want 2", got)
	}
	// In HEX a root is cut to a whole number and marked.
	e = New()
	e.SetBase(16)
	press(e, "2r")
	if got := e.Display().String(); got != "1" || !e.Approx() {
		t.Fatalf("√2 in HEX = %q (approx %v), want 1 marked approximate", got, e.Approx())
	}
}

func TestEntry(t *testing.T) {
	for _, c := range []struct{ keys, want string }{
		{"007", "7"},        // leading zeros are dropped
		{"0n5", "-5"},       // and after a sign
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

func TestUnaryOnGroup(t *testing.T) {
	for _, c := range []struct{ keys, want string }{
		{"(2+3)n=", "-5"},   // +/- takes the whole group
		{"((2))n=", "-2"},   // nested
		{"1+(2+3)n=", "-4"}, // and leaves what is before it
		{"(1+3)r=", "2"},    // √ of a group
		{"(2+3)~=", "-6"},   // NOT of a group
		{"1+(2)", "2"},      // a closed group shows its value
		{"(4)%", "0.04"},    // % of a group
	} {
		run(t, 10, 64, c.keys, c.want)
	}
	run(t, 10, 64, "0.5~", "ERR:NOT INTEGER")
	run(t, 10, 64, "0.5<1=", "ERR:NOT INTEGER")
	run(t, 10, 64, "1<0.5=", "ERR:NOT INTEGER")
	run(t, 10, 64, "(1/0)", "ERR:DIV BY ZERO")
	run(t, 10, 64, "2*(1/0=", "ERR:DIV BY ZERO")
}

func TestPercentInGroups(t *testing.T) {
	for _, c := range []struct{ keys, want string }{
		{"(200+10%)=", "220"},     // the left side stops at (
		{"(1+1)*100+10%=", "220"}, // and takes in a whole group
		{"5+(200-10%)=", "185"},   // inside a group after other terms
		{"10%", "0.1"},            // alone it divides by 100
	} {
		run(t, 10, 64, c.keys, c.want)
	}
	run(t, 10, 64, "1/0+10%", "ERR:DIV BY ZERO")
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

func TestClearEntryCases(t *testing.T) {
	e := New()
	press(e, "2+3=")
	e.ClearEntry() // after = it clears everything
	if e.Display().String() != "0" || e.Trail() != "" {
		t.Fatalf("C after = left %q, trail %q", e.Display(), e.Trail())
	}
	e = New()
	press(e, "1+(2)")
	e.ClearEntry() // takes away the closed group
	press(e, "3=")
	if got := e.Display().String(); got != "4" {
		t.Fatalf("1 + (2) C 3 = %q, want 4", got)
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

func TestModes(t *testing.T) {
	e := New()
	press(e, "12")
	e.SetWidth(8) // stops the typing, keeping 12
	press(e, "+3=")
	if got := e.Display().String(); got != "15" {
		t.Fatalf("got %q, want 15", got)
	}
	e.SetBase(10) // the same base changes nothing
	if got := e.Display().String(); got != "15" {
		t.Fatalf("got %q after SetBase(10)", got)
	}
	// Leaving DEC drops a fraction, from what is shown and what is pending.
	e = New()
	press(e, "7/2")
	e.Equals()
	e.SetBase(16)
	if got := e.Display().String(); got != "3" {
		t.Fatalf("7/2 in HEX = %q, want 3", got)
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
	} {
		e := New()
		e.SetBase(c.base)
		ok := e.Paste(c.in)
		if ok != c.ok || e.Display().String() != c.want {
			t.Errorf("Paste(%q) in base %d = %v, %q; want %v, %q", c.in, c.base, ok, e.Display(), c.ok, c.want)
		}
	}
}

// TestErrorBlocksInput checks that after an error every key but C and AC
// does nothing, and nothing panics.
func TestErrorBlocksInput(t *testing.T) {
	e := New()
	press(e, "1/0=")
	if e.Err() != ErrDivZero {
		t.Fatalf("1/0 gave %v", e.Err())
	}
	for _, f := range []func(){
		func() { e.Digit(5) }, e.Point, e.Backspace, e.Negate, e.Sqrt, e.Not,
		e.Percent, func() { e.Operator(OpAdd) }, e.LParen, e.RParen, e.Equals,
		e.MemRecall, e.MemAdd, e.MemSub, func() { e.ToggleBit(0) },
	} {
		f()
		if e.Err() != ErrDivZero {
			t.Fatalf("a key cleared or changed the error: %v", e.Err())
		}
	}
	if e.Paste("5") {
		t.Fatal("paste accepted over an error")
	}
	// M+ of something that fails gives that error and leaves memory alone.
	e = New()
	press(e, "1/0")
	e.MemAdd()
	if e.Err() != ErrDivZero || e.HasMemory() {
		t.Fatalf("M+ of 1/0: error %v, memory %v", e.Err(), e.HasMemory())
	}
	// A failing root is an error too.
	run(t, 10, 64, "1nr", "ERR:NEGATIVE ROOT")
}

// TestGuards covers checks that the keys cannot reach but that keep the
// engine's own functions safe.
func TestGuards(t *testing.T) {
	e := New()
	if v, ok := e.parseEntry(""); !ok || v.R.Sign() != 0 {
		t.Error(`parseEntry("") should be 0`)
	}
	if _, ok := e.parseEntry("12x"); ok {
		t.Error("parseEntry accepted a bad decimal")
	}
	if e.entryFits("12x") {
		t.Error("entryFits accepted a bad decimal")
	}
	e.Base = 16
	if _, ok := e.parseEntry("XYZ"); ok {
		t.Error("parseEntry accepted bad hex")
	}
	if v, err := e.eval(nil); err != nil || v.R.Sign() != 0 {
		t.Error("an empty expression should be 0")
	}
	huge := new(big.Int).Lsh(big.NewInt(1), 70)
	if _, err := apply(OpShl, ratInt(big.NewInt(1)), ratInt(huge), 10, 64); err != ErrShift {
		t.Errorf("a shift of 2^70 gave %v, want ErrShift", err)
	}
}
