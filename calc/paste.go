package calc

// Copy and paste: the exact value for the clipboard, and reading a number
// from text, within MaxBits.

import (
	"math/big"
	"strconv"
	"strings"
)

// CopyText is the exact current value for the clipboard.
func (e *Engine) CopyText() string {
	if e.typing {
		return e.entry
	}
	return ExactString(e.shown, e.Base, e.Width)
}

// Paste parses text as a number in the current base. It accepts 0x/0o/0b
// prefixes, "_", "," and spaces as separators, and in DEC "a/b" fractions
// and scientific notation such as 1.5e-30, as a tiny number is shown. It
// refuses a number beyond MaxBits, or too big for the word in OCT and HEX,
// and reports whether it took the text.
func (e *Engine) Paste(s string) bool {
	if e.err != nil {
		return false
	}
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("_", "", ",", "", " ", "", "\n", "", "\t", "").Replace(s)
	// Text too long for a number within MaxBits is refused before it is
	// read, as reading it would take long.
	if len(s) > maxPaste {
		return false
	}
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	// Only one prefix is read: 0x0b1 is hex B1, not binary 1.
	base := e.Base
	for _, p := range []struct {
		prefix string
		base   int
	}{{"0x", 16}, {"0o", 8}, {"0b", 2}} {
		if rest, ok := strings.CutPrefix(s, p.prefix); ok {
			s, base = rest, p.base
			break
		}
	}
	var r *big.Rat
	if base == 10 {
		// So is an exponent too big for MaxBits, such as 1e1000000000,
		// which would ask for a number of a billion digits.
		if exponentTooBig(s) {
			return false
		}
		var ok bool
		if r, ok = new(big.Rat).SetString(s); !ok {
			return false
		}
	} else {
		i, ok := new(big.Int).SetString(s, base)
		if !ok {
			return false
		}
		if e.Base != 10 && i.BitLen() > e.Width {
			return false
		}
		r = new(big.Rat).SetInt(i)
	}
	if neg {
		r.Neg(r)
	}
	if tooBig(r) {
		return false
	}
	e.beginOperand()
	e.setOperand(Value{R: r})
	return true
}

// maxPaste is the longest text Paste reads: a number of more digits than
// this is beyond MaxBits in decimal, octal and hex.
const maxPaste = MaxBits / 3

// exponentTooBig reports whether decimal text such as 1.5e-30 has an
// exponent that would put it beyond MaxBits.
func exponentTooBig(s string) bool {
	_, exp, ok := strings.Cut(s, "e")
	if !ok {
		return false
	}
	n, err := strconv.Atoi(exp)
	return err != nil || max(n, -n) > maxDigits
}
