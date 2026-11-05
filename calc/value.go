// Package calc implements an exact, arbitrary-precision programmer's
// calculator engine. All values are exact rationals (math/big.Rat); the only
// inexact operation is the square root of a non-square, which is computed to
// SqrtDigits decimal places and flagged as approximate.
package calc

import (
	"errors"
	"math/big"
)

// SqrtDigits is the number of decimal places used for irrational square roots.
const SqrtDigits = 64

// Value is an exact rational, optionally marked as approximate (it descends
// from an irrational square root).
type Value struct {
	R      *big.Rat
	Approx bool
}

func zero() Value { return Value{R: new(big.Rat)} }

func ratInt(i *big.Int) Value { return Value{R: new(big.Rat).SetInt(i)} }

func (v Value) IsInt() bool { return v.R.IsInt() }

// Trunc returns the integer part, rounding toward zero.
func (v Value) Trunc() *big.Int {
	if v.R.IsInt() {
		return new(big.Int).Set(v.R.Num())
	}
	return new(big.Int).Quo(v.R.Num(), v.R.Denom())
}

var (
	ErrDivZero    = errors.New("DIV BY ZERO")
	ErrNotInteger = errors.New("NOT INTEGER")
	ErrNegSqrt    = errors.New("NEGATIVE ROOT")
	ErrShift      = errors.New("SHIFT TOO BIG")
)

// Op is a binary operator.
type Op int

const (
	OpAdd Op = iota
	OpSub
	OpMul
	OpDiv
	OpAnd
	OpOr
	OpXor
	OpShl
	OpShr
)

// Precedence follows C: | < ^ < & < shifts < additive < multiplicative.
func (o Op) prec() int {
	switch o {
	case OpOr:
		return 1
	case OpXor:
		return 2
	case OpAnd:
		return 3
	case OpShl, OpShr:
		return 4
	case OpAdd, OpSub:
		return 5
	default:
		return 6
	}
}

func (o Op) String() string {
	return [...]string{"+", "−", "×", "÷", "AND", "OR", "XOR", "<<", ">>"}[o]
}

// wrap reduces an integer to a signed two's-complement value of the given
// bit width; non-integers pass through untouched.
func wrap(v Value, width int) Value {
	if !v.R.IsInt() {
		return v
	}
	return Value{R: new(big.Rat).SetInt(wrapInt(v.R.Num(), width)), Approx: v.Approx}
}

func wrapInt(x *big.Int, width int) *big.Int {
	mod := new(big.Int).Lsh(big.NewInt(1), uint(width))
	u := new(big.Int).Mod(x, mod) // Mod is Euclidean: 0 <= u < mod
	half := new(big.Int).Rsh(mod, 1)
	if u.Cmp(half) >= 0 {
		u.Sub(u, mod)
	}
	return u
}

// unsigned returns x as an unsigned two's-complement pattern of width bits.
func unsigned(x *big.Int, width int) *big.Int {
	mod := new(big.Int).Lsh(big.NewInt(1), uint(width))
	return new(big.Int).Mod(x, mod)
}

func apply(op Op, a, b Value, base, width int) (Value, error) {
	r := new(big.Rat)
	approx := a.Approx || b.Approx
	switch op {
	case OpAdd:
		r.Add(a.R, b.R)
	case OpSub:
		r.Sub(a.R, b.R)
	case OpMul:
		r.Mul(a.R, b.R)
	case OpDiv:
		if b.R.Sign() == 0 {
			return Value{}, ErrDivZero
		}
		r.Quo(a.R, b.R)
		if base != 10 && !r.IsInt() {
			// Programmer bases are integer-only: truncate toward zero.
			r.SetInt(new(big.Int).Quo(r.Num(), r.Denom()))
		}
	case OpAnd, OpOr, OpXor:
		if !a.R.IsInt() || !b.R.IsInt() {
			return Value{}, ErrNotInteger
		}
		x, y := a.R.Num(), b.R.Num()
		z := new(big.Int)
		switch op {
		case OpAnd:
			z.And(x, y)
		case OpOr:
			z.Or(x, y)
		default:
			z.Xor(x, y)
		}
		r.SetInt(z)
	case OpShl, OpShr:
		if !a.R.IsInt() || !b.R.IsInt() {
			return Value{}, ErrNotInteger
		}
		n := b.R.Num()
		if !n.IsInt64() {
			return Value{}, ErrShift
		}
		s := n.Int64()
		if op == OpShr {
			s = -s
		}
		// Anything beyond the word width gives the same result as width+1.
		lim := int64(width + 1)
		s = max(-lim, min(lim, s))
		z := new(big.Int)
		if s >= 0 {
			z.Lsh(a.R.Num(), uint(s))
		} else {
			z.Rsh(a.R.Num(), uint(-s)) // arithmetic shift (floors negatives)
		}
		r.SetInt(z)
	}
	return wrap(Value{R: r, Approx: approx}, width), nil
}

// Sqrt returns the exact square root when the value is a ratio of perfect
// squares, otherwise a truncated SqrtDigits-place decimal flagged Approx.
func Sqrt(v Value) (Value, error) {
	if v.R.Sign() < 0 {
		return Value{}, ErrNegSqrt
	}
	n, d := v.R.Num(), v.R.Denom()
	sn, sd := new(big.Int).Sqrt(n), new(big.Int).Sqrt(d)
	if new(big.Int).Mul(sn, sn).Cmp(n) == 0 && new(big.Int).Mul(sd, sd).Cmp(d) == 0 {
		return Value{R: new(big.Rat).SetFrac(sn, sd), Approx: v.Approx}, nil
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(SqrtDigits), nil)
	// floor(sqrt(n/d) * 10^k) = floor(sqrt(n * 10^2k / d))
	t := new(big.Int).Mul(n, scale)
	t.Mul(t, scale)
	t.Quo(t, d)
	t.Sqrt(t)
	return Value{R: new(big.Rat).SetFrac(t, scale), Approx: true}, nil
}
