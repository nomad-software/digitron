package ui

import (
	"strings"

	"digitron/calc"
)

// Press performs the action of the key with the given id.
func Press(e *calc.Engine, id string) {
	if !KeyEnabled(e, id) {
		return
	}
	if len(id) == 1 && id[0] >= '0' && id[0] <= '9' {
		e.Digit(int(id[0] - '0'))
		return
	}
	if len(id) == 2 && id[0] == 'h' {
		e.Digit(int(id[1]-'A') + 10)
		return
	}
	switch id {
	case "oct":
		e.SetBase(8)
	case "dec":
		e.SetBase(10)
	case "hex":
		e.SetBase(16)
	case "shl":
		e.Operator(calc.OpShl)
	case "shr":
		e.Operator(calc.OpShr)
	case "and":
		e.Operator(calc.OpAnd)
	case "or":
		e.Operator(calc.OpOr)
	case "xor":
		e.Operator(calc.OpXor)
	case "not":
		e.Not()
	case "(":
		e.LParen()
	case ")":
		e.RParen()
	case "bs":
		e.Backspace()
	case "mc":
		e.MemClear()
	case "mr":
		e.MemRecall()
	case "m+":
		e.MemAdd()
	case "m-":
		e.MemSub()
	case "div":
		e.Operator(calc.OpDiv)
	case "mul":
		e.Operator(calc.OpMul)
	case "sub":
		e.Operator(calc.OpSub)
	case "add":
		e.Operator(calc.OpAdd)
	case "c":
		e.ClearEntry()
	case "ac":
		e.AllClear()
	case "pct":
		e.Percent()
	case "sqrt":
		e.Sqrt()
	case "eq":
		e.Equals()
	case ".":
		e.Point()
	case "neg":
		e.Negate()
	}
}

// KeyForText maps typed text (from a key event) to a key id.
func KeyForText(t string) string {
	if len(t) != 1 {
		return ""
	}
	ch := t[0]
	switch {
	case ch >= '0' && ch <= '9':
		return t
	case ch >= 'a' && ch <= 'f', ch >= 'A' && ch <= 'F':
		return "h" + strings.ToUpper(t)
	}
	return map[byte]string{
		'+': "add", '-': "sub", '*': "mul", 'x': "mul", '/': "div",
		'=': "eq", '\r': "eq", '.': ".", ',': ".", '%': "pct",
		'(': "(", ')': ")", '&': "and", '|': "or", '^': "xor", '~': "not",
		'<': "shl", '>': "shr", 'r': "sqrt", 'n': "neg", 'w': "ws", 's': "snd",
	}[ch]
}
