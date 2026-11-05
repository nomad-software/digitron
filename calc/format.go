package calc

import (
	"math/big"
	"strings"
)

// MaxFracDigits bounds how far a decimal expansion is printed.
const MaxFracDigits = 1000

// Cell is one displayed character; Over marks a repeating (overlined) digit.
type Cell struct {
	Ch   rune
	Over bool
}

// Formatted is a value rendered for an LCD.
type Formatted struct {
	Cells []Cell
	// Truncated means the expansion was cut off (shown with a trailing "…").
	Truncated bool
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

// Format renders v in the given base. Base 10 shows exact decimals, marking a
// repeating tail with Over; other bases show the integer part as an
// unsigned two's-complement pattern of width bits.
func Format(v Value, base, width int) Formatted {
	if base != 10 {
		return plain(strings.ToUpper(unsigned(v.Trunc(), width).Text(base)))
	}
	return formatDecimal(v.R)
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
func ExactString(v Value, base, width int) string {
	if base != 10 {
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
