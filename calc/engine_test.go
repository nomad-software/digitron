package calc

import (
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
		{"9223372036854775807*2=", "-2"}, // wraps at 64 bits
		{"1/7=", "0.142857"},
		{"1/6=", "0.16"},
		{"1.5.5", "1.55"},
		{"6&3=", "2"},
		{"6|3=", "7"},
		{"6^3=", "5"},
		{"1<62=", "4611686018427387904"},
		{"1<63=", "-9223372036854775808"},
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

func TestSqrtApprox(t *testing.T) {
	e := New()
	press(e, "2r")
	if !e.Approx() {
		t.Fatal("√2 should be approx")
	}
	// It is kept to SqrtDigits places, however few are shown.
	if got := e.Value().R.FloatString(60); !strings.HasPrefix(got, "1.41421356237309504880168872420969807856967187537694") {
		t.Fatalf("√2 is kept as %s", got)
	}
	press(e, "*2=")
	if !e.Approx() {
		t.Fatal("approx should propagate")
	}
}

func TestWordSize(t *testing.T) {
	run(t, 16, 8, "FF", "FF")
	run(t, 10, 8, "127+1=", "-128")
	run(t, 10, 8, "128", "12") // 128 does not fit signed 8-bit
	run(t, 10, 8, "1n", "-1")
	run(t, 16, 8, "1n", "FF")
	run(t, 16, 8, "0~", "FF")
	run(t, 16, 16, "FF<4=", "FF0")
	run(t, 16, 8, "FF<4=", "F0")
	run(t, 16, 8, "1FF", "1F") // third digit would overflow
	run(t, 2, 8, "1<111=", "10000000")
	run(t, 16, 32, "80000000>4=", "F8000000") // arithmetic shift
	run(t, 16, 64, "1n", "FFFFFFFFFFFFFFFF")
	run(t, 10, 64, "9223372036854775807+1=", "-9223372036854775808")
	run(t, 16, 64, "FFFFFFFFFFFFFFFF+1=", "0")
	run(t, 16, 64, "FFFFFFFFFFFFFFFFF", "FFFFFFFFFFFFFFFF") // 17th digit would overflow
	run(t, 16, 64, "7/2=", "3")
	run(t, 2, 8, "1012", "101") // 2 is not a binary digit
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
	e.SetBase(2)
	if e.Display().String() != "11111111" {
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
}

func TestPaste(t *testing.T) {
	e := New()
	e.Paste("0xff")
	if e.Display().String() != "255" {
		t.Fatalf("got %s", e.Display())
	}
	e.Paste("1/3")
	press(e, "*3=")
	if e.Display().String() != "1" {
		t.Fatalf("got %s", e.Display())
	}
	if e.Paste("hello") {
		t.Fatal("bad paste accepted")
	}
	e.Paste("1,000,000")
	if e.CopyText() != "1000000" {
		t.Fatal(e.CopyText())
	}
	press(e, "/3=")
	if e.CopyText() != "1000000/3" {
		t.Fatal(e.CopyText())
	}
}

func TestClear(t *testing.T) {
	e := New()
	press(e, "5+3")
	e.ClearEntry()
	press(e, "4=")
	if e.Display().String() != "9" {
		t.Fatalf("got %s", e.Display())
	}
	press(e, "1/0=")
	e.ClearEntry()
	if e.Err() != nil {
		t.Fatal("C should clear error")
	}
}
