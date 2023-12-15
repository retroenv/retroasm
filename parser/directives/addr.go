package directives

import (
	"fmt"

	"github.com/retroenv/assembler/expression"
	"github.com/retroenv/assembler/parser/ast"
)

// Addr ...
func Addr(p Parser) (ast.Node, error) {
	addr, err := createAddressData(p)
	if err != nil {
		return nil, err
	}
	addr.ReferenceType = ast.FullAddress
	return addr, nil
}

// AddrHigh ...
func AddrHigh(p Parser) (ast.Node, error) {
	addr, err := createAddressData(p)
	if err != nil {
		return nil, err
	}
	addr.ReferenceType = ast.HighAddressByte
	return addr, nil
}

// AddrLow ...
func AddrLow(p Parser) (ast.Node, error) {
	addr, err := createAddressData(p)
	if err != nil {
		return nil, err
	}
	addr.ReferenceType = ast.LowAddressByte
	return addr, nil
}

func createAddressData(p Parser) (*ast.Data, error) {
	if p.NextToken(2).Type.IsTerminator() {
		return nil, errMissingParameter
	}

	data := &ast.Data{
		Type:  "address",
		Width: p.Arch().AddressWidth / 8,
		Size:  expression.New(),
	}

	p.AdvanceReadPosition(1)
	tokens, err := readDataTokens(p, false)
	if err != nil {
		return nil, fmt.Errorf("reading data tokens: %w", err)
	}
	data.Values = expression.New(tokens...)

	return data, nil
}
