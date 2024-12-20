package directives

import (
	"fmt"

	"github.com/retroenv/retroasm/arch"
	"github.com/retroenv/retroasm/expression"
	"github.com/retroenv/retroasm/parser/ast"
)

// Addr ...
func Addr(p arch.Parser) (ast.Node, error) {
	addr, err := createAddressData(p)
	if err != nil {
		return nil, err
	}
	addr.ReferenceType = ast.FullAddress
	return addr, nil
}

// AddrHigh ...
func AddrHigh(p arch.Parser) (ast.Node, error) {
	addr, err := createAddressData(p)
	if err != nil {
		return nil, err
	}
	addr.ReferenceType = ast.HighAddressByte
	return addr, nil
}

// AddrLow ...
func AddrLow(p arch.Parser) (ast.Node, error) {
	addr, err := createAddressData(p)
	if err != nil {
		return nil, err
	}
	addr.ReferenceType = ast.LowAddressByte
	return addr, nil
}

func createAddressData(p arch.Parser) (ast.Data, error) {
	if p.NextToken(2).Type.IsTerminator() {
		return ast.Data{}, errMissingParameter
	}

	data := ast.NewData(ast.AddressType, p.AddressWidth()/8)

	p.AdvanceReadPosition(1)
	tokens, err := readDataTokens(p, false)
	if err != nil {
		return ast.Data{}, fmt.Errorf("reading data tokens: %w", err)
	}
	data.Values = expression.New(tokens...)

	return data, nil
}
