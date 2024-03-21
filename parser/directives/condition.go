package directives

import (
	"fmt"

	"github.com/retroenv/retroasm/lexer/token"
	"github.com/retroenv/retroasm/parser/ast"
)

// If ...
func If(p Parser) (ast.Node, error) {
	if p.NextToken(2).Type.IsTerminator() {
		return nil, errMissingParameter
	}

	p.AdvanceReadPosition(1)
	tokens, err := readDataTokens(p, false)
	if err != nil {
		return nil, fmt.Errorf("reading data tokens: %w", err)
	}
	return ast.NewIf(tokens), nil
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
	return ast.NewIfdef(next.Value), nil
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
	return ast.NewIfndef(next.Value), nil
}

// Else ...
func Else(p Parser) (ast.Node, error) {
	p.AdvanceReadPosition(2)
	tok := p.NextToken(0)
	if !tok.Type.IsTerminator() {
		return nil, errUnexpectedParameter
	}

	return ast.NewElse(), nil
}

// Elseif ...
func Elseif(p Parser) (ast.Node, error) {
	if p.NextToken(2).Type.IsTerminator() {
		return nil, errMissingParameter
	}

	p.AdvanceReadPosition(1)
	tokens, err := readDataTokens(p, false)
	if err != nil {
		return nil, fmt.Errorf("reading data tokens: %w", err)
	}
	return ast.NewElseIf(tokens), nil
}

// Endif ...
func Endif(p Parser) (ast.Node, error) {
	p.AdvanceReadPosition(2)
	tok := p.NextToken(0)
	if !tok.Type.IsTerminator() {
		return nil, errUnexpectedParameter
	}

	return ast.NewEndif(), nil
}
