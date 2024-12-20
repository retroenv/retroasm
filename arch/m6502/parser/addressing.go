// Package parser implements the architecture specific parser functionality.
package parser

import (
	"fmt"

	"github.com/retroenv/retroasm/arch"
	"github.com/retroenv/retroasm/lexer/token"
	"github.com/retroenv/retrogolib/arch/cpu/m6502"
)

type addressingSize int

const (
	addressingDefault addressingSize = iota
	addressingAbsolute
	addressingZeroPage
)

const (
	XAddressing = m6502.AbsoluteXAddressing | m6502.ZeroPageXAddressing
	YAddressing = m6502.AbsoluteYAddressing | m6502.ZeroPageYAddressing
)

// parseAddressSize returns the addressing mode used for an instruction based on the following
// tokens.
func parseAddressSize(parser arch.Parser, ins *m6502.Instruction) (addressingSize, error) {
	tok := parser.NextToken(0)
	if tok.Type != token.Identifier && tok.Type != token.EOL {
		return addressingDefault, nil
	}

	accumulatorAddressing := ins.HasAddressing(m6502.AccumulatorAddressing)
	next1 := parser.NextToken(1)

	if accumulatorAddressing && (tok.Type == token.EOL || next1.Type != token.Colon) {
		return addressingDefault, nil
	}

	var addrSize addressingSize
	switch tok.Value {
	case "a", "A":
		addrSize = addressingAbsolute
	case "z", "Z":
		addrSize = addressingZeroPage
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

func extendedAddressingParam(ins *instruction, indirectAccess bool) ([]m6502.AddressingMode, error) {
	var absolute, zeropage bool
	switch ins.addressingSize {
	case addressingDefault:
		absolute = true
		zeropage = true
	case addressingAbsolute:
		absolute = true
	case addressingZeroPage:
		zeropage = true
	}

	var addressings []m6502.AddressingMode

	switch ins.arg2.Value {
	case "x", "X":
		if indirectAccess {
			return []m6502.AddressingMode{m6502.IndirectXAddressing}, nil
		}

		if absolute {
			addressings = append(addressings, m6502.AbsoluteXAddressing)
		}
		if zeropage {
			addressings = append(addressings, m6502.ZeroPageXAddressing)
		}

	case "y", "Y":
		if indirectAccess {
			return []m6502.AddressingMode{m6502.IndirectYAddressing}, nil
		}

		if absolute {
			addressings = append(addressings, m6502.AbsoluteYAddressing)
		}
		if zeropage {
			addressings = append(addressings, m6502.ZeroPageYAddressing)
		}

	default:
		return nil, fmt.Errorf("invalid second argument '%s'", ins.arg2.Value)
	}

	return addressings, nil
}
