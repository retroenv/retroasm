package directives

import (
	"fmt"

	"github.com/retroenv/retroasm/arch"
	"github.com/retroenv/retroasm/parser/ast"
)

// Enum ...
func Enum(p arch.Parser) (ast.Node, error) {
	p.AdvanceReadPosition(1)
	addressTokens, err := readDataTokens(p, true)
	if err != nil {
		return nil, fmt.Errorf("reading enum address tokens: %w", err)
	}

	return ast.NewEnum(addressTokens), nil
}

// Ende ...
func Ende(p arch.Parser) (ast.Node, error) {
	p.AdvanceReadPosition(1)
	return ast.NewEnumEnd(), nil
}
