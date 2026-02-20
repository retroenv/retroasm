package directives

import (
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// Addr parses an .addr directive for full address data.
func Addr(p arch.Parser) (ast.Node, error) {
	return createAddressData(p, ast.FullAddress)
}

// AddrHigh parses a .dh directive for high address byte data.
func AddrHigh(p arch.Parser) (ast.Node, error) {
	return createAddressData(p, ast.HighAddressByte)
}

// AddrLow parses a .dl directive for low address byte data.
func AddrLow(p arch.Parser) (ast.Node, error) {
	return createAddressData(p, ast.LowAddressByte)
}

func createAddressData(p arch.Parser, refType ast.ReferenceType) (ast.Data, error) {
	if p.NextToken(2).Type.IsTerminator() {
		return ast.Data{}, errMissingParameter
	}

	data := ast.NewData(ast.AddressType, p.AddressWidth()/8)
	data.ReferenceType = refType

	p.AdvanceReadPosition(1)
	tokens, err := readDataTokens(p, false)
	if err != nil {
		return ast.Data{}, fmt.Errorf("reading data tokens: %w", err)
	}
	data.Values = expression.New(tokens...)

	return data, nil
}
