// Package cpu65816 provides a WDC 65C816 architecture specific assembler code.
package cpu65816

import (
	"fmt"
	"slices"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/cpu65816/assembler"
	"github.com/retroenv/retroasm/pkg/arch/cpu65816/parser"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	retroarch "github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/arch/cpu/cpu65816"
)

// New returns a new 65816 architecture configuration.
func New() *config.Config[*cpu65816.Instruction] {
	p := &arch65816[*cpu65816.Instruction]{}
	cfg := &config.Config[*cpu65816.Instruction]{
		Arch: p,
	}
	return cfg
}

type arch65816[T any] struct {
}

func (ar *arch65816[T]) AddressWidth() int {
	return 24
}

func (*arch65816[T]) ByteOrder() ast.ByteOrder {
	return ast.ByteOrderLittle
}

func (ar *arch65816[T]) BuildInstruction(
	mnemonic string,
	operands parser.Operands,
) (ast.Instruction, error) {

	lookupName := strings.ToLower(strings.TrimSpace(mnemonic))
	instruction, ok := ar.Instruction(lookupName)
	if !ok {
		return ast.Instruction{}, fmt.Errorf("unknown CPU65816 instruction %q", mnemonic)
	}
	built, err := parser.BuildInstruction(lookupName, instruction, operands)
	if err != nil {
		return ast.Instruction{}, err //nolint:wrapcheck // architecture codec boundary adds context
	}
	built.SetOpcodeID(ar.OpcodeID(instruction))
	return built, nil
}

func (ar *arch65816[T]) BuildInstructionWithState(
	mnemonic string,
	operands parser.Operands,
	state parser.State,
) (ast.Instruction, parser.State, error) {

	lookupName := strings.ToLower(strings.TrimSpace(mnemonic))
	instruction, ok := ar.Instruction(lookupName)
	if !ok {
		return ast.Instruction{}, parser.State{}, fmt.Errorf("unknown CPU65816 instruction %q", mnemonic)
	}
	built, nextState, err := parser.BuildInstructionWithState(lookupName, instruction, operands, state)
	if err != nil {
		return ast.Instruction{}, parser.State{}, err //nolint:wrapcheck // architecture codec boundary adds context
	}
	built.SetOpcodeID(ar.OpcodeID(instruction))
	return built, nextState, nil
}

func (ar *arch65816[T]) Instruction(name string) (*cpu65816.Instruction, bool) {
	ins, ok := cpu65816.Instructions[name]
	return ins, ok
}

func (ar *arch65816[T]) OpcodeID(ins *cpu65816.Instruction) ast.OpcodeID {
	return ast.NewOpcodeID(retroarch.CPU65816, uint16(cpu65816.NameToOpcodeID[ins.Name]))
}

// InstructionRegistrations returns the CPU65816 mnemonic and addressing registrations.
func (ar *arch65816[T]) InstructionRegistrations() []arch.InstructionRegistration {
	registrations := make([]arch.InstructionRegistration, 0, len(cpu65816.Instructions))
	for _, instruction := range cpu65816.Instructions {
		addressings := make([]int, 0, len(instruction.Addressing))
		for addressing := range instruction.Addressing {
			addressings = append(addressings, int(addressing))
		}
		slices.Sort(addressings)
		registrations = append(registrations, arch.InstructionRegistration{
			Name: instruction.Name, OpcodeID: ar.OpcodeID(instruction), Addressings: addressings,
		})
	}
	slices.SortFunc(registrations, func(left, right arch.InstructionRegistration) int {
		return strings.Compare(left.Name, right.Name)
	})
	return registrations
}

func (ar *arch65816[T]) ParseIdentifier(p arch.Parser, _ string, ins *cpu65816.Instruction) (ast.Node, error) {
	return parser.ParseIdentifier(p, ins) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *arch65816[T]) ValidateInstruction(instruction ast.Instruction) error {
	details, ok := ar.Instruction(strings.ToLower(strings.TrimSpace(instruction.Name)))
	if !ok {
		return fmt.Errorf("unknown CPU65816 instruction %q", instruction.Name)
	}
	expectedID := ar.OpcodeID(details)
	if instruction.OpcodeID != expectedID {
		return fmt.Errorf(
			"CPU65816 opcode identity %+v does not match mnemonic %q identity %+v",
			instruction.OpcodeID,
			instruction.Name,
			expectedID,
		)
	}
	return parser.ValidateInstruction(instruction, details) //nolint:wrapcheck // architecture codec boundary adds context
}

func (ar *arch65816[T]) FormatInstruction(instruction ast.Instruction) (string, error) {
	if err := ar.ValidateInstruction(instruction); err != nil {
		return "", err
	}
	return parser.FormatInstruction(instruction) //nolint:wrapcheck // architecture codec boundary adds context
}

func (ar *arch65816[T]) AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	return assembler.AssignInstructionAddress(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *arch65816[T]) GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	return assembler.GenerateInstructionOpcode(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}
