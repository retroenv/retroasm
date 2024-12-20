// Package arch contains types and functions used for multi architecture support.
package arch

import (
	"github.com/retroenv/retroasm/lexer/token"
	"github.com/retroenv/retroasm/parser/ast"
)

// Architecture contains architecture specific information.
type Architecture[T any] interface {
	// AddressWidth returns the address width of the architecture in bits.
	AddressWidth() int
	// AssignInstructionAddress assigns an address to the instruction.
	AssignInstructionAddress(assigner AddressAssigner, ins Instruction) (uint64, error)
	// GenerateInstructionOpcode generates the instruction opcode based on the instruction base opcode,
	// its addressing mode and parameters.
	GenerateInstructionOpcode(assigner AddressAssigner, ins Instruction) error
	// Instruction returns the instruction with the given name.
	Instruction(name string) (T, bool)
	// ParseIdentifier parses an identifier and returns the corresponding node.
	ParseIdentifier(p Parser, ins T) (ast.Node, error)
}

// Parser processes an input stream and parses its token to produce an abstract syntax tree (AST) as output.
type Parser interface {
	// AddressWidth returns the address width of the architecture in bits.
	AddressWidth() int
	// AdvanceReadPosition advances the token read position.
	AdvanceReadPosition(offset int)
	// NextToken returns the current or a following token with the given offset from current token parse position.
	// If the offset exceeds the available tokens, a token of type EOF is returned.
	NextToken(offset int) token.Token
}

type AddressAssigner interface {
	// ArgumentValue returns the value of an instruction argument, either a number or a symbol value.
	ArgumentValue(argument any) (uint64, error)
	// RelativeOffset returns the relative offset between two addresses.
	RelativeOffset(destination, addressAfterInstruction uint64) (byte, error)
	// ProgramCounter returns the current program counter.
	ProgramCounter() uint64
}

type Instruction interface {
	// Address returns the assigned start address of the instruction.
	Address() uint64
	// Addressing returns the addressing mode of the instruction.
	Addressing() int
	// Argument returns the instruction argument.
	Argument() any
	// Name returns the instruction name.
	Name() string
	// Opcodes returns the instruction opcodes.
	Opcodes() []byte
	// Size returns the size of the instruction in bytes.
	Size() int

	// SetAddress sets the assigned start address of the instruction.
	SetAddress(uint64)
	// SetAddressing sets the addressing mode of the instruction.
	SetAddressing(int)
	// SetOpcodes sets the instruction opcodes.
	SetOpcodes([]byte)
	// SetSize sets the size of the instruction in bytes.
	SetSize(int)
}
