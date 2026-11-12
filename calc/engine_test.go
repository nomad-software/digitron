package calc

// The tests in this file check the operations: arithmetic, √, %, memory,
// the base and word size, bits, C and errors. press and run, used by every
// test in the package, are here.

import (
	"math/big"
	"strings"
	"testing"
)

// press drives the engine with a compact key script:
// digits/hex letters, . + - * / & | ^ < > ( ) = % ~ (NOT) n (+/-) r (√) b (backspace).
func press(e *Engine, keys string) {
	for _, k := range keys {
		switch {
		case k >= '0' && k <= '9':
			e.Digit(int(k - '0'))
		case k >= 'A' && k <= 'F':
			e.Digit(int(k-'A') + 10)
		}
		switch k {
		case '.':
			e.Point()
		case '+':
			e.Operator(OpAdd)
		case '-':
			e.Operator(OpSub)
		case '*':
			e.Operator(OpMul)
		case '/':
			e.Operator(OpDiv)
		case '&':
			e.Operator(OpAnd)
		case '|':
			e.Operator(OpOr)
		case '^':
			e.Operator(OpXor)
		case '<':
			e.Operator(OpShl)
		case '>':
			e.Operator(OpShr)
		case '(':
			e.LParen()
		case ')':
			e.RParen()
		case '=':
			e.Equals()
		case '%':
			e.Percent()
		case '~':
			e.Not()
		case 'n':
			e.Negate()
		case 'r':
			e.Sqrt()
		case 'b':
			e.Backspace()
		}
	}
}

func run(t *testing.T, base, width int, keys, want string) {
	t.Helper()
	e := New()
	e.Base, e.Width = base, width
	press(e, keys)
	got := e.Display().String()
	if e.Err() != nil {
		got = "ERR:" + e.Err().Error()
	}
	if got != want {
		t.Errorf("base %d width %d %q: got %q want %q", base, width, keys, got, want)
	}
}

func TestArithmetic(t *testing.T) {
	cases := []struct{ keys, want string }{
		{"1+2*3=", "7"},
		{"(1+2)*3=", "9"},
		{"2(3+4=", "14"},
		{"0.1+0.2=", "0.3"},
		{"1/3*3=", "1"},
		{"1/3=", "0.3"},
		{"10-2-3=", "5"},
		{"100/4/5=", "5"},
		{"5n*3=", "-15"},
		{"7/0=", "ERR:DIV BY ZERO"},
		{"200+10%=", "220"},
		{"50*10%=", "5"},
		{"16r", "4"},
		{"9/4=r", "1.5"},
		{"1n=r", "ERR:NEGATIVE ROOT"},
		{"5+=", "5"},
		{"12b3", "13"},
		{"2+3=*4=", "20"},
		{"2+3=7", "7"},
		{"3037000499*3037000499=", "9223372030926249001"},
		{"9223372036854775807*2=", "18446744073709551614"}, // DEC has no word size
		{"99999999999999999999*99999999999999999999=", "9999999999999999999800000000000000000001"},
		{"1/7=", "0.142857"},
		{"1/6=", "0.16"},
		{"1.5.5", "1.55"},
		{"6&3=", "2"},
		{"6|3=", "7"},
		{"6^3=", "5"},
		{"1<62=", "4611686018427387904"},
		{"1<63=", "9223372036854775808"},
		{"1<100=", "1267650600228229401496703205376"},
		{"1+2<3=", "24"}, // shifts bind looser than +
		{"1|2&3=", "3"},  // & binds tighter than |
		{"0.5&1=", "ERR:NOT INTEGER"},
		{"5~", "-6"},
		{"2+(3*(4+1))=", "17"},
	}
	for _, c := range cases {
		run(t, 10, 64, c.keys, c.want)
	}
}

func TestWordSize(t *testing.T) {
	run(t, 16, 8, "FF", "FF")
	run(t, 10, 8, "127+1=", "128") // DEC has no word size
	run(t, 10, 8, "300", "300")
	run(t, 16, 8, "7F+1=", "80") // HEX wraps to it
	run(t, 8, 8, "177+1=", "200")
	run(t, 10, 8, "1n", "-1")
	run(t, 16, 8, "1n", "FF")
	run(t, 16, 8, "0~", "FF")
	run(t, 16, 16, "FF<4=", "FF0")
	run(t, 16, 8, "FF<4=", "F0")
	run(t, 16, 8, "1FF", "1F") // third digit would overflow
	run(t, 8, 8, "1<7=", "200")
	run(t, 16, 32, "80000000>4=", "F8000000") // arithmetic shift
	run(t, 16, 64, "1n", "FFFFFFFFFFFFFFFF")
	run(t, 10, 64, "9223372036854775807+1=", "9223372036854775808")
	run(t, 16, 64, "7FFFFFFFFFFFFFFF+1=", "8000000000000000")
	run(t, 16, 64, "FFFFFFFFFFFFFFFF+1=", "0")
	run(t, 16, 64, "FFFFFFFFFFFFFFFFF", "FFFFFFFFFFFFFFFF") // 17th digit would overflow
	run(t, 16, 64, "7/2=", "3")
	run(t, 8, 8, "1018", "101") // 8 is not an octal digit
}

func TestWidthCap(t *testing.T) {
	e := New()
	if e.Width != 64 {
		t.Fatalf("default width %d, want 64", e.Width)
	}
	for _, w := range []int{0, 128, 7} {
		e.SetWidth(w)
		if e.Width != 64 {
			t.Fatalf("SetWidth(%d) changed width to %d", w, e.Width)
		}
	}
	e.SetWidth(16)
	if e.Width != 16 {
		t.Fatalf("SetWidth(16) gave %d", e.Width)
	}
}

func TestBaseSwitch(t *testing.T) {
	e := New()
	press(e, "255")
	e.SetBase(16)
	if e.Display().String() != "FF" {
		t.Fatalf("got %s", e.Display())
	}
	e.SetBase(10)
	press(e, "/2=")
	if e.Display().String() != "127.5" {
		t.Fatalf("got %s", e.Display())
	}
	e.SetBase(16)
	if e.Display().String() != "7F" {
		t.Fatalf("got %s", e.Display())
	}
}

func TestMemory(t *testing.T) {
	e := New()
	press(e, "2+3")
	e.MemAdd()
	if e.Display().String() != "5" || !e.HasMemory() {
		t.Fatal("M+ should evaluate and store")
	}
	press(e, "10")
	e.MemSub()
	e.MemRecall()
	if e.Display().String() != "-5" {
		t.Fatalf("MR = %s", e.Display())
	}
	press(e, "*2=")
	if e.Display().String() != "-10" {
		t.Fatalf("got %s", e.Display())
	}
	e.MemClear()
	if e.HasMemory() {
		t.Fatal("MC")
	}
}

func TestBitsAndToggle(t *testing.T) {
	e := New()
	e.SetWidth(8)
	press(e, "1n")
	for i, v := range BitsOf(e.Value(), 8) {
		if !v {
			t.Fatalf("bit %d of -1 should be set", i)
		}
	}
	e.ToggleBit(7)
	if e.Display().String() != "127" {
		t.Fatalf("got %s", e.Display())
	}
	e.SetWidth(64)
	e.ToggleBit(63)
	if e.Display().String() != "-9223372036854775681" {
		t.Fatalf("toggle bit 63: got %s", e.Display())
	}
	e.ToggleBit(64) // beyond the word: no change
	if e.Display().String() != "-9223372036854775681" {
		t.Fatalf("toggle bit 64: got %s", e.Display())
	}

	// A fraction's bits are floating point, and are not clicked: toggling
	// one would cut the fraction off.
	e = New()
	press(e, "1/2=")
	e.ToggleBit(0)
	if e.Display().String() != "0.5" {
		t.Fatalf("toggling a bit of 0.5 gave %s", e.Display())
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

// TestPercentInWholeBases checks that % in OCT and HEX gives a whole
// number, cut towards zero as ÷ is, and does not hide a fraction behind
// the display: 1% used to show 0 and hold 1/100, so that 1% × 64 (100) was
// 1. Found by TestKeysAtRandom.
func TestPercentInWholeBases(t *testing.T) {
	for _, c := range []struct {
		base       int
		keys, want string
	}{
		{16, "1%*64=", "0"},
		{16, "64%", "1"},
		{16, "C8+A%=", "DC"},      // 200 + 10% of 200
		{16, "C7+A%=", "DA"},      // 199 + 19.9, cut to 19
		{16, "1n%", "0"},          // −0.01, cut towards zero
		{16, "C8n%", "FFFE"},      // −2
		{8, "144%", "1"},          // 100 in octal
		{16, "7FFF+C8%=", "7FFD"}, // 32767 + 65534 wraps in 16 bits
	} {
		e := New()
		e.SetWidth(16)
		e.SetBase(c.base)
		press(e, c.keys)
		if got := e.Display().String(); got != c.want || !e.Value().IsInt() {
			t.Errorf("base %d %q = %q (%s), want %q", c.base, c.keys, got, e.Value().R.RatString(), c.want)
		}
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
	// Shift counts too big for an int64 act as one just past MaxBits.
	huge := ratInt(new(big.Int).Lsh(big.NewInt(1), 70))
	one, zero := ratInt(big.NewInt(1)), ratInt(big.NewInt(0))
	for _, c := range []struct {
		name  string
		op    Op
		a     Value
		want  int64
		error error
	}{
		{"1 << 2^70", OpShl, one, 0, ErrTooBig},
		{"0 << 2^70", OpShl, zero, 0, nil},
		{"1 >> 2^70", OpShr, one, 0, nil},
		{"-1 >> 2^70", OpShr, ratInt(big.NewInt(-1)), -1, nil},
	} {
		v, err := apply(c.op, c.a, huge, 10, 64)
		if err != c.error || (err == nil && v.R.Cmp(big.NewRat(c.want, 1)) != 0) {
			t.Errorf("%s = %v, %v; want %d, %v", c.name, v.R, err, c.want, c.error)
		}
	}
}

// TestSqrt checks that the root of a non-square is marked approximate, is
// kept to SqrtDigits places however few are shown, is shown and copied
// rounded to fill the display, and passes its mark on to what is worked
// out from it. Exact roots are in TestFamousBugs.
func TestSqrt(t *testing.T) {
	e := New()
	press(e, "2r")
	if !e.Approx() {
		t.Fatal("√2 should be marked approximate")
	}
	if got := e.Value().R.FloatString(60); !strings.HasPrefix(got, "1.41421356237309504880168872420969807856967187537694") {
		t.Fatalf("√2 is kept as %s", got)
	}
	if got := e.Display().String(); got != "1.4142135623730950488016887" {
		t.Errorf("√2 shows %q", got)
	}
	if got := e.CopyText(); got != "1.4142135623730950488016887" {
		t.Errorf("√2 copies as %q", got)
	}
	// Kept to 64 places, √2 × √2 rounds to 2 as it is shown.
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

// TestClear checks C: it clears the entry being typed, an error, the
// group just closed, and, straight after =, everything.
func TestClear(t *testing.T) {
	e := New()
	press(e, "5+3")
	e.ClearEntry()
	press(e, "4=")
	if e.Display().String() != "9" {
		t.Fatalf("5 + 3 C 4 = %s, want 9", e.Display())
	}
	press(e, "1/0=")
	e.ClearEntry()
	if e.Err() != nil {
		t.Fatal("C should clear an error")
	}
	e = New()
	press(e, "2+3=")
	e.ClearEntry()
	if e.Display().String() != "0" || e.Trail() != "" {
		t.Fatalf("C after = left %q, trail %q", e.Display(), e.Trail())
	}
	e = New()
	press(e, "1+(2)")
	e.ClearEntry()
	press(e, "3=")
	if got := e.Display().String(); got != "4" {
		t.Fatalf("1 + (2) C 3 = %q, want 4", got)
	}
}
