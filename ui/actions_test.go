package ui

import (
	"testing"

	"digitron/calc"
)

// TestKeyForText checks that every character the keyboard maps names a key
// on the calculator's face, and that the hex digits are the same keys typed
// in either case.
func TestKeyForText(t *testing.T) {
	for c := rune(0); c < 128; c++ {
		id := KeyForText(string(c))
		if id == "" {
			continue
		}
		if _, ok := keyIDs()[id]; !ok {
			t.Errorf("%q maps to %q, which is no key", c, id)
		}
	}
	for _, c := range "abcdef" {
		if lo, up := KeyForText(string(c)), KeyForText(string(c-'a'+'A')); lo != up || lo == "" {
			t.Errorf("%q gives %q and its capital %q", c, lo, up)
		}
	}
	if id := KeyForText("ab"); id != "" {
		t.Errorf("two characters gave %q", id)
	}
}

// TestKeyEnabled checks that a digit key is greyed out, and does nothing,
// exactly when it is not a digit of the base, and the point outside DEC.
func TestKeyEnabled(t *testing.T) {
	for _, base := range []int{8, 10, 16} {
		e := calc.New()
		e.SetBase(base)
		for d, id := range []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "hA", "hB", "hC", "hD", "hE", "hF"} {
			if got := KeyEnabled(e, id); got != (d < base) {
				t.Errorf("base %d: %s enabled = %v", base, id, got)
			}
			if d >= base {
				Press(e, id)
				if e.Display().String() != "0" {
					t.Errorf("base %d: a greyed-out %s typed %s", base, id, e.Display())
				}
			}
		}
		if got := KeyEnabled(e, "."); got != (base == 10) {
			t.Errorf("base %d: point enabled = %v", base, got)
		}
	}
}

func keyIDs() map[string]bool {
	ids := map[string]bool{}
	for _, k := range Keys {
		ids[k.ID] = true
	}
	return ids
}
