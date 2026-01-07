package directives

import (
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// FillValue ...
func FillValue(p arch.Parser) (ast.Node, error) {
	if p.NextToken(2).Type.IsTerminator() {
		return nil, errMissingParameter
	}

	cfg := ast.NewConfiguration(ast.ConfigFillValue)

	p.AdvanceReadPosition(1)
	tokens, err := readDataTokens(p, false)
	if err != nil {
		return nil, fmt.Errorf("reading data tokens: %w", err)
	}
	cfg.Expression = expression.New(tokens...)
	return cfg, nil
}
