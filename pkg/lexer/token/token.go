// Package token defines token types and structures for lexical analysis.
//
// Token represents a single lexical unit with position information and type classification.
// This package supports various token types including identifiers, numbers, operators,
// delimiters, and comments commonly found in assembly language syntax.
//
// # Token Types
//
// The package defines token types for:
//   - Literals: Number, Identifier, Comment
//   - Operators: Plus, Minus, Asterisk, etc.
//   - Delimiters: Parentheses, Brackets, Braces
//   - Control: EOF, EOL, Illegal
//
// # Position Tracking
//
// Each token includes position information (line and column) for accurate
// error reporting and debugging support.
package token

const (
	Illegal Type = iota

	EOF
	EOL

	// Identifiers and literals.
	Number
	Identifier
	Comment

	// Delimiters.
	Dot
	Colon
	Semicolon
	Comma
	Assign
	Plus
	Minus
	Equals
	Lt
	LtE
	Gt
	GtE
	Pipe
	Asterisk
	Percent

	LeftParentheses
	RightParentheses
	LeftBracket
	RightBracket
	LeftBrace
	RightBrace
	Slash
	Caret
)

var toString = map[Type]string{
	EOF:              "EOF",
	EOL:              "EOL",
	Illegal:          "Illegal",
	Number:           "Number",
	Identifier:       "Identifier",
	Comment:          "Comment",
	Dot:              ".",
	Colon:            ":",
	Semicolon:        ";",
	Comma:            ",",
	Assign:           "=",
	Plus:             "+",
	Minus:            "-",
	Equals:           "==",
	Lt:               "<",
	LtE:              "<=",
	Gt:               ">",
	GtE:              ">=",
	Pipe:             "|",
	Asterisk:         "*",
	Percent:          "%",
	LeftParentheses:  "(",
	RightParentheses: ")",
	LeftBracket:      "[",
	RightBracket:     "]",
	LeftBrace:        "{",
	RightBrace:       "}",
	Slash:            "/",
	Caret:            "^",
}

var toToken = map[rune]Type{
	'.': Dot,
	':': Colon,
	';': Semicolon,
	',': Comma,
	'=': Assign,
	'+': Plus,
	'-': Minus,
	'<': Lt,
	'>': Gt,
	'|': Pipe,
	'*': Asterisk,
	'%': Percent,
	'(': LeftParentheses,
	')': RightParentheses,
	'[': LeftBracket,
	']': RightBracket,
	'{': LeftBrace,
	'}': RightBrace,
	'/': Slash,
	'^': Caret,
}

// Token defines a token with position in the stream, its type and an optional value.
type Token struct {
	Position Position
	Type     Type
	Value    string
}
