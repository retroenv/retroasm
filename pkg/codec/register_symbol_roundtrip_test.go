package codec_test

import (
	"fmt"
	"strings"
	"testing"

	cpu65816parser "github.com/retroenv/retroasm/pkg/arch/cpu65816/parser"
	sm83parser "github.com/retroenv/retroasm/pkg/arch/sm83/parser"
	z80parser "github.com/retroenv/retroasm/pkg/arch/z80/parser"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/sm83"
	"github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/assert"
)

func TestZ80CodecRegisterNamedSymbolsRoundTrip(t *testing.T) {
	c := newZ80Codec(t)
	for _, name := range []string{"A", "AF", "B", "BC", "C", "D", "DE", "E", "H", "HL", "I", "IX", "IY", "L", "R", "SP", "NZ", "Z", "NC", "PO", "PE", "P", "M", "hl", "ix", "IXH", "IXL", "IYH", "IYL", "ixh", "ixl", "iyh", "iyl", "ordinary"} {
		value := z80parser.ValueOperand(ast.NewLabel(name))
		memory := z80parser.IndirectValueOperand(ast.NewLabel(name))
		for _, test := range []struct {
			mnemonic string
			operands z80parser.Operands
		}{
			{"ld", z80parser.Operands{z80parser.RegisterOperand(z80.RegHL), value}},
			{"ld", z80parser.Operands{z80parser.RegisterOperand(z80.RegA), memory}},
			{"ld", z80parser.Operands{memory, z80parser.RegisterOperand(z80.RegA)}},
			{"jp", z80parser.Operands{value}},
			{"jp", z80parser.Operands{z80parser.RegisterOperand(z80.RegCondNZ), value}},
		} {
			t.Run(fmt.Sprintf("%s/%s/%d", name, test.mnemonic, test.operands[0].Kind), func(t *testing.T) {
				built, err := codec.BuildInstruction(c, test.mnemonic, test.operands)
				assert.NoError(t, err)
				argument := built.Argument.(ast.InstructionArgument).Value.(z80parser.ResolvedInstruction)
				for index, operand := range test.operands {
					assert.Equal(t, operand.Kind, argument.Operands[index].Kind)
				}
				formatted, err := c.FormatInstruction(built)
				assert.NoError(t, err)
				parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
				assert.NoError(t, err)
				assert.NoError(t, c.ValidateInstruction(parsed))
				want, err := c.Assemble(t.Context(), []ast.Node{ast.NewLabel(name), built})
				assert.NoError(t, err)
				got, err := c.Assemble(t.Context(), []ast.Node{ast.NewLabel(name), parsed})
				assert.NoError(t, err)
				assert.Equal(t, want.Binary, got.Binary)
			})
		}
	}
}

func TestCPU65816CodecRegisterNamedSymbolsRoundTrip(t *testing.T) {
	c := newCPU65816Codec(t)
	for _, name := range []string{"A", "a", "C", "X", "Y", "S", "SP", "Z", "F", "ordinary"} {
		for _, test := range []struct {
			mnemonic string
			kind     cpu65816parser.OperandKind
		}{
			{"lda", cpu65816parser.OperandAddress},
			{"sta", cpu65816parser.OperandIndexedX},
			{"inc", cpu65816parser.OperandAddress},
			{"jsr", cpu65816parser.OperandAddress},
		} {
			t.Run(name+"/"+test.mnemonic, func(t *testing.T) {
				operands := cpu65816parser.Operands{cpu65816parser.MemoryOperand(test.kind, cpu65816parser.AddressDefault, ast.NewLabel(name))}
				built, err := codec.BuildInstruction(c, test.mnemonic, operands)
				assert.NoError(t, err)
				formatted, err := c.FormatInstruction(built)
				assert.NoError(t, err)
				parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
				assert.NoError(t, err)
				assert.NoError(t, c.ValidateInstruction(parsed))
				want, err := c.Assemble(t.Context(), []ast.Node{ast.NewLabel(name), built})
				assert.NoError(t, err)
				got, err := c.Assemble(t.Context(), []ast.Node{ast.NewLabel(name), parsed})
				assert.NoError(t, err)
				assert.Equal(t, want.Binary, got.Binary)
			})
		}
	}
}

func TestSM83CodecRegisterNamedSymbolsRoundTrip(t *testing.T) {
	c := newSM83Codec(t)
	for _, name := range []string{"A", "AF", "B", "BC", "C", "D", "DE", "E", "H", "HL", "L", "SP", "NZ", "Z", "NC", "hl", "c", "ordinary"} {
		value := sm83parser.ValueOperand(ast.NewLabel(name))
		memory := sm83parser.IndirectValueOperand(ast.NewLabel(name))
		for _, test := range []struct {
			mnemonic string
			operands sm83parser.Operands
		}{
			{"ld", sm83parser.Operands{sm83parser.RegisterOperand(sm83.RegHL), value}},
			{"ld", sm83parser.Operands{sm83parser.RegisterOperand(sm83.RegA), memory}},
			{"ld", sm83parser.Operands{memory, sm83parser.RegisterOperand(sm83.RegA)}},
			{"jp", sm83parser.Operands{value}},
			{"jp", sm83parser.Operands{sm83parser.RegisterOperand(sm83.RegCondNZ), value}},
		} {
			t.Run(fmt.Sprintf("%s/%s/%d", name, test.mnemonic, test.operands[0].Kind), func(t *testing.T) {
				built, err := codec.BuildInstruction(c, test.mnemonic, test.operands)
				assert.NoError(t, err)
				argument := built.Argument.(ast.InstructionArgument).Value.(sm83parser.ResolvedInstruction)
				for index, operand := range test.operands {
					assert.Equal(t, operand.Kind, argument.Operands[index].Kind)
				}
				formatted, err := c.FormatInstruction(built)
				assert.NoError(t, err)
				parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
				assert.NoError(t, err)
				assert.NoError(t, c.ValidateInstruction(parsed))
				want, err := c.Assemble(t.Context(), []ast.Node{ast.NewLabel(name), built})
				assert.NoError(t, err)
				got, err := c.Assemble(t.Context(), []ast.Node{ast.NewLabel(name), parsed})
				assert.NoError(t, err)
				assert.Equal(t, want.Binary, got.Binary)
			})
		}
	}
}
