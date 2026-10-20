package arch

import (
	"errors"

	"github.com/retroenv/retroasm/pkg/lexer/token"
)

// OperandSymbolToken joins a dot and identifier at the given token offset.
// It uses the local label scope when the active mode supplies one.
func OperandSymbolToken(parser Parser, offset int) (token.Token, error) {
	current := parser.NextToken(offset)
	if current.Type != token.Dot {
		return current, nil
	}
	identifier := parser.NextToken(offset + 1)
	if identifier.Type != token.Identifier {
		return token.Token{}, errors.New("expected identifier after '.' in operand")
	}
	name := parser.ResolveDotLocalLabel(identifier.Value)
	if name == "" {
		name = "." + identifier.Value
	}
	identifier.Value = name
	parser.AdvanceReadPosition(1)
	return identifier, nil
}
