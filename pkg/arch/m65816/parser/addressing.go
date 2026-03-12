// Package parser implements the architecture specific parser functionality.
package parser

import (
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/arch/cpu/m65816"
)

type addressingSize int

const (
	addressingDefault    addressingSize = iota
	addressingAbsolute                  // a: prefix
	addressingDirectPage                // z: prefix
	addressingLong                      // f: prefix
)

const (
	AbsoluteDirectPageAddressing = m65816.AbsoluteAddressing | m65816.DirectPageAddressing
	XAddressing                  = m65816.AbsoluteIndexedXAddressing | m65816.DirectPageIndexedXAddressing
	YAddressing                  = m65816.AbsoluteIndexedYAddressing | m65816.DirectPageIndexedYAddressing
)

// parseAddressSize returns the addressing mode used for an instruction based on the following
// tokens.
func parseAddressSize(parser arch.Parser, ins *m65816.Instruction) (addressingSize, error) {
	tok := parser.NextToken(0)
	if tok.Type != token.Identifier && tok.Type != token.EOL {
		return addressingDefault, nil
	}

	accumulatorAddressing := ins.HasAddressing(m65816.AccumulatorAddressing)
	next1 := parser.NextToken(1)

	if accumulatorAddressing && (tok.Type == token.EOL || next1.Type != token.Colon) {
		return addressingDefault, nil
	}

	var addrSize addressingSize
	switch tok.Value {
	case "a", "A":
		addrSize = addressingAbsolute
	case "z", "Z":
		addrSize = addressingDirectPage
	case "f", "F":
		addrSize = addressingLong
	default:
		return addressingDefault, nil
	}

	switch next1.Type {
	case token.EOF, token.EOL:
		return addressingDefault, nil

	case token.Colon:
		parser.AdvanceReadPosition(2)
		return addrSize, nil

	default:
		return addressingDefault, fmt.Errorf("invalid token type %s after addressing token", tok.Type)
	}
}

func extendedAddressingParam(ins *instruction, indirectAccess bool) ([]m65816.AddressingMode, error) {
	var absolute, directPage bool
	switch ins.addressingSize {
	case addressingDefault:
		absolute = true
		directPage = true
	case addressingAbsolute:
		absolute = true
	case addressingDirectPage:
		directPage = true
	case addressingLong:
	}

	secondArg := ins.arg2.Value

	switch {
	case secondArg == "x" || secondArg == "X":
		if indirectAccess {
			return []m65816.AddressingMode{m65816.DirectPageIndexedXIndirectAddressing}, nil
		}

		var addressings []m65816.AddressingMode
		if absolute {
			addressings = append(addressings, m65816.AbsoluteIndexedXAddressing)
		}
		if directPage {
			addressings = append(addressings, m65816.DirectPageIndexedXAddressing)
		}
		return addressings, nil

	case secondArg == "y" || secondArg == "Y":
		if indirectAccess {
			return []m65816.AddressingMode{m65816.DirectPageIndirectIndexedYAddressing}, nil
		}

		var addressings []m65816.AddressingMode
		if absolute {
			addressings = append(addressings, m65816.AbsoluteIndexedYAddressing)
		}
		if directPage {
			addressings = append(addressings, m65816.DirectPageIndexedYAddressing)
		}
		return addressings, nil

	case secondArg == "s" || secondArg == "S":
		return []m65816.AddressingMode{m65816.StackRelativeAddressing}, nil

	default:
		return nil, fmt.Errorf("invalid second argument '%s'", ins.arg2.Value)
	}
}
