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
		return ast.NewBase(addressTokens), nil
	}

	p.AdvanceReadPosition(-1)
	tokens, err := readDataTokens(p, false)
	if err != nil {
		return nil, fmt.Errorf("reading base data tokens: %w", err)
	}

	data := ast.NewData(ast.DataType, 1)
	data.Fill = true
	data.Size.AddTokens(addressTokens...)
	data.Values = expression.New(tokens...)
	return addSizeProgramCounterReference(data)
}
