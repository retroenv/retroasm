package directives

import (
	"fmt"

	"github.com/retroenv/assembler/expression"
	"github.com/retroenv/assembler/parser/ast"
)

// FillValue ...
func FillValue(p Parser) (ast.Node, error) {
	if p.NextToken(2).Type.IsTerminator() {
		return nil, errMissingParameter
	}

	cfg := &ast.Configuration{
		Item: ast.ConfigFillValue,
	}

	p.AdvanceReadPosition(1)
	tokens, err := readDataTokens(p, false)
	if err != nil {
		return nil, fmt.Errorf("reading data tokens: %w", err)
	}
	cfg.Expression = expression.New(tokens...)
	return cfg, nil
}
