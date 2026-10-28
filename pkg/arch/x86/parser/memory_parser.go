package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	isa "github.com/retroenv/retrogolib/arch/cpu/x86"
)

func parseMemory(p arch.Parser) (ast.Node, error) {
	operand := MemoryOperand{Address: isa.RegMem}
	value := p.NextToken(0)
	if value.Type == token.Identifier {
		operand.Width = 1
		if strings.EqualFold(value.Value, "word") {
			operand.Width = 2
		}

		p.AdvanceReadPosition(1)
		if strings.EqualFold(p.NextToken(0).Value, "ptr") {
			p.AdvanceReadPosition(1)
		}
	}

	if p.NextToken(0).Type != token.LeftBracket {
		return nil, errors.New("8086 memory operand needs brackets")
	}
	p.AdvanceReadPosition(1)
	if p.NextToken(1).Type == token.Colon {
		segment, valid := namedRegister(p.NextToken(0).Value)
		if !valid || !segment.IsSegment() {
			return nil, errors.New("invalid 8086 segment override")
		}
		operand.Segment = segment
		p.AdvanceReadPosition(2)
	}

	registers, err := parseMemoryTerms(p, &operand)
	if err != nil {
		return nil, err
	}
	address, err := memoryAddress(registers)
	if err != nil {
		return nil, err
	}
	operand.Address = address

	return ast.NewInstructionArgument(operand), nil
}

func parseMemoryTerms(p arch.Parser, operand *MemoryOperand) ([]isa.RegisterParam, error) {
	var registers []isa.RegisterParam
	sign := int64(1)
	wantTerm := true
	for {
		value := p.NextToken(0)
		if value.Type == token.RightBracket && !wantTerm {
			return registers, nil
		}
		if wantTerm {
			register, err := parseMemoryTerm(p, operand, value, sign)
			if err != nil {
				return nil, err
			}
			if register != isa.RegMem {
				registers = append(registers, register)
			}

			wantTerm = false
			p.AdvanceReadPosition(1)

			continue
		}
		switch value.Type {
		case token.Plus:
			sign = 1
		case token.Minus:
			sign = -1
		default:
			return nil, errors.New("8086 address needs +, -, or ]")
		}

		wantTerm = true
		p.AdvanceReadPosition(1)
	}
}

func parseMemoryTerm(p arch.Parser, operand *MemoryOperand, value token.Token, sign int64) (isa.RegisterParam, error) {
	switch value.Type {
	case token.Number:
		parsed, err := number.Parse(value.Value)
		if err != nil || parsed > 65535 {
			return 0, fmt.Errorf("invalid 8086 displacement %q", value.Value)
		}
		operand.Displacement += sign * int64(parsed)
		if operand.Displacement < -32768 || operand.Displacement > 65535 {
			return 0, errors.New("8086 displacement exceeds one word")
		}

	case token.Identifier:
		if register, known := namedRegister(value.Value); known {
			if sign < 0 {
				return 0, errors.New("8086 address cannot subtract a register")
			}
			return register, nil
		}
		if sign < 0 || operand.Symbol != "" {
			return 0, errors.New("8086 address permits one added symbol")
		}
		operand.Symbol = p.ScopeLocalLabel(value.Value)

	default:
		return 0, fmt.Errorf("invalid 8086 memory term %q", value.Value)
	}

	return isa.RegMem, nil
}

func memoryAddress(registers []isa.RegisterParam) (isa.RegisterParam, error) {
	if len(registers) == 0 {
		return isa.RegMem, nil
	}
	if len(registers) == 1 {
		switch registers[0] {
		case isa.RegBX:
			return isa.RegBXRef, nil
		case isa.RegBP:
			return isa.RegBPRef, nil
		case isa.RegSI:
			return isa.RegSIRef, nil
		case isa.RegDI:
			return isa.RegDIRef, nil
		}
	}

	if len(registers) == 2 {
		base := registers[0]
		index := registers[1]
		if base == isa.RegSI || base == isa.RegDI {
			base, index = index, base
		}

		switch {
		case base == isa.RegBX && index == isa.RegSI:
			return isa.RegBXSIRef, nil
		case base == isa.RegBX && index == isa.RegDI:
			return isa.RegBXDIRef, nil
		case base == isa.RegBP && index == isa.RegSI:
			return isa.RegBPSIRef, nil
		case base == isa.RegBP && index == isa.RegDI:
			return isa.RegBPDIRef, nil
		}
	}

	return 0, errors.New("invalid 8086 base and index registers")
}
