// Package token contains the tokens supported by the lexer.
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

// Token defines a token with position in the stream, its type and a optional value.
type Token struct {
	Position Position
	Type     Type
	Value    string
}
