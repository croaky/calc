package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// maxShift limits the count of a shift.
	maxShift = 10000
	// maxBits limits integer bit length, float binary exponent, and
	// rational numerator or denominator bit length. go/constant holds
	// numbers of any size, so chained shifts, repeated products, or
	// huge-exponent float additions grow without bound and exhaust CPU
	// and memory.
	maxBits = 10000
	// maxErrorTokenRunes limits how many runes of input an error echoes.
	maxErrorTokenRunes = 64
)

// sanitize replaces every non-printable rune or invalid UTF-8 byte in s
// with a Go escape sequence, so terminal output stays on one line and
// carries no control characters.
func sanitize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			fmt.Fprintf(&b, "\\x%02x", s[i])
			i++
			continue
		}
		if unicode.IsPrint(r) {
			b.WriteRune(r)
		} else {
			q := strconv.QuoteRune(r)
			b.WriteString(q[1 : len(q)-1])
		}
		i += size
	}
	return b.String()
}

// safeToken caps s at maxErrorTokenRunes runes and escapes non-printable
// runes for error messages that echo input.
func safeToken(s string) string {
	n := 0
	for i := range s {
		if n == maxErrorTokenRunes {
			return sanitize(s[:i]) + "..."
		}
		n++
	}
	return sanitize(s)
}

func checkValue(v constant.Value) (constant.Value, error) {
	switch v.Kind() {
	case constant.Unknown:
		return nil, errors.New("number overflow")
	case constant.Int:
		if constant.BitLen(v) > maxBits {
			return nil, fmt.Errorf("integer exceeds %d bits", maxBits)
		}
	case constant.Float:
		switch x := constant.Val(v).(type) {
		case *big.Float:
			exp := x.MantExp(nil)
			if x.IsInf() || exp > maxBits || exp < -maxBits {
				return nil, fmt.Errorf("float exceeds %d bits", maxBits)
			}
		case *big.Rat:
			if x.Num().BitLen() > maxBits || x.Denom().BitLen() > maxBits {
				return nil, fmt.Errorf("float exceeds %d bits", maxBits)
			}
		}
	}
	return v, nil
}

// eval parses line as a Go expression and computes it with exact
// arithmetic. The identifier _ reads prev, the result of the line
// before, as in irb. A nil prev makes _ an error.
//
// The operators are Go's, with one difference: / of two integers gives
// the exact quotient, so 7/2 is 3.5 and not 3. go/constant keeps a
// quotient as a rational, so 0.1*3 is exactly 0.3.
func eval(line string, prev constant.Value) (constant.Value, error) {
	expr, err := parser.ParseExpr(line)
	if err != nil {
		return nil, errors.New("syntax error")
	}
	return evalExpr(expr, prev)
}

func evalExpr(e ast.Expr, prev constant.Value) (constant.Value, error) {
	switch e := e.(type) {
	case *ast.BasicLit:
		if e.Kind != token.INT && e.Kind != token.FLOAT {
			return nil, fmt.Errorf("%s is not a number", safeToken(e.Value))
		}
		return checkValue(constant.MakeFromLiteral(e.Value, e.Kind, 0))
	case *ast.Ident:
		if e.Name != "_" {
			return nil, fmt.Errorf("unknown name %s", safeToken(e.Name))
		}
		if prev == nil {
			return nil, errors.New("_ has no value yet")
		}
		return checkValue(prev)
	case *ast.ParenExpr:
		return evalExpr(e.X, prev)
	case *ast.UnaryExpr:
		x, err := evalExpr(e.X, prev)
		if err != nil {
			return nil, err
		}
		return unary(e.Op, x)
	case *ast.BinaryExpr:
		x, err := evalExpr(e.X, prev)
		if err != nil {
			return nil, err
		}
		y, err := evalExpr(e.Y, prev)
		if err != nil {
			return nil, err
		}
		return binary(e.Op, x, y)
	}
	return nil, errors.New("unsupported expression")
}

func unary(op token.Token, x constant.Value) (constant.Value, error) {
	switch op {
	case token.ADD, token.SUB:
		return checkValue(constant.UnaryOp(op, x, 0))
	case token.XOR:
		if x.Kind() != constant.Int {
			return nil, errors.New("^ needs an integer")
		}
		return checkValue(constant.UnaryOp(op, x, 0))
	}
	return nil, fmt.Errorf("unsupported operator %s", op)
}

func binary(op token.Token, x, y constant.Value) (constant.Value, error) {
	ints := x.Kind() == constant.Int && y.Kind() == constant.Int
	switch op {
	case token.ADD, token.SUB, token.MUL:
		if op == token.MUL && ints && constant.Sign(x) != 0 && constant.Sign(y) != 0 {
			if constant.BitLen(x)+constant.BitLen(y)-1 > maxBits {
				return nil, fmt.Errorf("integer exceeds %d bits", maxBits)
			}
		}
		return checkValue(constant.BinaryOp(x, op, y))
	case token.QUO:
		if constant.Sign(y) == 0 {
			return nil, errors.New("division by zero")
		}
		// BinaryOp gives the exact quotient for token.QUO, also for
		// two integers. token.QUO_ASSIGN is what asks for Go's
		// truncated integer division.
		return checkValue(constant.BinaryOp(x, op, y))
	case token.REM:
		if !ints {
			return nil, errors.New("% needs integers")
		}
		if constant.Sign(y) == 0 {
			return nil, errors.New("division by zero")
		}
		return checkValue(constant.BinaryOp(x, op, y))
	case token.AND, token.OR, token.XOR, token.AND_NOT:
		if !ints {
			return nil, fmt.Errorf("%s needs integers", op)
		}
		return checkValue(constant.BinaryOp(x, op, y))
	case token.SHL, token.SHR:
		if !ints {
			return nil, fmt.Errorf("%s needs integers", op)
		}
		n, ok := constant.Uint64Val(y)
		if !ok || n > maxShift {
			return nil, fmt.Errorf("shift count must be 0 to %d", maxShift)
		}
		if op == token.SHL && constant.Sign(x) != 0 && uint64(constant.BitLen(x))+n > maxBits {
			return nil, fmt.Errorf("integer exceeds %d bits", maxBits)
		}
		return checkValue(constant.Shift(x, op, uint(n)))
	}
	return nil, fmt.Errorf("unsupported operator %s", op)
}

// format writes v for a person to read. An integer prints every digit.
// Any other value prints as a decimal with the fewest digits that
// identify its nearest float64, and in exponent form when that decimal
// is very large or very small.
func format(v constant.Value) string {
	if v.Kind() == constant.Int {
		return v.ExactString()
	}
	f, _ := constant.Float64Val(v)
	if math.IsInf(f, 0) {
		return "overflow"
	}
	if a := math.Abs(f); a != 0 && (a < 1e-6 || a >= 1e21) {
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}
