package expression

import (
	"strings"

	"github.com/retroenv/retroasm/pkg/lexer/token"
)

const (
	keywordOperatorShiftLeft  = "SHL"
	keywordOperatorShiftRight = "SHR"
	keywordOperatorAnd        = "AND"
	keywordOperatorOr         = "OR"
	keywordOperatorXor        = "XOR"
)

// keywordOperators maps assembly keyword operator names to token types.
// These allow expressions to use keyword-style operators (e.g., SHL, AND)
// in addition to symbolic operators.
var keywordOperators = map[string]token.Type{
	keywordOperatorShiftLeft:  token.ShiftLeft,
	keywordOperatorShiftRight: token.ShiftRight,
	keywordOperatorAnd:        token.Ampersand,
	keywordOperatorOr:         token.Pipe,
	keywordOperatorXor:        token.BitwiseXor,
}

// resolveKeywordOperator converts keyword operator identifiers (SHL, SHR, AND, OR, XOR)
// to their corresponding operator token types for expression evaluation.
func resolveKeywordOperator(tok token.Token) token.Token {
	if tok.Type != token.Identifier {
		return tok
	}
	if opType, ok := keywordOperators[strings.ToUpper(tok.Value)]; ok {
		tok.Type = opType
	}
	return tok
}
