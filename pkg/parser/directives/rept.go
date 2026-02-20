package directives

import (
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// Rept parses a .rept directive for defining a repeat block.
func Rept(p arch.Parser) (ast.Node, error) {
	p.AdvanceReadPosition(1)
	countTokens, err := readDataTokens(p, true)
	if err != nil {
		return nil, fmt.Errorf("reading rept count tokens: %w", err)
	}

	return ast.NewRept(countTokens), nil
}

// Endr parses a .endr directive for ending a repeat block.
func Endr(p arch.Parser) (ast.Node, error) {
	p.AdvanceReadPosition(1)
	return ast.NewEndr(), nil
}
