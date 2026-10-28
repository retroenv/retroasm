// Package parser retains typed operands for the supported 8086 subset.
package parser

import (
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	isa "github.com/retroenv/retrogolib/arch/cpu/x86"
)

// BuildInstruction constructs an instruction with operands in Intel order.
func BuildInstruction(name string, operands ...ast.Node) ast.Instruction {
	mode := isa.ImpliedAddressing
	var argument ast.Node
	switch len(operands) {
	case 0:
	case 1:
		mode = isa.ImmediateAddressing
		if _, register := operands[0].(ast.InstructionArgument); register {
			mode = isa.RegisterAddressing
		}

		if name == isa.CallName || name == isa.JmpName || name == isa.JzName || name == isa.JnzName {
			mode = isa.RelativeAddressing
		}

		argument = operands[0]

	default:
		mode = isa.ImmediateAddressing
		if _, register := operands[1].(ast.InstructionArgument); register {
			mode = isa.ModRMRegisterAddressing
		}

		argument = ast.NewInstructionArguments(operands...)
	}

	return ast.NewInstruction(strings.ToUpper(name), int(mode), argument, nil)
}

// ParseIdentifier reads registers, unsigned immediate values, and near labels.
func ParseIdentifier(p arch.Parser, name string) (ast.Node, error) {
	if name == isa.RetName || name == isa.NopName || name == isa.CldName {
		return BuildInstruction(name), nil
	}
	p.AdvanceReadPosition(1)
	first, err := parseOperand(p.NextToken(0))
	if err != nil {
		return nil, err
	}
	if p.NextToken(1).Type != token.Comma {
		return BuildInstruction(name, first), nil
	}
	p.AdvanceReadPosition(2)
	second, err := parseOperand(p.NextToken(0))
	if err != nil {
		return nil, err
	}
	return BuildInstruction(name, first, second), nil
}

func parseOperand(value token.Token) (ast.Node, error) {
	switch value.Type {
	case token.Identifier:
		for register := isa.RegAL; register <= isa.RegDS; register++ {
			if strings.EqualFold(value.Value, register.String()) {
				return ast.NewInstructionArgument(register), nil
			}
		}

		return ast.NewLabel(value.Value), nil

	case token.Number:
		parsed, err := number.Parse(value.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid 8086 immediate %q: %w", value.Value, err)
		}
		return ast.NewNumber(parsed), nil

	default:
		return nil, fmt.Errorf("unsupported 8086 operand %q", value.Value)
	}
}
