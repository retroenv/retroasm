package directives

import (
	"fmt"

	"github.com/retroenv/retroasm/parser/ast"
)

// Enum ...
func Enum(p Parser) (ast.Node, error) {
	p.AdvanceReadPosition(1)
	addressTokens, err := readDataTokens(p, true)
	if err != nil {
		return nil, fmt.Errorf("reading enum address tokens: %w", err)
	}

	return ast.NewEnum(addressTokens), nil
}

// Ende ...
func Ende(p Parser) (ast.Node, error) {
	p.AdvanceReadPosition(1)
	return ast.NewEnumEnd(), nil
}
