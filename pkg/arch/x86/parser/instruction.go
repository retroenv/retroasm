// Package parser retains typed operands for the supported 8086 subset.
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

// BuildInstruction constructs an instruction with operands in Intel order.
func BuildInstruction(name string, operands ...ast.Node) ast.Instruction {
	normalized := make([]ast.Node, len(operands))
	for index, operand := range operands {
		normalized[index] = normalizeRegisterOperand(operand)
	}

	operands = normalized

	mode := isa.ImpliedAddressing
	var argument ast.Node
	switch len(operands) {
	case 0:
	case 1:
		mode = isa.ImmediateAddressing
		if argument, typed := operands[0].(ast.InstructionArgument); typed {
			if _, register := argument.Value.(RegisterOperand); register {
				mode = isa.RegisterAddressing
			}
		}

		if mode == isa.ImmediateAddressing &&
			(name == isa.CallName || name == isa.JmpName || isa.ConditionalJumpInstructions.Contains(name)) {

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

	for _, operand := range operands {
		if typed, ok := operand.(ast.InstructionArgument); ok {
			if _, memory := typed.Value.(MemoryOperand); memory {
				mode = isa.ModRMMemoryAddressing
			}
		}
	}

	return ast.NewInstruction(strings.ToUpper(name), int(mode), argument, nil)
}

// ParseIdentifier reads the retained 8086 operand syntax.
func ParseIdentifier(p arch.Parser, name string) (ast.Node, error) {
	if impliedInstruction(name) && (name != isa.RetName || p.NextToken(1).Type == token.EOL ||
		p.NextToken(1).Type == token.EOF || p.NextToken(1).Type == token.Comment) {

		return BuildInstruction(name), nil
	}
	p.AdvanceReadPosition(1)
	if name == isa.RepName || name == isa.RepzName || name == isa.RepnzName {
		value := p.NextToken(0)
		if value.Type != token.Identifier {
			return nil, errors.New("repeat prefix needs a string instruction")
		}
		return BuildInstruction(name, ast.NewInstructionArgument(RepeatedString(strings.ToLower(value.Value)))), nil
	}
	first, err := parseOperand(p)
	if err != nil {
		return nil, err
	}
	if p.NextToken(1).Type != token.Comma {
		return BuildInstruction(name, first), nil
	}
	p.AdvanceReadPosition(2)
	second, err := parseOperand(p)
	if err != nil {
		return nil, err
	}
	return BuildInstruction(name, first, second), nil
}

func impliedInstruction(name string) bool {
	switch name {
	case isa.RetName, isa.NopName, isa.CldName, isa.StdName, isa.ClcName, isa.StcName, isa.CmcName,
		isa.CbwName, isa.CwdName, isa.IretName, isa.MovsbName, isa.MovswName, isa.StosbName, isa.StoswName,
		isa.LodsbName, isa.LodswName, isa.CmpsbName, isa.CmpswName, isa.ScasbName, isa.ScaswName:
		return true
	default:
		return false
	}
}

func parseOperand(p arch.Parser) (ast.Node, error) {
	value := p.NextToken(0)
	if value.Type == token.LeftBracket || value.Type == token.Identifier &&
		(strings.EqualFold(value.Value, "byte") || strings.EqualFold(value.Value, "word")) {

		return parseMemory(p)
	}
	if value.Type == token.Minus {
		p.AdvanceReadPosition(1)
		value = p.NextToken(0)
		parsed, err := number.Parse(value.Value)
		if value.Type != token.Number || err != nil || parsed > 32768 {
			return nil, fmt.Errorf("invalid signed 8086 immediate %q", value.Value)
		}
		return ast.NewInstructionArgument(SignedImmediate(-int64(parsed))), nil
	}
	switch value.Type {
	case token.Identifier:
		if register, valid := namedRegister(value.Value); valid {
			return ast.NewInstructionArgument(RegisterOperand{Register: register}), nil
		}
		return ast.NewLabel(p.ScopeLocalLabel(value.Value)), nil

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

func namedRegister(name string) (isa.RegisterParam, bool) {
	for register := isa.RegAL; register <= isa.RegDS; register++ {
		if strings.EqualFold(name, register.String()) {
			return register, true
		}
	}

	return 0, false
}
