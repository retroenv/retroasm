package parser

import (
	"fmt"

	"github.com/retroenv/retroasm/lexer/token"
	. "github.com/retroenv/retrogolib/addressing"
	"github.com/retroenv/retrogolib/arch/cpu/m6502"
)

type addressingSize int

const (
	addressingDefault addressingSize = iota
	addressingAbsolute
	addressingZeroPage
)

// parseAddressSize returns the addressing mode used for an instruction based on the following
// tokens.
func (p *Parser) parseAddressSize(ins *m6502.Instruction) (addressingSize, error) {
	tok := p.NextToken(0)
	if tok.Type != token.Identifier && tok.Type != token.EOL {
		return addressingDefault, nil
	}

	accumulatorAddressing := ins.HasAddressing(AccumulatorAddressing)
	next1 := p.NextToken(1)

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
		p.readPosition += 2
		return addrSize, nil

	default:
		return addressingDefault, fmt.Errorf("invalid token type %s after addressing token", tok.Type)
	}
}

func extendedAddressingParam(ins *instruction, indirectAccess bool) ([]Mode, error) {
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

	var addressings []Mode

	switch ins.arg2.Value {
	case "x", "X":
		if indirectAccess {
			return []Mode{IndirectXAddressing}, nil
		}

		if absolute {
			addressings = append(addressings, AbsoluteXAddressing)
		}
		if zeropage {
			addressings = append(addressings, ZeroPageXAddressing)
		}

	case "y", "Y":
		if indirectAccess {
			return []Mode{IndirectYAddressing}, nil
		}

		if absolute {
			addressings = append(addressings, AbsoluteYAddressing)
		}
		if zeropage {
			addressings = append(addressings, ZeroPageYAddressing)
		}

	default:
		return nil, fmt.Errorf("invalid second argument '%s'", ins.arg2.Value)
	}

	return addressings, nil
}
