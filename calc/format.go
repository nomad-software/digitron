package calc

import (
	"math/big"
	"strconv"
	"strings"
)

// MaxFracDigits bounds how far a decimal expansion is printed.
const MaxFracDigits = 1000

// DisplayCells is how many characters the main display shows in its big
// digits, sign and point included. A decimal is shown exactly where it fits
// (a repeating one with its repeating part marked), and is otherwise
// rounded to as many decimal places as fill the display, wherever its point
// is. A test in package ui checks that the display has room for these.
const DisplayCells = 27

// Cell is one displayed character; Over marks a repeating (overlined) digit.
type Cell struct {
	Ch   rune
	Over bool
}

// Formatted is a value rendered for an LCD.
type Formatted struct {
	Cells []Cell
	// Truncated means the expansion was cut off at MaxFracDigits places.
	Truncated bool
	// Rounded means the value was rounded to fit DisplayCells, so what is
	// shown is not exact.
	Rounded bool
}

func (f Formatted) String() string {
	var b strings.Builder
	for _, c := range f.Cells {
		b.WriteRune(c.Ch)
	}
	return b.String()
}

func plain(s string) Formatted {
	cells := make([]Cell, 0, len(s))
	for _, r := range s {
		cells = append(cells, Cell{Ch: r})
	}
	return Formatted{Cells: cells}
}

// Format renders v in the given base. Base 10 shows a decimal fitted to
// DisplayCells (see fitDecimal); other bases show the integer part as an
// unsigned two's-complement pattern of width bits.
func Format(v Value, base, width int) Formatted {
	if base != 10 {
		return plain(strings.ToUpper(unsigned(v.Trunc(), width).Text(base)))
	}
	return fitDecimal(v)
}

// fitDecimal shows v exactly if it is exact and fits DisplayCells, marking a
// repeating tail with Over, or is a whole number. Otherwise it is rounded to as many decimal places
// as fill DisplayCells after its sign, integer part and point, so that a
// number with a long integer part has fewer places and a small one more. An
// approximate value is always rounded, as its last places are not right.
func fitDecimal(v Value) Formatted {
	if !v.Approx {
		// A whole number is never rounded: one too long for the display,
		// which DEC can make, is shown whole on more lines.
		if f := formatDecimal(v.R); v.IsInt() || (!f.Truncated && len(f.Cells) <= DisplayCells) {
			return f
		}
	}
	whole := len(new(big.Int).Abs(v.Trunc()).String())
	places := DisplayCells - whole - 1 // the point
	if v.R.Sign() < 0 {
		places-- // the sign
	}
	// A number so small that few of its digits would be left after the
	// zeros behind the point is shown in scientific notation instead.
	if v.R.Sign() != 0 {
		if exp := decimalExponent(v.R); exp < 0 && places-(-exp-1) < MinSignificant {
			return scientific(v, exp)
		}
	}
	// Rounding up can carry into the integer part (9.99… to 10.00…), but
	// then every place is 0 and none is shown, so it still fits.
	f := formatDecimal(roundPlaces(v.R, max(0, places)))
	f.Rounded = true
	return f
}

// MinSignificant is the fewest significant digits a small number is shown
// with before the display turns to scientific notation for it.
const MinSignificant = 6

// decimalExponent returns the power of ten at or below |r|, e such that
// 10^e <= |r| < 10^(e+1), for r other than 0.
func decimalExponent(r *big.Rat) int {
	a := new(big.Rat).Abs(r)
	// An estimate from the sizes in bits, put right by comparing with
	// powers of ten.
	e := digitsOfBits(a.Num().BitLen() - a.Denom().BitLen())
	for a.Cmp(pow10(e)) < 0 {
		e--
	}
	for a.Cmp(pow10(e+1)) >= 0 {
		e++
	}
	return e
}

// pow10 returns 10^e as a fraction, for e of either sign.
func pow10(e int) *big.Rat {
	p := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(max(e, -e))), nil)
	if e < 0 {
		return new(big.Rat).SetFrac(big.NewInt(1), p)
	}
	return new(big.Rat).SetInt(p)
}

// scientific shows v, whose decimal exponent is exp, as a mantissa of one
// digit before the point and as many after as fill the display, then E and
// the exponent: 1/97 ÷ 10^27 is 1.030927835051546391753E-29. It is marked
// rounded only if the mantissa had to be. It is used for exponents of -20
// and below, so a carry, which brings the exponent nearer 0, never needs
// more room.
func scientific(v Value, exp int) Formatted {
	m := new(big.Rat).Mul(v.R, pow10(-exp))                 // 1 <= |m| < 10
	places := DisplayCells - len("E"+strconv.Itoa(exp)) - 2 // the digit before the point, and the point
	if m.Sign() < 0 {
		places-- // the sign
	}
	rm := roundPlaces(m, places)
	rounded := v.Approx || rm.Cmp(m) != 0
	if new(big.Rat).Abs(rm).Cmp(big.NewRat(10, 1)) == 0 {
		// Rounded up to ±10: ±1 at the next power.
		rm.Quo(rm, big.NewRat(10, 1))
		exp++
	}
	f := formatDecimal(rm)
	for _, c := range "E" + strconv.Itoa(exp) {
		f.Cells = append(f.Cells, Cell{Ch: c})
	}
	f.Rounded = rounded
	return f
}

// roundPlaces rounds r to the given number of decimal places, halves away
// from zero.
func roundPlaces(r *big.Rat, places int) *big.Rat {
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(places)), nil)
	n := new(big.Int).Mul(new(big.Int).Abs(r.Num()), scale)
	q, rem := new(big.Int).QuoRem(n, r.Denom(), new(big.Int))
	if rem.Lsh(rem, 1).Cmp(r.Denom()) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	if r.Sign() < 0 {
		q.Neg(q)
	}
	return new(big.Rat).SetFrac(q, scale)
}

func formatDecimal(r *big.Rat) Formatted {
	num := new(big.Int).Abs(r.Num())
	den := r.Denom()
	ip, rem := new(big.Int).QuoRem(num, den, new(big.Int))

	var f Formatted
	if r.Sign() < 0 {
		f.Cells = append(f.Cells, Cell{Ch: '-'})
	}
	for _, c := range ip.String() {
		f.Cells = append(f.Cells, Cell{Ch: c})
	}
	if rem.Sign() == 0 {
		return f
	}
	f.Cells = append(f.Cells, Cell{Ch: '.'})

	// Long division, remembering where each remainder first appeared so the
	// repeating cycle can be found exactly.
	seen := map[string]int{}
	var digits []rune
	ten := big.NewInt(10)
	q := new(big.Int)
	repeatAt := -1
	for rem.Sign() != 0 && len(digits) < MaxFracDigits {
		key := string(rem.Bytes())
		if at, ok := seen[key]; ok {
			repeatAt = at
			break
		}
		seen[key] = len(digits)
		rem.Mul(rem, ten)
		q.QuoRem(rem, den, rem)
		digits = append(digits, rune('0'+q.Int64()))
	}
	for i, d := range digits {
		f.Cells = append(f.Cells, Cell{Ch: d, Over: repeatAt >= 0 && i >= repeatAt})
	}
	if rem.Sign() != 0 && repeatAt < 0 {
		f.Truncated = true
	}
	return f
}

// ExactString gives a copy/paste-friendly exact representation: a decimal
// when it terminates, otherwise "num/den" (base 10), or the integer in base.
// An approximate value is not exact to begin with, and is given as it is
// shown.
func ExactString(v Value, base, width int) string {
	if base != 10 || v.Approx {
		return Format(v, base, width).String()
	}
	if v.R.IsInt() {
		return v.R.Num().String()
	}
	f := formatDecimal(v.R)
	for _, c := range f.Cells {
		if c.Over {
			return v.R.String()
		}
	}
	if f.Truncated {
		return v.R.String()
	}
	return f.String()
}

// BitsOf returns the width-bit two's-complement pattern of v's integer
// part; bit i of the pattern is element i, least significant first.
func BitsOf(v Value, width int) []bool {
	u := unsigned(v.Trunc(), width)
	bits := make([]bool, width)
	for i := range width {
		bits[i] = u.Bit(i) == 1
	}
	return bits
}
