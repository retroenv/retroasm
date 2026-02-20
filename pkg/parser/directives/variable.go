package directives

import (
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// Res parses a .res directive for reserving space for a variable.
func Res(p arch.Parser) (ast.Node, error) {
	p.AdvanceReadPosition(2)
	next := p.NextToken(0)

	switch next.Type {
	case token.EOF, token.EOL:
		return nil, errMissingParameter
	case token.Number:
		break
	default:
		return nil, fmt.Errorf("unsupported size type %s", next.Type)
	}

	i, err := number.Parse(next.Value)
	if err != nil {
		return nil, fmt.Errorf("parsing number '%s': %w", next.Value, err)
	}

	return ast.NewVariable("", int(i)), nil
}
