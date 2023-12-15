package directives

import (
	"fmt"

	"github.com/retroenv/assembler/lexer/token"
	"github.com/retroenv/assembler/number"
	"github.com/retroenv/assembler/parser/ast"
)

// Res ...
func Res(p Parser) (ast.Node, error) {
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

	return &ast.Variable{
		Size: int(i),
	}, nil
}
