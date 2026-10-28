package parser

import (
	"github.com/retroenv/retroasm/pkg/parser/ast"
	isa "github.com/retroenv/retrogolib/arch/cpu/x86"
)

// RegisterOperand retains a canonical ISA register and its symbol-free contract.
type RegisterOperand struct {
	Register isa.RegisterParam
}

// InstructionFormKey identifies the exact register view.
func (operand RegisterOperand) InstructionFormKey() string {
	return "x86-register/" + operand.Register.String()
}

// InstructionReferences reports that a register has no symbol references.
func (RegisterOperand) InstructionReferences() []ast.InstructionReference {
	return nil
}

// RewriteInstructionSymbols retains a register without invoking the callback.
func (operand RegisterOperand) RewriteInstructionSymbols(_ func(string) string) (any, error) {
	return operand, nil
}

// RegisterValue resolves a typed register operand to its canonical ISA value.
func RegisterValue(value any) (isa.RegisterParam, bool) {
	operand, valid := value.(RegisterOperand)
	return operand.Register, valid
}

func normalizeRegisterOperand(node ast.Node) ast.Node {
	argument, typed := node.(ast.InstructionArgument)
	if !typed {
		return node
	}
	register, canonical := argument.Value.(isa.RegisterParam)
	if !canonical {
		return node
	}
	argument = argument.Copy().(ast.InstructionArgument)
	argument.Value = RegisterOperand{
		Register: register,
	}

	return argument
}
