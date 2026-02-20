package expression

import (
	"fmt"
	"math"

	"github.com/retroenv/retroasm/pkg/lexer/token"
)

var operatorPriority = map[token.Type]struct {
	priority         int
	rightAssociative bool
}{
	token.Caret:    {priority: 3, rightAssociative: true},
	token.Asterisk: {priority: 2, rightAssociative: false},
	token.Slash:    {priority: 2, rightAssociative: false},
	token.Percent:  {priority: 2, rightAssociative: false},
	token.Plus:     {priority: 1, rightAssociative: false},
	token.Minus:    {priority: 1, rightAssociative: false},
	token.Equals:   {priority: 1, rightAssociative: false},
	token.Gt:       {priority: 1, rightAssociative: false},
	token.GtE:      {priority: 1, rightAssociative: false},
	token.Lt:       {priority: 1, rightAssociative: false},
	token.LtE:      {priority: 1, rightAssociative: false},
}

// evaluateOperator executes an operator.
func evaluateOperator(operator token.Type, a, b any) (any, error) {
	firstInt, firstIsInt := a.(int64)
	secondInt, secondIsInt := b.(int64)
	firstByte, firstIsByte := a.([]byte)
	secondByte, secondIsByte := b.([]byte)

	switch {
	case firstIsInt && secondIsInt:
		return evaluateOperatorIntInt(operator, firstInt, secondInt)

	case firstIsByte && secondIsInt:
		return evaluateOperatorByteInt(operator, firstByte, secondInt)

	case firstIsByte && secondIsByte:
		return evaluateOperatorByteByte(operator, firstByte, secondByte)

	default:
		return 0, fmt.Errorf("unsupported operator argument type combination of %T and %T",
			a, b)
	}
}

// evaluateOperatorIntInt executes an operator for a and b of type int64.
func evaluateOperatorIntInt(operator token.Type, a, b int64) (any, error) {
	switch operator {
	case token.Plus:
		return a + b, nil
	case token.Minus:
		return a - b, nil
	case token.Asterisk:
		return a * b, nil
	case token.Percent:
		if b == 0 {
			return 0, errDivisionByZero
		}
		return a % b, nil
	case token.Slash:
		if b == 0 {
			return 0, errDivisionByZero
		}
		return a / b, nil
	case token.Caret:
		return int64(math.Pow(float64(a), float64(b))), nil
	case token.Equals:
		return a == b, nil
	case token.Lt:
		return a < b, nil
	case token.LtE:
		return a <= b, nil
	case token.Gt:
		return a > b, nil
	case token.GtE:
		return a >= b, nil
	default:
		return 0, fmt.Errorf("unsupported operator %d for arguments of type int64", operator)
	}
}

// evaluateOperatorByteInt executes an operator for type a of []byte and type b of int64.
func evaluateOperatorByteInt(operator token.Type, a []byte, b int64) (any, error) {
	switch operator {
	case token.Plus:
		for i := range a {
			a[i] += byte(b)
		}
	case token.Minus:
		for i := range a {
			a[i] -= byte(b)
		}
	case token.Asterisk:
		for i := range a {
			a[i] *= byte(b)
		}
	case token.Percent:
		for i := range a {
			a[i] %= byte(b)
		}
	case token.Slash:
		for i := range a {
			if b == 0 {
				return nil, errDivisionByZero
			}
			a[i] /= byte(b)
		}
	case token.Caret:
		for i := range a {
			a[i] = byte(math.Pow(float64(a[i]), float64(b)))
		}
	default:
		return nil, fmt.Errorf("unsupported operator %d for arguments of type []byte and int64", operator)
	}

	return a, nil
}

// evaluateOperatorByteByte executes an operator for a and b of type []byte.
func evaluateOperatorByteByte(operator token.Type, a, b []byte) (any, error) {
	if len(b) == 0 {
		return a, nil
	}

	var operate func(i, j int, a, b []byte) error

	switch operator {
	case token.Plus:
		operate = func(i, j int, a, b []byte) error {
			a[i] += b[j]
			return nil
		}
	case token.Minus:
		operate = func(i, j int, a, b []byte) error {
			a[i] -= b[j]
			return nil
		}
	case token.Asterisk:
		operate = func(i, j int, a, b []byte) error {
			a[i] *= b[j]
			return nil
		}
	case token.Percent:
		operate = func(i, j int, a, b []byte) error {
			a[i] %= b[j]
			return nil
		}
	case token.Slash:
		operate = func(i, j int, a, b []byte) error {
			if b[j] == 0 {
				return errDivisionByZero
			}
			a[i] /= b[j]
			return nil
		}
	case token.Caret:
		operate = func(i, j int, a, b []byte) error {
			a[i] = byte(math.Pow(float64(a[i]), float64(b[j])))
			return nil
		}
	default:
		return nil, fmt.Errorf("unsupported operator %d for arguments of type []byte and []byte", operator)
	}

	j := 0
	for i := range a {
		if err := operate(i, j, a, b); err != nil {
			return nil, err
		}

		j++
		if j >= len(b) {
			j = 0
		}
	}

	return a, nil
}
