package parser

import (
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/parser/ast"
	isa "github.com/retroenv/retrogolib/arch/cpu/x86"
)

// FormatInstruction returns parseable Intel syntax for the typed operand set.
func FormatInstruction(instruction ast.Instruction) (string, error) {
	text := strings.ToLower(instruction.Name)
	if instruction.Argument == nil {
		return text, nil
	}
	var operands []ast.Node
	if list, ok := instruction.Argument.(ast.InstructionArguments); ok {
		operands = list.Values
	} else {
		operands = []ast.Node{instruction.Argument}
	}

	values := make([]string, 0, len(operands))
	for _, operand := range operands {
		value, err := formatOperand(operand)
		if err != nil {
			return "", err
		}
		values = append(values, value)
	}

	return text + " " + strings.Join(values, ", "), nil
}

func formatOperand(operand ast.Node) (string, error) {
	if typed, ok := operand.(ast.InstructionArgument); ok {
		switch value := typed.Value.(type) {
		case MemoryOperand:
			return formatMemory(value)
		case SignedImmediate:
			return fmt.Sprintf("%d", value), nil
		case RepeatedString:
			return string(value), nil
		}

		register, ok := RegisterValue(typed.Value)
		if !ok || register < isa.RegAL || register > isa.RegDS {
			return "", fmt.Errorf("unsupported 8086 register %v", typed.Value)
		}
		return register.String(), nil
	}
	text, err := ast.FormatValue(operand, ast.ValueFormatOptions{})
	if err != nil {
		return "", fmt.Errorf("formatting 8086 operand: %w", err)
	}
	return text, nil
}
