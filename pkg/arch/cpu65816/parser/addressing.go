// Package parser implements the architecture specific parser functionality.
package parser

import (
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/arch/cpu/cpu65816"
)

type addressingSize int

const (
	addressingDefault    addressingSize = iota
	addressingAbsolute                  // a: prefix
	addressingDirectPage                // z: prefix
	addressingLong                      // f: prefix
)

const (
	AbsoluteDirectPageAddressing = cpu65816.AbsoluteAddressing | cpu65816.DirectPageAddressing
	XAddressing                  = cpu65816.AbsoluteIndexedXAddressing | cpu65816.DirectPageIndexedXAddressing
	YAddressing                  = cpu65816.AbsoluteIndexedYAddressing | cpu65816.DirectPageIndexedYAddressing
)

// parseAddressSize returns the addressing mode used for an instruction based on the following
// tokens.
func parseAddressSize(parser arch.Parser, ins *cpu65816.Instruction) (addressingSize, error) {
	tok := parser.NextToken(0)
	if tok.Type != token.Identifier && tok.Type != token.EOL {
		return addressingDefault, nil
	}

	accumulatorAddressing := ins.HasAddressing(cpu65816.AccumulatorAddressing)
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

func extendedAddressingParam(ins *instruction, indirectAccess bool) ([]cpu65816.AddressingMode, error) {
	absolute, directPage := addressingSizeAvailability(ins.addressingSize)

	switch ins.arg2.Value {
	case "x", "X":
		return indexedXAddressings(absolute, directPage, indirectAccess), nil
	case "y", "Y":
		return indexedYAddressings(absolute, directPage, indirectAccess), nil
	case "s", "S":
		return []cpu65816.AddressingMode{cpu65816.StackRelativeAddressing}, nil
	default:
		return nil, fmt.Errorf("invalid second argument '%s'", ins.arg2.Value)
	}
}

func addressingSizeAvailability(size addressingSize) (absolute, directPage bool) {
	switch size {
	case addressingDefault:
		absolute = true
		directPage = true
	case addressingAbsolute:
		absolute = true
	case addressingDirectPage:
		directPage = true
	case addressingLong:
	}
	return absolute, directPage
}

func indexedXAddressings(absolute, directPage, indirect bool) []cpu65816.AddressingMode {
	var addressings []cpu65816.AddressingMode
	if indirect {
		if absolute {
			addressings = append(addressings, cpu65816.AbsoluteIndexedXIndirectAddressing)
		}
		if directPage {
			addressings = append(addressings, cpu65816.DirectPageIndexedXIndirectAddressing)
		}
		return addressings
	}
	if absolute {
		addressings = append(addressings, cpu65816.AbsoluteIndexedXAddressing)
	}
	if directPage {
		addressings = append(addressings, cpu65816.DirectPageIndexedXAddressing)
	}
	return addressings
}

func indexedYAddressings(absolute, directPage, indirect bool) []cpu65816.AddressingMode {
	if indirect {
		return []cpu65816.AddressingMode{cpu65816.DirectPageIndirectIndexedYAddressing}
	}
	var addressings []cpu65816.AddressingMode
	if absolute {
		addressings = append(addressings, cpu65816.AbsoluteIndexedYAddressing)
	}
	if directPage {
		addressings = append(addressings, cpu65816.DirectPageIndexedYAddressing)
	}
	return addressings
}
