package directives

import (
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// Enum parses a .enum directive for starting an enumeration block.
func Enum(p arch.Parser) (ast.Node, error) {
	p.AdvanceReadPosition(1)
	addressTokens, err := readDataTokens(p, true)
	if err != nil {
		return nil, fmt.Errorf("reading enum address tokens: %w", err)
	}

	return ast.NewEnum(addressTokens), nil
}

// Ende parses a .ende directive for ending an enumeration block.
func Ende(p arch.Parser) (ast.Node, error) {
	p.AdvanceReadPosition(1)
	return ast.NewEnumEnd(), nil
}
