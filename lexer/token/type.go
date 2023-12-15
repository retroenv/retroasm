package token

import "fmt"

// Type defines the token type.
type Type int

var operators = map[Type]struct{}{
	Plus:     {},
	Minus:    {},
	Asterisk: {},
	Percent:  {},
	Slash:    {},
	Caret:    {},
	Equals:   {},
	Lt:       {},
	LtE:      {},
	Gt:       {},
	GtE:      {},
}

// NewType creates a new token type from the given rune.
func NewType(r rune) (Type, error) {
	t, ok := toToken[r]
	if !ok {
		return Illegal, fmt.Errorf("unknown rune %c", r)
	}
	return t, nil
}

// String returns the string representation of the identifier.
func (t Type) String() string {
	s, ok := toString[t]
	if !ok {
		panic(fmt.Sprintf("unknown type %d", t))
	}
	return s
}

// IsTerminator returns whether the token terminates the current usable nodes in the line.
func (t Type) IsTerminator() bool {
	return t == EOF || t == EOL || t == Comment
}

// IsOperator returns whether the token is an operator for a math operation.
func (t Type) IsOperator() bool {
	_, isOperator := operators[t]
	return isOperator
}
