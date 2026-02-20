package directives

import (
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// If parses an .if conditional assembly directive.
func If(p arch.Parser) (ast.Node, error) {
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

// Ifdef parses an .ifdef directive.
func Ifdef(p arch.Parser) (ast.Node, error) {
	return parseSymbolExistsDirective(p, func(s string) ast.Node { return ast.NewIfdef(s) })
}

// Ifndef parses an .ifndef directive.
func Ifndef(p arch.Parser) (ast.Node, error) {
	return parseSymbolExistsDirective(p, func(s string) ast.Node { return ast.NewIfndef(s) })
}

// Else parses an .else directive.
func Else(p arch.Parser) (ast.Node, error) {
	return parseTerminatingDirective(p, ast.NewElse())
}

// Elseif parses an .elseif directive.
func Elseif(p arch.Parser) (ast.Node, error) {
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

// Endif parses an .endif directive.
func Endif(p arch.Parser) (ast.Node, error) {
	return parseTerminatingDirective(p, ast.NewEndif())
}

func parseSymbolExistsDirective(p arch.Parser, constructor func(string) ast.Node) (ast.Node, error) {
	next := p.NextToken(2)
	if next.Type.IsTerminator() {
		return nil, errMissingParameter
	}
	if next.Type != token.Identifier {
		return nil, fmt.Errorf("unsupported condition type %s", next.Type)
	}

	p.AdvanceReadPosition(2)
	return constructor(next.Value), nil
}

// parseTerminatingDirective handles directives that take no parameters
// and expect the line to end immediately after.
func parseTerminatingDirective(p arch.Parser, node ast.Node) (ast.Node, error) {
	p.AdvanceReadPosition(2)
	tok := p.NextToken(0)
	if !tok.Type.IsTerminator() {
		return nil, errUnexpectedParameter
	}

	return node, nil
}
