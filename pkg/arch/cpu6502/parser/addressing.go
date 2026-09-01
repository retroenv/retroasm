// Package parser implements the architecture specific parser functionality.
package parser

import (
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/arch/cpu/cpu6502"
)

type addressingSize int

const (
	addressingDefault addressingSize = iota
	addressingAbsolute
	addressingZeroPage
)

const (
	AbsoluteZeroPageAddressing = cpu6502.AbsoluteAddressing | cpu6502.ZeroPageAddressing
	XAddressing                = cpu6502.AbsoluteXAddressing | cpu6502.ZeroPageXAddressing
	YAddressing                = cpu6502.AbsoluteYAddressing | cpu6502.ZeroPageYAddressing
)

// parseAddressSize returns the addressing mode used for an instruction based on the following
// tokens.
func parseAddressSize(parser arch.Parser, ins *cpu6502.Instruction) (addressingSize, error) {
	tok := parser.NextToken(0)
	if tok.Type != token.Identifier && tok.Type != token.EOL {
		return addressingDefault, nil
	}

	accumulatorAddressing := ins.HasAddressing(cpu6502.AccumulatorAddressing)
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

func extendedAddressingParam(ins *instruction, indirectAccess bool) ([]cpu6502.AddressingMode, error) {
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

	var addressings []cpu6502.AddressingMode

	switch ins.arg2.Value {
	case "x", "X":
		if indirectAccess {
			return []cpu6502.AddressingMode{cpu6502.IndirectXAddressing}, nil
		}

		if absolute {
			addressings = append(addressings, cpu6502.AbsoluteXAddressing)
		}
		if zeropage {
			addressings = append(addressings, cpu6502.ZeroPageXAddressing)
		}

	case "y", "Y":
		if indirectAccess {
			return []cpu6502.AddressingMode{cpu6502.IndirectYAddressing}, nil
		}

		if absolute {
			addressings = append(addressings, cpu6502.AbsoluteYAddressing)
		}
		if zeropage {
			addressings = append(addressings, cpu6502.ZeroPageYAddressing)
		}

	default:
		return nil, fmt.Errorf("invalid second argument '%s'", ins.arg2.Value)
	}

	return addressings, nil
}
