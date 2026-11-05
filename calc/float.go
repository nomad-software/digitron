package calc

import (
	"fmt"
	"math/big"
)

// FloatFormat is the IEEE 754 binary floating-point format of a word size:
// a sign bit, then ExpBits of exponent, then MantBits of mantissa (the
// fraction, without its leading 1).
type FloatFormat struct {
	ExpBits, MantBits int
}

// FloatFormats are the formats the binary display shows a fraction in, by
// word size: double, single and half precision, and for 8 bits FP8 E5M2,
// which is laid out as IEEE 754 is.
var FloatFormats = map[int]FloatFormat{
	8:  {ExpBits: 5, MantBits: 2},
	16: {ExpBits: 5, MantBits: 10},
	32: {ExpBits: 8, MantBits: 23},
	64: {ExpBits: 11, MantBits: 52},
}

// IsFloat reports whether the binary display shows the value as floating
// point: in DEC, when it has a fractional part.
func (e *Engine) IsFloat() bool {
	return e.Base == 10 && !e.shown.IsInt()
}

// FloatBitsOf returns v encoded in the floating-point format of the word
// size, bit i of the pattern being element i, least significant first. It
// is rounded to the nearest value the format holds, a tie to the one with
// an even mantissa, as IEEE 754 does by default; a value too small for the
// format becomes subnormal or zero, and one too big becomes infinity.
func FloatBitsOf(v Value, width int) []bool {
	f := FloatFormats[width]
	pattern := floatPattern(v.R, f)
	bits := make([]bool, width)
	for i := range width {
		bits[i] = pattern.Bit(i) == 1
	}
	return bits
}

// floatPattern encodes r in f, as the unsigned integer the bits make.
func floatPattern(r *big.Rat, f FloatFormat) *big.Int {
	bias := 1<<(f.ExpBits-1) - 1
	maxExp := 1<<f.ExpBits - 1 // all ones: infinity

	a := new(big.Rat).Abs(r)
	// The exponent and mantissa together, as an integer: for a normal
	// number exponent<<MantBits plus the mantissa, which, with the leading 1
	// counted in the mantissa, is (exponent-1)<<MantBits plus it. A subnormal
	// one, and zero, is just its mantissa, so a mantissa rounded up into the
	// next exponent carries into it of itself.
	mag := new(big.Int)
	if a.Sign() != 0 {
		// e is the power of 2 at or below a: 2^e <= a < 2^(e+1).
		e := a.Num().BitLen() - a.Denom().BitLen()
		if pow2(e).Cmp(a) > 0 {
			e--
		}
		if e < 1-bias {
			// Subnormal: the mantissa counts units of 2^(1-bias-MantBits).
			mag = roundEven(new(big.Rat).Mul(a, pow2(bias-1+f.MantBits)))
		} else {
			// Normal: the mantissa, with its leading 1, is a scaled to
			// MantBits+1 bits.
			m := roundEven(new(big.Rat).Mul(a, pow2(f.MantBits-e)))
			mag.Lsh(big.NewInt(int64(e+bias-1)), uint(f.MantBits))
			mag.Add(mag, m)
		}
		if limit := new(big.Int).Lsh(big.NewInt(int64(maxExp)), uint(f.MantBits)); mag.Cmp(limit) >= 0 {
			mag = limit // too big: infinity
		}
	}
	if r.Sign() < 0 {
		mag.SetBit(mag, f.ExpBits+f.MantBits, 1)
	}
	return mag
}

// FloatRangeError is the error of a fraction too big for the floating-point
// format of a word size.
type FloatRangeError struct{ Width int }

func (err FloatRangeError) Error() string {
	return fmt.Sprintf("TOO BIG FOR FP%d", err.Width)
}

// toFloat returns v as the nearest number the floating-point format of the
// word size holds, marked approximate if that is not v itself. A whole
// number, which DEC keeps as it is, is returned unchanged.
func toFloat(v Value, width int) (Value, error) {
	if v.IsInt() {
		return v, nil
	}
	f := FloatFormats[width]
	r, ok := floatValue(floatPattern(v.R, f), f)
	if !ok {
		return Value{}, FloatRangeError{width}
	}
	return Value{R: r, Approx: v.Approx || r.Cmp(v.R) != 0}, nil
}

// floatValue returns the number a pattern of f stands for, and false for an
// infinity, which has none. Patterns that are not numbers are never made.
func floatValue(pattern *big.Int, f FloatFormat) (*big.Rat, bool) {
	bias := 1<<(f.ExpBits-1) - 1
	// field returns the bits of the pattern from bit from on.
	field := func(from, bits int) *big.Int {
		mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(bits)), big.NewInt(1))
		return mask.And(mask, new(big.Int).Rsh(pattern, uint(from)))
	}
	exp, mant := int(field(f.MantBits, f.ExpBits).Int64()), field(0, f.MantBits)
	if exp == 1<<f.ExpBits-1 {
		return nil, false
	}
	r := new(big.Rat).SetInt(mant)
	if exp == 0 {
		// Subnormal: units of 2^(1-bias-MantBits).
		r.Mul(r, pow2(1-bias-f.MantBits))
	} else {
		r.Add(r, pow2(f.MantBits))
		r.Mul(r, pow2(exp-bias-f.MantBits))
	}
	if pattern.Bit(f.ExpBits+f.MantBits) == 1 {
		r.Neg(r)
	}
	return r, true
}

// pow2 returns 2^e, for e of either sign.
func pow2(e int) *big.Rat {
	if e >= 0 {
		return new(big.Rat).SetInt(new(big.Int).Lsh(big.NewInt(1), uint(e)))
	}
	return new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Lsh(big.NewInt(1), uint(-e)))
}

// roundEven rounds a non-negative r to the nearest integer, a tie to the even
// one.
func roundEven(r *big.Rat) *big.Int {
	q, rem := new(big.Int).QuoRem(r.Num(), r.Denom(), new(big.Int))
	switch rem.Lsh(rem, 1).Cmp(r.Denom()) {
	case 1:
		q.Add(q, big.NewInt(1))
	case 0:
		if q.Bit(0) == 1 {
			q.Add(q, big.NewInt(1))
		}
	}
	return q
}
