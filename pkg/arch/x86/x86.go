// Package x86 supplies the tested 8086 instruction subset for DOS output.
package x86

import (
	"fmt"
	"slices"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/x86/assembler"
	"github.com/retroenv/retroasm/pkg/arch/x86/parser"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	retroarch "github.com/retroenv/retrogolib/arch"
	isa "github.com/retroenv/retrogolib/arch/cpu/x86"
)

// Keep existing identities stable when the subset grows.
var opcodeNames = []string{
	isa.MovName, isa.IntName, isa.CldName, isa.RetName, isa.NopName,
	isa.PushName, isa.PopName, isa.CallName, isa.JmpName, isa.JzName, isa.JnzName,
	isa.AddName, isa.SubName, isa.CmpName, isa.AndName, isa.OrName, isa.XorName,
	isa.AdcName, isa.SbbName, isa.TestName, isa.XchgName, isa.LeaName,
	isa.IncName, isa.DecName, isa.NegName, isa.NotName, isa.MulName, isa.ImulName, isa.DivName, isa.IdivName,
	isa.ShlName, isa.ShrName, isa.SarName, isa.RolName, isa.RorName, isa.RclName, isa.RcrName,
	isa.ClcName, isa.StcName, isa.CmcName, isa.StdName, isa.CbwName, isa.CwdName, isa.IretName,
	isa.MovsbName, isa.MovswName, isa.CmpsbName, isa.CmpswName, isa.StosbName, isa.StoswName,
	isa.LodsbName, isa.LodswName, isa.ScasbName, isa.ScaswName, isa.RepName, isa.RepzName, isa.RepnzName,
	isa.JoName, isa.JnoName, isa.JbName, isa.JnbName, isa.JbeName, isa.JnbeName, isa.JsName, isa.JnsName,
	isa.JpName, isa.JnpName, isa.JlName, isa.JnlName, isa.JleName, isa.JnleName,
}

var mnemonicAliases = map[string]string{
	"je": isa.JzName, "jne": isa.JnzName, "jc": isa.JbName, "jnae": isa.JbName,
	"jnc": isa.JnbName, "jae": isa.JnbName, "jna": isa.JbeName, "ja": isa.JnbeName,
	"jpe": isa.JpName, "jpo": isa.JnpName, "jnge": isa.JlName, "jge": isa.JnlName,
	"jng": isa.JleName, "jg": isa.JnleName, "sal": isa.ShlName, "repe": isa.RepzName,
	"repne": isa.RepnzName,
}

// New returns an 8086 configuration. Unsupported operand forms return errors.
func New() *config.Config[*isa.Instruction] {
	return &config.Config[*isa.Instruction]{
		Arch: &architecture{},
	}
}

// NewInstruction retains all operands without parsing assembly text.
func NewInstruction(name string, operands ...ast.Node) (ast.Instruction, error) {
	metadata, ok := (&architecture{}).Instruction(name)
	if !ok {
		return ast.Instruction{}, fmt.Errorf("unsupported 8086 instruction %q", name)
	}
	instruction := parser.BuildInstruction(metadata.Name, operands...)
	instruction.OpcodeID = (&architecture{}).OpcodeID(metadata)
	if err := assembler.ValidateInstruction(instruction); err != nil {
		return ast.Instruction{}, fmt.Errorf("building 8086 instruction: %w", err)
	}

	return instruction, nil
}

type architecture struct{}

func (*architecture) AddressWidth() int        { return 16 }
func (*architecture) ByteOrder() ast.ByteOrder { return ast.ByteOrderLittle }

func (*architecture) Instruction(name string) (*isa.Instruction, bool) {
	name = strings.ToLower(name)

	if canonical, known := mnemonicAliases[name]; known {
		name = canonical
	}

	if slices.Contains(opcodeNames, name) {
		instruction, ok := isa.Instructions[name]
		return instruction, ok
	}
	return nil, false
}

func (*architecture) OpcodeID(instruction *isa.Instruction) ast.OpcodeID {
	for index, name := range opcodeNames {
		if name == instruction.Name {
			return ast.NewOpcodeID(retroarch.X86, uint16(index+1))
		}
	}

	return ast.OpcodeID{}
}

func (*architecture) ParseIdentifier(p arch.Parser, _ string, instruction *isa.Instruction) (ast.Node, error) {
	node, err := parser.ParseIdentifier(p, instruction.Name)
	if err != nil {
		return nil, fmt.Errorf("parsing 8086 instruction: %w", err)
	}
	return node, nil
}

func (*architecture) AssignInstructionAddress(assigner arch.AddressAssigner,
	instruction arch.Instruction) (uint64, error) {

	next, err := assembler.AssignInstructionAddress(assigner, instruction)
	if err != nil {
		return 0, fmt.Errorf("assigning 8086 instruction address: %w", err)
	}
	return next, nil
}

func (*architecture) GenerateInstructionOpcode(assigner arch.AddressAssigner, instruction arch.Instruction) error {
	if err := assembler.GenerateInstructionOpcode(assigner, instruction); err != nil {
		return fmt.Errorf("encoding 8086 instruction: %w", err)
	}
	return nil
}

// FormatInstruction returns Intel syntax with both retained operands.
func (*architecture) FormatInstruction(instruction ast.Instruction) (string, error) {
	text, err := parser.FormatInstruction(instruction)
	if err != nil {
		return "", fmt.Errorf("formatting 8086 instruction: %w", err)
	}
	return text, nil
}

// ValidateInstruction checks the same forms used for sizing and encoding.
func (*architecture) ValidateInstruction(instruction ast.Instruction) error {
	if err := assembler.ValidateInstruction(instruction); err != nil {
		return fmt.Errorf("validating 8086 instruction: %w", err)
	}
	return nil
}

// RelaxInstruction expands a conditional branch that cannot reach its target.
func (*architecture) RelaxInstruction(assigner arch.AddressAssigner, instruction arch.Instruction) (bool, error) {
	changed, err := assembler.RelaxInstruction(assigner, instruction)
	if err != nil {
		return false, fmt.Errorf("relaxing 8086 instruction: %w", err)
	}
	return changed, nil
}
