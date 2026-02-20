package directives

import (
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// Base parses a .org or .base directive for program counter assignment.
func Base(p arch.Parser) (ast.Node, error) {
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
