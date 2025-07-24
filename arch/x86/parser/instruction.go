// Package parser provides x86 instruction parsing functionality.
package parser

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/retroenv/retroasm/arch"
	"github.com/retroenv/retroasm/lexer/token"
	"github.com/retroenv/retroasm/number"
	"github.com/retroenv/retroasm/parser/ast"
)

// x86 addressing modes (avoiding import cycle).
const (
	registerAddressing  = 0
	immediateAddressing = 1
	directAddressing    = 2
)

// x86 registers map.
var x86Registers = map[string]int{
	"AL": 0, "CL": 1, "DL": 2, "BL": 3, "AH": 4, "CH": 5, "DH": 6, "BH": 7,
	"AX": 0, "CX": 1, "DX": 2, "BX": 3, "SP": 4, "BP": 5, "SI": 6, "DI": 7,
	"ES": 0, "CS": 1, "SS": 2, "DS": 3,
}

// ParseIdentifier parses an x86 instruction and returns the corresponding AST node.
func ParseIdentifier(parser arch.Parser, ins interface{}) (ast.Node, error) {
	// Extract name from instruction - this is a simplified approach to avoid import cycles
	name := extractInstructionName(ins)

	// Handle implied instructions (NOP, RET)
	if name == "NOP" || name == "RET" {
		return ast.NewInstruction(name, registerAddressing, nil, nil), nil
	}

	node, err := parseInstruction(parser, name)
	if err != nil {
		return nil, fmt.Errorf("parsing instruction %s: %w", name, err)
	}
	return node, nil
}

func extractInstructionName(ins interface{}) string {
	// Use reflection-like approach to get name
	if insPtr, ok := ins.(interface{ Name() string }); ok {
		return insPtr.Name()
	}
	return "UNKNOWN"
}

func parseInstruction(parser arch.Parser, instructionName string) (ast.Node, error) {
	parser.AdvanceReadPosition(1)

	arg1 := parser.NextToken(0)

	// Handle single register instructions (INC, DEC, PUSH, POP)
	if isRegisterToken(arg1) && isSingleRegisterInstruction(instructionName) {
		regName := strings.ToUpper(arg1.Value)
		if _, ok := x86Registers[regName]; !ok {
			return nil, fmt.Errorf("unknown register: %s", arg1.Value)
		}

		// Create a register node with the register information
		regNode := ast.NewIdentifier(regName)
		return ast.NewInstruction(instructionName, registerAddressing, regNode, nil), nil
	}

	// Check for second argument
	next1 := parser.NextToken(1)
	if next1.Type == token.Comma {
		// Two operand instruction
		parser.AdvanceReadPosition(2)
		arg2 := parser.NextToken(0)
		return parseTwoOperandInstruction(instructionName, arg1, arg2)
	}

	// Single operand instruction
	switch {
	case arg1.Type == token.Number && arg1.Value[0] == '#':
		return parseImmediateInstruction(instructionName, arg1)
	case arg1.Type == token.Number:
		return parseDirectAddressInstruction(instructionName, arg1)
	case arg1.Type == token.Identifier:
		return parseLabelInstruction(instructionName, arg1)
	case arg1.Type.IsTerminator():
		// Implied instruction
		return ast.NewInstruction(instructionName, registerAddressing, nil, nil), nil
	default:
		return nil, fmt.Errorf("unsupported instruction argument type %s", arg1.Type)
	}
}

func parseTwoOperandInstruction(instructionName string, arg1, arg2 token.Token) (ast.Node, error) {
	// Determine addressing mode based on operands
	var addressing int
	var argument ast.Node

	// For now, handle simple register-to-register and immediate-to-register
	switch {
	case isRegisterToken(arg1) && isRegisterToken(arg2):
		addressing = registerAddressing
		regName := strings.ToUpper(arg2.Value)
		if _, ok := x86Registers[regName]; !ok {
			return nil, fmt.Errorf("unknown register: %s", arg2.Value)
		}
		argument = ast.NewIdentifier(regName)
	case isRegisterToken(arg1) && arg2.Type == token.Number:
		switch {
		case arg2.Value[0] == '#':
			addressing = immediateAddressing
			i, err := number.Parse(arg2.Value)
			if err != nil {
				return nil, fmt.Errorf("parsing immediate value '%s': %w", arg2.Value, err)
			}
			if i > math.MaxUint16 {
				return nil, fmt.Errorf("immediate value '%s' exceeds word value", arg2.Value)
			}
			argument = ast.NewNumber(i)
		default:
			addressing = directAddressing
			i, err := number.Parse(arg2.Value)
			if err != nil {
				return nil, fmt.Errorf("parsing address '%s': %w", arg2.Value, err)
			}
			argument = ast.NewNumber(i)
		}
	default:
		return nil, fmt.Errorf("unsupported operand combination: %s, %s", arg1.Value, arg2.Value)
	}

	if !hasAddressing(instructionName, addressing) {
		return nil, fmt.Errorf("instruction %s does not support addressing mode %d", instructionName, addressing)
	}

	return ast.NewInstruction(instructionName, addressing, argument, nil), nil
}

func parseImmediateInstruction(instructionName string, arg token.Token) (ast.Node, error) {
	if !hasAddressing(instructionName, immediateAddressing) {
		return nil, errors.New("invalid immediate addressing mode usage")
	}

	i, err := number.Parse(arg.Value)
	if err != nil {
		return nil, fmt.Errorf("parsing immediate argument '%s': %w", arg.Value, err)
	}
	if i > math.MaxUint16 {
		return nil, fmt.Errorf("immediate argument '%s' exceeds word value", arg.Value)
	}

	n := ast.NewNumber(i)
	return ast.NewInstruction(instructionName, immediateAddressing, n, nil), nil
}

func parseDirectAddressInstruction(instructionName string, arg token.Token) (ast.Node, error) {
	if !hasAddressing(instructionName, directAddressing) {
		return nil, errors.New("invalid direct addressing mode usage")
	}

	i, err := number.Parse(arg.Value)
	if err != nil {
		return nil, fmt.Errorf("parsing address argument '%s': %w", arg.Value, err)
	}

	n := ast.NewNumber(i)
	return ast.NewInstruction(instructionName, directAddressing, n, nil), nil
}

func parseLabelInstruction(instructionName string, arg token.Token) (ast.Node, error) {
	// For jumps and calls, use immediate addressing (relative)
	if instructionName == "JMP" || instructionName == "CALL" || strings.HasPrefix(instructionName, "J") {
		if !hasAddressing(instructionName, immediateAddressing) {
			return nil, fmt.Errorf("instruction %s does not support label addressing", instructionName)
		}
		l := ast.NewLabel(arg.Value)
		return ast.NewInstruction(instructionName, immediateAddressing, l, nil), nil
	}

	// For other instructions, assume direct addressing
	if !hasAddressing(instructionName, directAddressing) {
		return nil, errors.New("invalid label addressing mode usage")
	}

	l := ast.NewLabel(arg.Value)
	return ast.NewInstruction(instructionName, directAddressing, l, nil), nil
}

func isRegisterToken(t token.Token) bool {
	if t.Type != token.Identifier {
		return false
	}
	_, ok := x86Registers[strings.ToUpper(t.Value)]
	return ok
}

func hasAddressing(instructionName string, addressing int) bool {
	switch instructionName {
	case "MOV", "ADD", "SUB", "CMP", "AND", "OR", "XOR":
		return addressing == registerAddressing || addressing == immediateAddressing || addressing == directAddressing
	case "INC", "DEC", "PUSH", "POP":
		return addressing == registerAddressing
	case "JMP", "CALL", "JE", "JNE", "JC", "JNC", "JL", "JLE", "JG", "JGE", "JS", "JNS", "JO", "JNO":
		return addressing == immediateAddressing
	case "NOP", "RET":
		return addressing == registerAddressing
	default:
		return false
	}
}

func isSingleRegisterInstruction(name string) bool {
	switch name {
	case "INC", "DEC", "PUSH", "POP":
		return true
	default:
		return false
	}
}
