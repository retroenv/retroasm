package directives

import (
	"fmt"

	"github.com/retroenv/assembler/expression"
	"github.com/retroenv/assembler/parser/ast"
)

// Base ...
func Base(p Parser) (ast.Node, error) {
	p.AdvanceReadPosition(1)
	addressTokens, err := readDataTokens(p, true)
	if err != nil {
		return nil, fmt.Errorf("reading base size tokens: %w", err)
	}

	tok := p.NextToken(1)
	if tok.Type.IsTerminator() {
		return &ast.Base{
			Address: expression.New(addressTokens...),
		}, nil
	}

	p.AdvanceReadPosition(-1)
	tokens, err := readDataTokens(p, false)
	if err != nil {
		return nil, fmt.Errorf("reading base data tokens: %w", err)
	}

	data := &ast.Data{
		Type:   "data",
		Width:  1,
		Fill:   true,
		Size:   expression.New(addressTokens...),
		Values: expression.New(tokens...),
	}
	return addSizeProgramCounterReference(data)
}
