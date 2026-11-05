package calc

import (
	"math/big"
	"slices"
	"strings"
)

// Widths are the selectable word sizes, in bits. 64 is the largest.
var Widths = []int{8, 16, 32, 64}

type tokKind int

const (
	tNum tokKind = iota
	tOp
	tLParen
	tRParen
)

type token struct {
	kind tokKind
	v    Value
	op   Op
}

// Engine is the calculator state machine. It is driven by key presses and
// queried by the UI for what to show.
type Engine struct {
	Base  int // 2, 8, 10 or 16
	Width int // word size in bits; one of Widths

	tokens   []token
	entry    string // digits being typed, in Base (may start with '-')
	typing   bool
	hasCur   bool  // cur is a complete operand not yet pushed to tokens
	cur      Value // the operand being built
	shown    Value // what the displays show when not typing
	mem      Value
	err      error
	justEval bool
	trail    string // expression shown after '='

	// The operation = repeats: the last operator worked out by =, and the
	// operand on its right ("+ 3" of "2 + 3 ="), while hasRepeat.
	repeatOp  Op
	repeatRHS Value
	hasRepeat bool
}

func New() *Engine {
	return &Engine{Base: 10, Width: 64, cur: zero(), shown: zero(), mem: zero()}
}

// ---- queries --------------------------------------------------------------

func (e *Engine) Err() error      { return e.err }
func (e *Engine) HasMemory() bool { return e.mem.R.Sign() != 0 }
func (e *Engine) Value() Value    { return e.shown }
func (e *Engine) Approx() bool    { return e.shown.Approx }
func (e *Engine) OpenParens() int { return e.depth() }

// Display returns the main display content.
func (e *Engine) Display() Formatted {
	if e.typing {
		return plain(strings.ToUpper(e.entry))
	}
	return Format(e.shown, e.Base, e.Width)
}

// Trail is the pending expression (or the evaluated one after '=').
func (e *Engine) Trail() string {
	if e.justEval {
		return e.trail
	}
	return e.exprString(e.tokens)
}

// CopyText is the exact current value for the clipboard.
func (e *Engine) CopyText() string {
	if e.typing {
		return e.entry
	}
	return ExactString(e.shown, e.Base, e.Width)
}

func (e *Engine) exprString(toks []token) string {
	var parts []string
	for _, t := range toks {
		switch t.kind {
		case tNum:
			s := Format(t.v, e.Base, e.Width).String()
			if len(s) > 12 {
				s = s[:10] + "…"
			}
			parts = append(parts, s)
		case tOp:
			parts = append(parts, t.op.String())
		case tLParen:
			parts = append(parts, "(")
		case tRParen:
			parts = append(parts, ")")
		}
	}
	s := strings.Join(parts, " ")
	s = strings.ReplaceAll(s, "( ", "(")
	return strings.ReplaceAll(s, " )", ")")
}

func (e *Engine) depth() int {
	d := 0
	for _, t := range e.tokens {
		switch t.kind {
		case tLParen:
			d++
		case tRParen:
			d--
		}
	}
	return d
}

func (e *Engine) last() (token, bool) {
	if len(e.tokens) == 0 {
		return token{}, false
	}
	return e.tokens[len(e.tokens)-1], true
}

func (e *Engine) lastIs(k tokKind) bool {
	t, ok := e.last()
	return ok && t.kind == k
}

// ---- input ----------------------------------------------------------------

func (e *Engine) fail(err error) {
	e.err = err
	e.typing = false
}

// beginOperand prepares for a new operand.
func (e *Engine) beginOperand() {
	if e.justEval {
		e.tokens = nil
		e.justEval = false
		e.hasCur = false
	}
	if e.lastIs(tRParen) && !e.hasCur {
		e.tokens = append(e.tokens, token{kind: tOp, op: OpMul})
	}
}

// Digit types a digit 0-15 in the current base.
func (e *Engine) Digit(d int) {
	if e.err != nil || d >= e.Base {
		return
	}
	if !e.typing {
		e.beginOperand()
		e.entry = ""
	}
	ch := strings.ToUpper(big.NewInt(int64(d)).Text(16))
	next := e.entry + ch
	// Drop a redundant leading zero ("0" -> "7", "-0" -> "-7").
	if strings.TrimPrefix(e.entry, "-") == "0" {
		next = strings.TrimSuffix(e.entry, "0") + ch
	}
	if !e.entryFits(next) {
		return
	}
	e.entry = next
	e.typing = true
	e.syncEntry()
}

// Point types a decimal point (base 10 only).
func (e *Engine) Point() {
	if e.err != nil || e.Base != 10 {
		return
	}
	if !e.typing {
		e.beginOperand()
		e.entry = "0"
	}
	if strings.Contains(e.entry, ".") {
		return
	}
	e.entry += "."
	e.typing = true
	e.syncEntry()
}

func (e *Engine) parseEntry(s string) (Value, bool) {
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	s = strings.TrimSuffix(s, ".")
	if s == "" {
		s = "0"
	}
	var r *big.Rat
	if e.Base == 10 {
		var ok bool
		r, ok = new(big.Rat).SetString(s)
		if !ok {
			return Value{}, false
		}
	} else {
		i, ok := new(big.Int).SetString(s, e.Base)
		if !ok {
			return Value{}, false
		}
		r = new(big.Rat).SetInt(i)
	}
	if neg {
		r.Neg(r)
	}
	return Value{R: r}, true
}

// entryFits checks a candidate entry. In BIN/OCT/HEX any pattern up to
// Width bits is accepted (and read as two's complement); DEC has no word
// size, and any number is.
func (e *Engine) entryFits(s string) bool {
	v, ok := e.parseEntry(s)
	if !ok {
		return false
	}
	return e.Base == 10 || v.Trunc().BitLen() <= e.Width
}

// FitsWord reports whether the value shown fits the word size, as a signed
// two's-complement number of Width bits, so that its bits can be shown. In
// BIN, OCT and HEX it always does; in DEC, which has no word size, it may
// not.
func (e *Engine) FitsWord() bool {
	x := e.shown.Trunc()
	lim := new(big.Int).Lsh(big.NewInt(1), uint(e.Width-1))
	return x.Cmp(lim) < 0 && x.Cmp(new(big.Int).Neg(lim)) >= 0
}

func (e *Engine) syncEntry() {
	v, _ := e.parseEntry(e.entry)
	if e.Base != 10 {
		v = wrap(v, e.Width)
	}
	e.cur = v
	e.hasCur = true
	e.shown = v
}

// Backspace removes the last typed character.
func (e *Engine) Backspace() {
	if e.err != nil || !e.typing {
		return
	}
	r := []rune(e.entry)
	e.entry = string(r[:len(r)-1])
	if e.entry == "" || e.entry == "-" {
		e.entry = "0"
	}
	e.syncEntry()
}

// takeOperand removes and returns the operand the next unary op applies to.
func (e *Engine) takeOperand() Value {
	e.typing = false
	if e.hasCur {
		e.hasCur = false
		return e.cur
	}
	if e.lastIs(tRParen) {
		// Pop the whole parenthesised group; its value is already shown.
		d := 0
		for i := len(e.tokens) - 1; i >= 0; i-- {
			switch e.tokens[i].kind {
			case tRParen:
				d++
			case tLParen:
				d--
			}
			if d == 0 {
				e.tokens = e.tokens[:i]
				break
			}
		}
	}
	return e.shown
}

func (e *Engine) setOperand(v Value) {
	v = bound(v, e.Base, e.Width)
	e.cur = v
	e.hasCur = true
	e.typing = false
	e.shown = v
}

// Negate implements +/-.
func (e *Engine) Negate() {
	if e.err != nil {
		return
	}
	if e.typing && e.Base == 10 {
		// Change the sign of what is being typed.
		if strings.HasPrefix(e.entry, "-") {
			e.entry = e.entry[1:]
		} else {
			e.entry = "-" + e.entry
		}
		e.syncEntry()
		return
	}
	v := e.takeOperand()
	e.setOperand(Value{R: new(big.Rat).Neg(v.R), Approx: v.Approx})
}

// Sqrt implements √.
func (e *Engine) Sqrt() {
	if e.err != nil {
		return
	}
	v, err := Sqrt(e.takeOperand())
	if err != nil {
		e.fail(err)
		return
	}
	if e.Base != 10 && !v.IsInt() {
		v = Value{R: new(big.Rat).SetInt(v.Trunc()), Approx: true}
	}
	e.setOperand(v)
}

// Not implements bitwise NOT (two's complement: NOT x = -x-1, then wrapped).
func (e *Engine) Not() {
	if e.err != nil {
		return
	}
	v := e.takeOperand()
	if !v.IsInt() {
		e.fail(ErrNotInteger)
		return
	}
	e.setOperand(Value{R: new(big.Rat).SetInt(new(big.Int).Not(v.R.Num())), Approx: v.Approx})
}

// Percent: after + or − it is a percentage of the left-hand side
// (200 + 10% = 220); otherwise it divides by 100 (50 × 10% = 5).
func (e *Engine) Percent() {
	if e.err != nil {
		return
	}
	v := e.takeOperand()
	hundred := new(big.Rat).SetInt64(100)
	r := new(big.Rat).Quo(v.R, hundred)
	approx := v.Approx
	if t, ok := e.last(); ok && t.kind == tOp && (t.op == OpAdd || t.op == OpSub) {
		start := 0
		d := 0
		for i := len(e.tokens) - 1; i >= 0; i-- {
			if e.tokens[i].kind == tRParen {
				d++
			} else if e.tokens[i].kind == tLParen {
				if d == 0 {
					start = i + 1
					break
				}
				d--
			}
		}
		// Left side = the additive term(s) before the final + / −.
		left, err := e.eval(e.tokens[start : len(e.tokens)-1])
		if err != nil {
			e.fail(err)
			return
		}
		r.Mul(r, left.R)
		approx = approx || left.Approx
	}
	e.setOperand(Value{R: r, Approx: approx})
}

// Operator enters a binary operator.
func (e *Engine) Operator(op Op) {
	if e.err != nil {
		return
	}
	e.typing = false
	e.justEval = false
	switch {
	case e.hasCur:
		e.tokens = append(e.tokens, token{kind: tNum, v: e.cur})
		e.hasCur = false
	case e.lastIs(tRParen):
	case e.lastIs(tOp):
		e.tokens[len(e.tokens)-1].op = op
		return
	case e.lastIs(tLParen):
		return
	default: // empty expression: operate on what is shown
		e.tokens = append(e.tokens, token{kind: tNum, v: e.shown})
	}
	e.tokens = append(e.tokens, token{kind: tOp, op: op})
}

// LParen opens a group; "2(" is read as "2 × (".
func (e *Engine) LParen() {
	if e.err != nil {
		return
	}
	if e.justEval {
		e.tokens = nil
		e.justEval = false
		e.hasCur = false
	}
	if e.hasCur {
		e.tokens = append(e.tokens, token{kind: tNum, v: e.cur}, token{kind: tOp, op: OpMul})
		e.hasCur = false
	} else if e.lastIs(tRParen) {
		e.tokens = append(e.tokens, token{kind: tOp, op: OpMul})
	}
	e.typing = false
	e.tokens = append(e.tokens, token{kind: tLParen})
}

// RParen closes the innermost group and shows its value.
func (e *Engine) RParen() {
	if e.err != nil || e.depth() == 0 {
		return
	}
	if !e.hasCur && (e.lastIs(tOp) || e.lastIs(tLParen)) {
		return
	}
	e.typing = false
	if e.hasCur {
		e.tokens = append(e.tokens, token{kind: tNum, v: e.cur})
		e.hasCur = false
	}
	e.tokens = append(e.tokens, token{kind: tRParen})
	// Find the matching '(' and evaluate the group.
	d := 0
	for i := len(e.tokens) - 1; i >= 0; i-- {
		switch e.tokens[i].kind {
		case tRParen:
			d++
		case tLParen:
			d--
		}
		if d == 0 {
			v, err := e.eval(e.tokens[i:])
			if err != nil {
				e.fail(err)
				return
			}
			e.shown = v
			return
		}
	}
}

// Equals evaluates the whole expression, closing any open groups, and
// remembers its last operation. With no expression, straight after = or
// with only a number typed since, it does that operation again: 2 + 3 = = is
// 8, and 2 + 3 = 7 = is 10.
func (e *Engine) Equals() {
	if e.err != nil {
		return
	}
	var v Value
	if e.hasRepeat && len(e.tokens) == 0 {
		lhs := e.shown
		if e.hasCur {
			lhs = e.cur
		}
		toks := []token{{kind: tNum, v: lhs}, {kind: tOp, op: e.repeatOp}, {kind: tNum, v: e.repeatRHS}}
		var err error
		if v, err = e.eval(toks); err != nil {
			e.fail(err)
			return
		}
		e.typing = false
		e.trail = e.exprString(toks) + " ="
	} else {
		toks, ok := e.pending()
		if !ok {
			return
		}
		if v, ok = e.commit(toks); !ok {
			return
		}
		e.remember(toks)
	}
	e.cur = v
	e.hasCur = true
	e.shown = v
	e.justEval = true
}

// remember keeps the last operation of an expression for = to repeat: its
// last operator outside any group, and the value of what follows it.
func (e *Engine) remember(toks []token) {
	d := 0
	for i := len(toks) - 1; i >= 0; i-- {
		switch toks[i].kind {
		case tRParen:
			d++
		case tLParen:
			d--
		case tOp:
			if d == 0 {
				// It was worked out already, as part of the whole.
				rhs, _ := e.eval(toks[i+1:])
				e.repeatOp, e.repeatRHS, e.hasRepeat = toks[i].op, rhs, true
				return
			}
		}
	}
}

// pending returns the expression to be worked out, complete: what is typed
// added, a dangling operator or ( dropped, and open groups closed. Straight
// after =, there is none, and ok is false.
func (e *Engine) pending() (toks []token, ok bool) {
	if e.justEval {
		return nil, false
	}
	toks = append([]token(nil), e.tokens...)
	if e.hasCur {
		toks = append(toks, token{kind: tNum, v: e.cur})
	}
	for len(toks) > 0 && (toks[len(toks)-1].kind == tOp || toks[len(toks)-1].kind == tLParen) {
		toks = toks[:len(toks)-1]
	}
	if len(toks) == 0 {
		toks = []token{{kind: tNum, v: e.shown}}
	}
	d := 0
	for _, t := range toks {
		switch t.kind {
		case tLParen:
			d++
		case tRParen:
			d--
		}
	}
	for ; d > 0; d-- {
		toks = append(toks, token{kind: tRParen})
	}
	return toks, true
}

// commit works out a pending expression and clears it.
func (e *Engine) commit(toks []token) (Value, bool) {
	e.typing = false
	v, err := e.eval(toks)
	if err != nil {
		e.fail(err)
		return Value{}, false
	}
	e.trail = e.exprString(toks) + " ="
	e.tokens = nil
	e.hasCur = false
	return v, true
}

// ClearEntry (C) clears the current operand, or an error.
func (e *Engine) ClearEntry() {
	if e.err != nil {
		e.AllClear()
		return
	}
	if e.justEval {
		e.AllClear()
		return
	}
	if !e.hasCur && e.lastIs(tRParen) {
		e.takeOperand()
	}
	e.typing = false
	e.hasCur = false
	e.entry = ""
	e.shown = zero()
}

// AllClear (AC) resets everything except memory, base and width.
func (e *Engine) AllClear() {
	e.tokens = nil
	e.typing = false
	e.hasCur = false
	e.entry = ""
	e.cur = zero()
	e.shown = zero()
	e.err = nil
	e.justEval = false
	e.trail = ""
	e.hasRepeat = false
}

// ---- memory ---------------------------------------------------------------

func (e *Engine) MemClear() { e.mem = zero() }

func (e *Engine) MemRecall() {
	if e.err != nil {
		return
	}
	e.beginOperand()
	e.setOperand(e.mem)
}

func (e *Engine) memAdd(sign int) {
	if e.err != nil {
		return
	}
	// Straight after = there is nothing pending, and the result is taken.
	v := e.cur
	if toks, ok := e.pending(); ok {
		if v, ok = e.commit(toks); !ok {
			return
		}
	}
	e.cur, e.hasCur, e.shown, e.justEval = v, true, v, true
	r := new(big.Rat)
	if sign > 0 {
		r.Add(e.mem.R, v.R)
	} else {
		r.Sub(e.mem.R, v.R)
	}
	e.mem = bound(Value{R: r, Approx: e.mem.Approx || v.Approx}, e.Base, e.Width)
}

func (e *Engine) MemAdd() { e.memAdd(1) }
func (e *Engine) MemSub() { e.memAdd(-1) }

// ---- modes ----------------------------------------------------------------

// SetBase switches number base. Leaving DEC, which has no word size, drops
// a fraction and wraps the value into the word.
func (e *Engine) SetBase(b int) {
	if e.Base == b {
		return
	}
	if e.typing {
		e.typing = false
	}
	e.Base = b
	if b != 10 && e.err == nil {
		toWord := func(v Value) Value {
			return wrap(Value{R: new(big.Rat).SetInt(v.Trunc()), Approx: v.Approx}, e.Width)
		}
		e.shown = toWord(e.shown)
		if e.hasCur {
			e.cur = toWord(e.cur)
		}
	}
}

// SetWidth changes word size, wrapping current values into range in BIN,
// OCT and HEX; DEC has no word size and keeps them. Sizes not in Widths are
// ignored.
func (e *Engine) SetWidth(w int) {
	if !slices.Contains(Widths, w) {
		return
	}
	e.Width = w
	if e.typing {
		e.typing = false
	}
	e.cur = bound(e.cur, e.Base, w)
	e.shown = bound(e.shown, e.Base, w)
	e.mem = bound(e.mem, e.Base, w)
}

// ToggleBit flips bit i of the displayed value (binary display click). A
// value too big for the word has no bits shown, and is left alone.
func (e *Engine) ToggleBit(i int) {
	if e.err != nil || !e.FitsWord() {
		return
	}
	wasEval := e.justEval
	v := e.takeOperand()
	if i < 0 || i >= e.Width {
		e.setOperand(v)
		e.justEval = wasEval
		return
	}
	u := unsigned(v.Trunc(), e.Width)
	u.SetBit(u, i, u.Bit(i)^1)
	e.setOperand(ratInt(wrapInt(u, e.Width)))
	e.justEval = wasEval
}

// Paste parses text as a number in the current base. It accepts 0x/0o/0b
// prefixes, "_", "," and spaces as separators, and "a/b" fractions in DEC.
func (e *Engine) Paste(s string) bool {
	if e.err != nil {
		return false
	}
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("_", "", ",", "", " ", "", "\n", "", "\t", "").Replace(s)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	base := e.Base
	for p, b := range map[string]int{"0x": 16, "0o": 8, "0b": 2} {
		if strings.HasPrefix(s, p) {
			s, base = s[2:], b
		}
	}
	var r *big.Rat
	if base == 10 {
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
	e.beginOperand()
	e.setOperand(Value{R: r})
	return true
}

// ---- evaluation -----------------------------------------------------------

// eval evaluates a token slice with precedence and parentheses.
func (e *Engine) eval(toks []token) (Value, error) {
	p := parser{toks: toks, e: e}
	return p.expr(0)
}

type parser struct {
	toks []token
	pos  int
	e    *Engine
}

func (p *parser) primary() (Value, error) {
	if p.pos >= len(p.toks) {
		return zero(), nil
	}
	t := p.toks[p.pos]
	p.pos++
	if t.kind == tLParen {
		v, err := p.expr(0)
		if err != nil {
			return Value{}, err
		}
		if p.pos < len(p.toks) && p.toks[p.pos].kind == tRParen {
			p.pos++
		}
		return v, nil
	}
	return t.v, nil
}

// expr is precedence climbing over left-associative binary operators.
func (p *parser) expr(minPrec int) (Value, error) {
	lhs, err := p.primary()
	if err != nil {
		return Value{}, err
	}
	for p.pos < len(p.toks) {
		t := p.toks[p.pos]
		if t.kind != tOp || t.op.prec() < minPrec {
			break
		}
		p.pos++
		rhs, err := p.expr(t.op.prec() + 1)
		if err != nil {
			return Value{}, err
		}
		if lhs, err = apply(t.op, lhs, rhs, p.e.Base, p.e.Width); err != nil {
			return Value{}, err
		}
	}
	return lhs, nil
}
