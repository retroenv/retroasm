package directives

import (
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// Scope parses a .scope directive for defining a named scope without an entry label.
func Scope(p arch.Parser) (ast.Node, error) {
	next := p.NextToken(2)
	if next.Type.IsTerminator() {
		// Anonymous scope (no name).
		p.AdvanceReadPosition(1)
		return ast.NewScope(""), nil
	}

	p.AdvanceReadPosition(2)
	return ast.NewScope(next.Value), nil
}

// EndScope parses a .endscope directive for ending a scope definition.
func EndScope(p arch.Parser) (ast.Node, error) {
	p.AdvanceReadPosition(1)
	return ast.NewScopeEnd(), nil
}

// Asciiz parses a .asciiz directive for null-terminated string data.
func Asciiz(p arch.Parser) (ast.Node, error) {
	if p.NextToken(2).Type.IsTerminator() {
		return nil, errMissingParameter
	}

	p.AdvanceReadPosition(1)
	tokens, err := readDataTokens(p, false)
	if err != nil {
		return nil, fmt.Errorf("reading asciiz data tokens: %w", err)
	}

	// Append a null terminator byte.
	nullTok := token.Token{
		Type:  token.Number,
		Value: "0",
	}
	tokens = append(tokens, nullTok)

	data := ast.NewData(ast.DataType, 1)
	data.Values = expression.New(tokens...)
	return data, nil
}

// FarAddr parses a .faraddr directive for 24-bit (3-byte) address data.
func FarAddr(p arch.Parser) (ast.Node, error) {
	if p.NextToken(2).Type.IsTerminator() {
		return nil, errMissingParameter
	}

	data := ast.NewData(ast.AddressType, 3)
	data.ReferenceType = ast.FullAddress

	p.AdvanceReadPosition(1)
	tokens, err := readDataTokens(p, false)
	if err != nil {
		return nil, fmt.Errorf("reading faraddr tokens: %w", err)
	}
	data.Values = expression.New(tokens...)

	return data, nil
}

// BankBytes parses a .bankbytes directive for emitting the bank byte (bits 16-23) of addresses.
func BankBytes(p arch.Parser) (ast.Node, error) {
	return createAddressData(p, ast.BankAddressByte)
}

// Warning parses a .warning directive for emitting an assembler warning message.
func Warning(p arch.Parser) (ast.Node, error) {
	msg := p.NextToken(2)
	if msg.Type.IsTerminator() {
		return nil, errMissingParameter
	}

	p.AdvanceReadPosition(2)
	return ast.NewError(strings.Trim(msg.Value, "\"'")), nil
}

// Out parses a .out directive for printing a message during assembly.
//
//nolint:nilnil // directive is intentionally ignored after consuming tokens
func Out(p arch.Parser) (ast.Node, error) {
	for {
		p.AdvanceReadPosition(1)
		if p.NextToken(0).Type.IsTerminator() {
			return nil, nil
		}
	}
}
