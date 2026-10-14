package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"math"
	"strconv"
)

// maxShift limits the count of a shift. go/constant holds an integer
// of any size, so a typo such as 1 << 1e9 builds a value of a billion
// bits and holds the prompt for minutes.
const maxShift = 10000

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
			return nil, fmt.Errorf("%s is not a number", e.Value)
		}
		return constant.MakeFromLiteral(e.Value, e.Kind, 0), nil
	case *ast.Ident:
		if e.Name != "_" {
			return nil, fmt.Errorf("unknown name %s", e.Name)
		}
		if prev == nil {
			return nil, errors.New("_ has no value yet")
		}
		return prev, nil
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
		return constant.UnaryOp(op, x, 0), nil
	case token.XOR:
		if x.Kind() != constant.Int {
			return nil, errors.New("^ needs an integer")
		}
		return constant.UnaryOp(op, x, 0), nil
	}
	return nil, fmt.Errorf("unsupported operator %s", op)
}

func binary(op token.Token, x, y constant.Value) (constant.Value, error) {
	ints := x.Kind() == constant.Int && y.Kind() == constant.Int
	switch op {
	case token.ADD, token.SUB, token.MUL:
		return constant.BinaryOp(x, op, y), nil
	case token.QUO:
		if constant.Sign(y) == 0 {
			return nil, errors.New("division by zero")
		}
		// BinaryOp gives the exact quotient for token.QUO, also for
		// two integers. token.QUO_ASSIGN is what asks for Go's
		// truncated integer division.
		return constant.BinaryOp(x, op, y), nil
	case token.REM:
		if !ints {
			return nil, errors.New("% needs integers")
		}
		if constant.Sign(y) == 0 {
			return nil, errors.New("division by zero")
		}
		return constant.BinaryOp(x, op, y), nil
	case token.AND, token.OR, token.XOR, token.AND_NOT:
		if !ints {
			return nil, fmt.Errorf("%s needs integers", op)
		}
		return constant.BinaryOp(x, op, y), nil
	case token.SHL, token.SHR:
		if !ints {
			return nil, fmt.Errorf("%s needs integers", op)
		}
		n, ok := constant.Uint64Val(y)
		if !ok || n > maxShift {
			return nil, fmt.Errorf("shift count must be 0 to %d", maxShift)
		}
		return constant.Shift(x, op, uint(n)), nil
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
