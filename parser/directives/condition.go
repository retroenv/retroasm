package directives

import (
	"fmt"

	"github.com/retroenv/assembler/expression"
	"github.com/retroenv/assembler/lexer/token"
	"github.com/retroenv/assembler/parser/ast"
)

// If ...
func If(p Parser) (ast.Node, error) {
	if p.NextToken(2).Type.IsTerminator() {
		return nil, errMissingParameter
	}

	cond := &ast.If{}

	p.AdvanceReadPosition(1)
	tokens, err := readDataTokens(p, false)
	if err != nil {
		return nil, fmt.Errorf("reading data tokens: %w", err)
	}
	cond.Condition = expression.New(tokens...)
	return cond, nil
}

// Ifdef ...
func Ifdef(p Parser) (ast.Node, error) {
	next := p.NextToken(2)
	if next.Type.IsTerminator() {
		return nil, errMissingParameter
	}
	if next.Type != token.Identifier {
		return nil, fmt.Errorf("unsupported condition type %s", next.Type)
	}

	p.AdvanceReadPosition(2)
	return &ast.Ifdef{
		Identifier: next.Value,
	}, nil
}

// Ifndef ...
func Ifndef(p Parser) (ast.Node, error) {
	next := p.NextToken(2)
	if next.Type.IsTerminator() {
		return nil, errMissingParameter
	}
	if next.Type != token.Identifier {
		return nil, fmt.Errorf("unsupported condition type %s", next.Type)
	}

	p.AdvanceReadPosition(2)
	return &ast.Ifndef{
		Identifier: next.Value,
	}, nil
}

// Else ...
func Else(p Parser) (ast.Node, error) {
	cond := &ast.Else{}

	p.AdvanceReadPosition(2)
	tok := p.NextToken(0)
	if !tok.Type.IsTerminator() {
		return nil, errUnexpectedParameter
	}

	return cond, nil
}

// Elseif ...
func Elseif(p Parser) (ast.Node, error) {
	if p.NextToken(2).Type.IsTerminator() {
		return nil, errMissingParameter
	}

	cond := &ast.ElseIf{}

	p.AdvanceReadPosition(1)
	tokens, err := readDataTokens(p, false)
	if err != nil {
		return nil, fmt.Errorf("reading data tokens: %w", err)
	}
	cond.Condition = expression.New(tokens...)
	return cond, nil
}

// Endif ...
func Endif(p Parser) (ast.Node, error) {
	cond := &ast.Endif{}

	p.AdvanceReadPosition(2)
	tok := p.NextToken(0)
	if !tok.Type.IsTerminator() {
		return nil, errUnexpectedParameter
	}

	return cond, nil
}
