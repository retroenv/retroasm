package directives

import (
	"fmt"

	"github.com/retroenv/assembler/parser/ast"
)

// Rept ...
func Rept(p Parser) (ast.Node, error) {
	p.AdvanceReadPosition(1)
	countTokens, err := readDataTokens(p, true)
	if err != nil {
		return nil, fmt.Errorf("reading rept count tokens: %w", err)
	}

	return ast.NewRept(countTokens), nil
}

// Endr ...
func Endr(p Parser) (ast.Node, error) {
	p.AdvanceReadPosition(1)
	return ast.NewEndr(), nil
}
