// Package parser implements CPU68000 assembly instruction parsing.
package parser

import (
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
)

// ResolvedInstruction contains the fully parsed CPU68000 instruction.
type ResolvedInstruction struct {
	Instruction  *cpu68000.Instruction
	Size         cpu68000.OperandSize
	ExplicitSize bool
	SrcEA        *EffectiveAddress
	DstEA        *EffectiveAddress
	Extra        uint16 // condition code, quick value, trap vector, etc.
}

// EffectiveAddress represents a parsed effective address operand.
type EffectiveAddress struct {
	Mode      cpu68000.AddressingMode
	Register  uint8                // register number (0-7)
	IndexReg  uint8                // index register for indexed modes
	IndexSize cpu68000.OperandSize // index register size (.W/.L)
	IsAddrReg bool                 // index is address reg (vs data reg)
	Value     ast.Node             // immediate/displacement/address value
	RegList   uint16               // MOVEM register list bitmask
	Negative  bool                 // numeric value was written with a leading minus
}

// CopyInstructionArgument returns a deep copy suitable for AST duplication.
func (resolved ResolvedInstruction) CopyInstructionArgument() any {
	resolved.SrcEA = copyEffectiveAddress(resolved.SrcEA)
	resolved.DstEA = copyEffectiveAddress(resolved.DstEA)
	return resolved
}

// InstructionFormKey returns the selected size and effective-address shapes.
func (resolved ResolvedInstruction) InstructionFormKey() string {
	return fmt.Sprintf(
		"size=%d;explicit=%t;source=%s;destination=%s;extra=%t",
		resolved.Size,
		resolved.ExplicitSize,
		cpu68000EffectiveAddressForm(resolved.SrcEA),
		cpu68000EffectiveAddressForm(resolved.DstEA),
		resolved.Extra != 0,
	)
}

func cpu68000EffectiveAddressForm(address *EffectiveAddress) string {
	if address == nil {
		return "none"
	}
	parts := []string{
		fmt.Sprintf("mode=%d", address.Mode),
		fmt.Sprintf("register=%d", address.Register),
		fmt.Sprintf("index=%d/%d/%t", address.IndexReg, address.IndexSize, address.IsAddrReg),
		fmt.Sprintf("register-list=%t", address.RegList != 0),
		fmt.Sprintf("negative=%t", address.Negative),
		"value=" + ast.InstructionArgumentForm(address.Value),
	}
	return strings.Join(parts, "/")
}

// InstructionReferences returns copied symbol-bearing effective-address values for stream validation.
func (resolved ResolvedInstruction) InstructionReferences() []ast.InstructionReference {
	references := make([]ast.InstructionReference, 0, 2)

	for _, address := range []*EffectiveAddress{resolved.SrcEA, resolved.DstEA} {
		if address == nil || !effectiveAddressReferencesSymbol(address.Value) {
			continue
		}
		references = append(references, ast.InstructionReference{
			Value:         address.Value.Copy(),
			ReferenceType: ast.FullAddress,
		})
	}
	return references
}

func copyEffectiveAddress(address *EffectiveAddress) *EffectiveAddress {
	if address == nil {
		return nil
	}
	copied := *address
	if address.Value != nil {
		copied.Value = address.Value.Copy()
	}
	return &copied
}

func effectiveAddressReferencesSymbol(value ast.Node) bool {
	if value == nil {
		return false
	}
	if ast.SymbolName(value) != "" {
		return true
	}
	expression, ok := value.(ast.Expression)
	if !ok {
		return false
	}
	_, _, ok = ast.ParseSymbolReference(expression.Value)
	return ok
}
