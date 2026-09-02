package codec_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	asmcpu6502 "github.com/retroenv/retroasm/pkg/arch/cpu6502"
	cpu6502parser "github.com/retroenv/retroasm/pkg/arch/cpu6502/parser"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/arch/cpu/cpu6502"
	"github.com/retroenv/retrogolib/assert"
)

//nolint:funlen // Addressing-family coverage is intentionally one auditable table.
func TestCPU6502Codec_AddressingFamiliesRoundTrip(t *testing.T) {
	t.Parallel()

	c := newCPU6502AssemblyCodec(t)
	tests := []struct {
		name     string
		mnemonic string
		operands cpu6502parser.Operands
		wantText string
		wantCode []byte
	}{
		{name: "implied", mnemonic: cpu6502.NopName, wantText: "nop", wantCode: []byte{0xea}},
		{name: "accumulator", mnemonic: cpu6502.AslName, wantText: "asl a", wantCode: []byte{0x0a}},
		{
			name: "immediate", mnemonic: cpu6502.LdaName,
			operands: cpu6502parser.Operands{cpu6502parser.ImmediateOperand(ast.NewNumber(0x12))},
			wantText: "lda #$12", wantCode: []byte{0xa9, 0x12},
		},
		{
			name: "zero page", mnemonic: cpu6502.LdaName,
			operands: cpu6502parser.Operands{cpu6502parser.MemoryOperand(
				cpu6502parser.OperandAddress, cpu6502parser.AddressZeroPage, ast.NewNumber(0x10),
			)},
			wantText: "lda z:$10", wantCode: []byte{0xa5, 0x10},
		},
		{
			name: "absolute", mnemonic: cpu6502.LdaName,
			operands: cpu6502parser.Operands{cpu6502parser.MemoryOperand(
				cpu6502parser.OperandAddress, cpu6502parser.AddressAbsolute, ast.NewNumber(0x1234),
			)},
			wantText: "lda a:$1234", wantCode: []byte{0xad, 0x34, 0x12},
		},
		{
			name: "zero page indexed x", mnemonic: cpu6502.LdaName,
			operands: cpu6502parser.Operands{cpu6502parser.MemoryOperand(
				cpu6502parser.OperandIndexedX, cpu6502parser.AddressZeroPage, ast.NewNumber(0x10),
			)},
			wantText: "lda z:$10,x", wantCode: []byte{0xb5, 0x10},
		},
		{
			name: "absolute indexed x", mnemonic: cpu6502.LdaName,
			operands: cpu6502parser.Operands{cpu6502parser.MemoryOperand(
				cpu6502parser.OperandIndexedX, cpu6502parser.AddressAbsolute, ast.NewNumber(0x1234),
			)},
			wantText: "lda a:$1234,x", wantCode: []byte{0xbd, 0x34, 0x12},
		},
		{
			name: "zero page indexed y", mnemonic: cpu6502.LdxName,
			operands: cpu6502parser.Operands{cpu6502parser.MemoryOperand(
				cpu6502parser.OperandIndexedY, cpu6502parser.AddressZeroPage, ast.NewNumber(0x10),
			)},
			wantText: "ldx z:$10,y", wantCode: []byte{0xb6, 0x10},
		},
		{
			name: "absolute indexed y", mnemonic: cpu6502.LdaName,
			operands: cpu6502parser.Operands{cpu6502parser.MemoryOperand(
				cpu6502parser.OperandIndexedY, cpu6502parser.AddressAbsolute, ast.NewNumber(0x1234),
			)},
			wantText: "lda a:$1234,y", wantCode: []byte{0xb9, 0x34, 0x12},
		},
		{
			name: "absolute indirect", mnemonic: cpu6502.JmpName,
			operands: cpu6502parser.Operands{cpu6502parser.MemoryOperand(
				cpu6502parser.OperandIndirect, cpu6502parser.AddressAbsolute, ast.NewNumber(0x1234),
			)},
			wantText: "jmp a:($1234)", wantCode: []byte{0x6c, 0x34, 0x12},
		},
		{
			name: "indexed x indirect", mnemonic: cpu6502.LdaName,
			operands: cpu6502parser.Operands{cpu6502parser.MemoryOperand(
				cpu6502parser.OperandIndexedXIndirect, cpu6502parser.AddressZeroPage, ast.NewNumber(0x10),
			)},
			wantText: "lda z:($10,x)", wantCode: []byte{0xa1, 0x10},
		},
		{
			name: "indirect indexed y", mnemonic: cpu6502.LdaName,
			operands: cpu6502parser.Operands{cpu6502parser.MemoryOperand(
				cpu6502parser.OperandIndirectIndexedY, cpu6502parser.AddressZeroPage, ast.NewNumber(0x10),
			)},
			wantText: "lda z:($10),y", wantCode: []byte{0xb1, 0x10},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			built, err := codec.BuildInstruction(c, test.mnemonic, test.operands)
			assert.NoError(t, err)
			assert.True(t, built.OpcodeID.ValidFor(arch.CPU6502))
			assert.NoError(t, c.ValidateInstruction(built))

			formatted, err := c.FormatInstruction(built)
			assert.NoError(t, err)
			assert.Equal(t, test.wantText, formatted)

			parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
			assert.NoError(t, err)
			assert.NoError(t, c.ValidateInstruction(parsed))

			builtAssembly, err := c.Assemble(t.Context(), []ast.Node{built})
			assert.NoError(t, err)
			parsedAssembly, err := c.Assemble(t.Context(), []ast.Node{parsed})
			assert.NoError(t, err)
			assert.Equal(t, test.wantCode, builtAssembly.Binary)
			assert.Equal(t, builtAssembly.Binary, parsedAssembly.Binary)
		})
	}
}

func TestCPU6502Codec_RelativeBranchRoundTrip(t *testing.T) {
	t.Parallel()

	c := newCPU6502AssemblyCodec(t)
	branch, err := codec.BuildInstruction(
		c,
		cpu6502.BneName,
		cpu6502parser.Operands{cpu6502parser.MemoryOperand(
			cpu6502parser.OperandAddress, cpu6502parser.AddressDefault, ast.NewLabel("target"),
		)},
	)
	assert.NoError(t, err)
	formatted, err := c.FormatInstruction(branch)
	assert.NoError(t, err)
	assert.Equal(t, "bne target", formatted)
	parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
	assert.NoError(t, err)
	assert.NoError(t, c.ValidateInstruction(parsed))

	nop, err := codec.BuildInstruction(c, cpu6502.NopName, cpu6502parser.Operands{})
	assert.NoError(t, err)
	builtAssembly, err := c.Assemble(t.Context(), []ast.Node{branch, nop, ast.NewLabel("target")})
	assert.NoError(t, err)
	parsedAssembly, err := c.Assemble(t.Context(), []ast.Node{parsed, nop, ast.NewLabel("target")})
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xd0, 0x01, 0xea}, builtAssembly.Binary)
	assert.Equal(t, builtAssembly.Binary, parsedAssembly.Binary)
}

//nolint:funlen // The three 65C02-only addressing families belong in one parity table.
func TestCPU6502Codec_65C02AddressingFamiliesRoundTrip(t *testing.T) {
	t.Parallel()

	c := newCPU6502AssemblyCodecForVariant(t, cpu6502.Variant65C02)
	tests := []struct {
		name     string
		mnemonic string
		operands cpu6502parser.Operands
		wantText string
		wantCode []byte
	}{
		{
			name: "zero page indirect", mnemonic: cpu6502.LdaName,
			operands: cpu6502parser.Operands{cpu6502parser.MemoryOperand(
				cpu6502parser.OperandIndirect, cpu6502parser.AddressZeroPage, ast.NewNumber(0x10),
			)},
			wantText: "lda z:($10)", wantCode: []byte{0xb2, 0x10},
		},
		{
			name: "absolute x indirect", mnemonic: cpu6502.JmpName,
			operands: cpu6502parser.Operands{cpu6502parser.MemoryOperand(
				cpu6502parser.OperandIndexedXIndirect, cpu6502parser.AddressAbsolute, ast.NewNumber(0x1234),
			)},
			wantText: "jmp a:($1234,x)", wantCode: []byte{0x7c, 0x34, 0x12},
		},
		{
			name:     "zero page relative",
			mnemonic: cpu6502.Bbr3.Name,
			operands: cpu6502parser.ZeroPageRelativeOperands(ast.NewNumber(0x12), ast.NewNumber(4)),
			wantText: "bbr3 z:$12,$4", wantCode: []byte{0x3f, 0x12, 0x01},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			built, err := codec.BuildInstruction(c, test.mnemonic, test.operands)
			assert.NoError(t, err)
			assert.True(t, built.OpcodeID.ValidFor(arch.CPU6502))
			assert.NoError(t, c.ValidateInstruction(built))

			formatted, err := c.FormatInstruction(built)
			assert.NoError(t, err)
			assert.Equal(t, test.wantText, formatted)
			parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
			assert.NoError(t, err)
			assert.NoError(t, c.ValidateInstruction(parsed))

			builtAssembly, err := c.Assemble(t.Context(), []ast.Node{built})
			assert.NoError(t, err)
			parsedAssembly, err := c.Assemble(t.Context(), []ast.Node{parsed})
			assert.NoError(t, err)
			assert.Equal(t, test.wantCode, builtAssembly.Binary)
			assert.Equal(t, builtAssembly.Binary, parsedAssembly.Binary)
		})
	}
}

func TestCPU6502Codec_65C02ZeroPageRelativeSymbolRoundTrip(t *testing.T) {
	t.Parallel()

	c := newCPU6502AssemblyCodecForVariant(t, cpu6502.Variant65C02)
	branch, err := codec.BuildInstruction(
		c,
		cpu6502.Bbs7.Name,
		cpu6502parser.ZeroPageRelativeOperands(ast.NewNumber(0x20), ast.NewLabel("target")),
	)
	assert.NoError(t, err)
	formatted, err := c.FormatInstruction(branch)
	assert.NoError(t, err)
	assert.Equal(t, "bbs7 z:$20,target", formatted)
	parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
	assert.NoError(t, err)

	nop, err := codec.BuildInstruction(c, cpu6502.NopName, cpu6502parser.Operands{})
	assert.NoError(t, err)
	nodes := []ast.Node{branch, nop, ast.NewLabel("target")}
	parsedNodes := []ast.Node{parsed, nop, ast.NewLabel("target")}
	builtAssembly, err := c.Assemble(t.Context(), nodes)
	assert.NoError(t, err)
	parsedAssembly, err := c.Assemble(t.Context(), parsedNodes)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xff, 0x20, 0x01, 0xea}, builtAssembly.Binary)
	assert.Equal(t, builtAssembly.Binary, parsedAssembly.Binary)
}

func TestCPU6502Codec_65C02ConventionalSyntax(t *testing.T) {
	t.Parallel()

	c := newCPU6502AssemblyCodecForVariant(t, cpu6502.Variant65C02)
	tests := []struct {
		name       string
		source     string
		addressing cpu6502.AddressingMode
		wantCode   []byte
	}{
		{
			name: "zero page indirect", source: "lda ($10)",
			addressing: cpu6502.ZeroPageIndirectAddressing, wantCode: []byte{0xb2, 0x10},
		},
		{
			name: "absolute x indirect", source: "jmp ($1234,x)",
			addressing: cpu6502.AbsoluteXIndirectAddressing, wantCode: []byte{0x7c, 0x34, 0x12},
		},
		{
			name: "zero page relative", source: "bbr0 $10,$3",
			addressing: cpu6502.ZeroPageRelativeAddressing, wantCode: []byte{0x0f, 0x10, 0x00},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			instruction, err := c.ParseInstruction(t.Context(), strings.NewReader(test.source))
			assert.NoError(t, err)
			assert.Equal(t, int(test.addressing), instruction.Addressing)
			assert.NoError(t, c.ValidateInstruction(instruction))
			assembly, err := c.Assemble(t.Context(), []ast.Node{instruction})
			assert.NoError(t, err)
			assert.Equal(t, test.wantCode, assembly.Binary)
		})
	}
}

func TestCPU6502Codec_VariantRejectsUnsupportedInstructionsAndModes(t *testing.T) {
	t.Parallel()

	nmos := newCPU6502AssemblyCodecForVariant(t, cpu6502.VariantNMOS6502)
	_, err := codec.BuildInstruction(
		nmos,
		cpu6502.LdaName,
		cpu6502parser.Operands{cpu6502parser.MemoryOperand(
			cpu6502parser.OperandIndirect, cpu6502parser.AddressZeroPage, ast.NewNumber(0x10),
		)},
	)
	assert.Error(t, err)
	_, err = nmos.ParseInstruction(t.Context(), strings.NewReader("bbr0 $10,target"))
	assert.Error(t, err)

	synertek := newCPU6502AssemblyCodecForVariant(t, cpu6502.VariantSynertek65C02)
	_, err = synertek.ParseInstruction(t.Context(), strings.NewReader("bbr0 $10,target"))
	assert.Error(t, err)
}

//nolint:funlen // Every variant and declared mode forms one cross-layer contract matrix.
func TestCPU6502Codec_VariantInstructionModesRoundTrip(t *testing.T) {
	t.Parallel()

	variants := []struct {
		name    string
		variant cpu6502.CPUVariant
	}{
		{name: "nmos6502", variant: cpu6502.VariantNMOS6502},
		{name: "nes6502", variant: cpu6502.VariantNES6502},
		{name: "6507", variant: cpu6502.Variant6507},
		{name: "6510", variant: cpu6502.Variant6510},
		{name: "65c02", variant: cpu6502.Variant65C02},
		{name: "synertek65c02", variant: cpu6502.VariantSynertek65C02},
	}
	for _, variant := range variants {
		t.Run(variant.name, func(t *testing.T) {
			t.Parallel()

			c := newCPU6502AssemblyCodecForVariant(t, variant.variant)
			instructions := cpu6502.InstructionsForVariant(variant.variant)
			names := make([]string, 0, len(instructions))
			for name := range instructions {
				names = append(names, name)
			}
			slices.Sort(names)

			for _, name := range names {
				instruction := instructions[name]
				addressings := make([]cpu6502.AddressingMode, 0, len(instruction.Addressing))
				for addressing := range instruction.Addressing {
					addressings = append(addressings, addressing)
				}
				slices.Sort(addressings)

				for _, addressing := range addressings {
					t.Run(fmt.Sprintf("%s/%d", name, addressing), func(t *testing.T) {
						operands, err := cpu6502OperandsForAddressing(addressing)
						assert.NoError(t, err)
						built, err := codec.BuildInstruction(c, name, operands)
						assert.NoError(t, err)
						assert.Equal(t, int(addressing), built.Addressing)
						assert.NoError(t, c.ValidateInstruction(built))

						formatted, err := c.FormatInstruction(built)
						assert.NoError(t, err)
						parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
						assert.NoError(t, err)
						assert.NoError(t, c.ValidateInstruction(parsed))

						builtAssembly, err := c.Assemble(t.Context(), []ast.Node{built})
						assert.NoError(t, err)
						parsedAssembly, err := c.Assemble(t.Context(), []ast.Node{parsed})
						assert.NoError(t, err)
						info := instruction.Addressing[addressing]
						assert.Len(t, builtAssembly.Binary, int(info.Size))
						assert.Equal(t, info.Opcode, builtAssembly.Binary[0])
						assert.Equal(t, builtAssembly.Binary, parsedAssembly.Binary)
					})
				}
			}
		})
	}
}

func cpu6502OperandsForAddressing(addressing cpu6502.AddressingMode) (cpu6502parser.Operands, error) {
	byteValue := ast.NewNumber(0x12)
	wordValue := ast.NewNumber(0x1234)
	switch addressing {
	case cpu6502.ImpliedAddressing:
		return nil, nil
	case cpu6502.AccumulatorAddressing:
		return cpu6502parser.Operands{cpu6502parser.AccumulatorOperand()}, nil
	case cpu6502.ImmediateAddressing:
		return cpu6502parser.Operands{cpu6502parser.ImmediateOperand(byteValue)}, nil
	case cpu6502.RelativeAddressing:
		return cpu6502MemoryOperands(cpu6502parser.OperandAddress, cpu6502parser.AddressDefault, ast.NewNumber(2)), nil
	case cpu6502.ZeroPageAddressing:
		return cpu6502MemoryOperands(cpu6502parser.OperandAddress, cpu6502parser.AddressZeroPage, byteValue), nil
	case cpu6502.AbsoluteAddressing:
		return cpu6502MemoryOperands(cpu6502parser.OperandAddress, cpu6502parser.AddressAbsolute, wordValue), nil
	case cpu6502.ZeroPageXAddressing:
		return cpu6502MemoryOperands(cpu6502parser.OperandIndexedX, cpu6502parser.AddressZeroPage, byteValue), nil
	case cpu6502.AbsoluteXAddressing:
		return cpu6502MemoryOperands(cpu6502parser.OperandIndexedX, cpu6502parser.AddressAbsolute, wordValue), nil
	case cpu6502.ZeroPageYAddressing:
		return cpu6502MemoryOperands(cpu6502parser.OperandIndexedY, cpu6502parser.AddressZeroPage, byteValue), nil
	case cpu6502.AbsoluteYAddressing:
		return cpu6502MemoryOperands(cpu6502parser.OperandIndexedY, cpu6502parser.AddressAbsolute, wordValue), nil
	case cpu6502.IndirectAddressing:
		return cpu6502MemoryOperands(cpu6502parser.OperandIndirect, cpu6502parser.AddressAbsolute, wordValue), nil
	case cpu6502.IndirectXAddressing:
		return cpu6502MemoryOperands(cpu6502parser.OperandIndexedXIndirect, cpu6502parser.AddressZeroPage, byteValue), nil
	case cpu6502.IndirectYAddressing:
		return cpu6502MemoryOperands(cpu6502parser.OperandIndirectIndexedY, cpu6502parser.AddressZeroPage, byteValue), nil
	case cpu6502.ZeroPageIndirectAddressing:
		return cpu6502MemoryOperands(cpu6502parser.OperandIndirect, cpu6502parser.AddressZeroPage, byteValue), nil
	case cpu6502.AbsoluteXIndirectAddressing:
		return cpu6502MemoryOperands(cpu6502parser.OperandIndexedXIndirect, cpu6502parser.AddressAbsolute, wordValue), nil
	case cpu6502.ZeroPageRelativeAddressing:
		return cpu6502parser.ZeroPageRelativeOperands(byteValue, ast.NewNumber(3)), nil
	default:
		return nil, fmt.Errorf("unsupported CPU6502 addressing %d", addressing)
	}
}

func cpu6502MemoryOperands(
	kind cpu6502parser.OperandKind,
	size cpu6502parser.AddressSize,
	value ast.Node,
) cpu6502parser.Operands {

	return cpu6502parser.Operands{cpu6502parser.MemoryOperand(kind, size, value)}
}

func TestCPU6502Codec_SymbolicAddressAndImmediateExpressionRoundTrip(t *testing.T) {
	t.Parallel()

	c := newCPU6502AssemblyCodec(t)
	load, err := codec.BuildInstruction(
		c,
		cpu6502.LdaName,
		cpu6502parser.Operands{cpu6502parser.MemoryOperand(
			cpu6502parser.OperandAddress,
			cpu6502parser.AddressDefault,
			ast.NewLabel("target"),
		)},
	)
	assert.NoError(t, err)
	formattedLoad, err := c.FormatInstruction(load)
	assert.NoError(t, err)
	assert.Equal(t, "lda target", formattedLoad)
	parsedLoad, err := c.ParseInstruction(t.Context(), strings.NewReader(formattedLoad))
	assert.NoError(t, err)
	assert.NoError(t, c.ValidateInstruction(parsedLoad))

	builtAssembly, err := c.Assemble(t.Context(), []ast.Node{load, ast.NewLabel("target")})
	assert.NoError(t, err)
	parsedAssembly, err := c.Assemble(t.Context(), []ast.Node{parsedLoad, ast.NewLabel("target")})
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xad, 0x03, 0x00}, builtAssembly.Binary)
	assert.Equal(t, builtAssembly.Binary, parsedAssembly.Binary)

	expression := ast.NewExpression(
		token.Token{Type: token.LeftParentheses, Value: "("},
		token.Token{Type: token.Identifier, Value: "value"},
		token.Token{Type: token.Minus, Value: "-"},
		token.Token{Type: token.Number, Value: "1"},
		token.Token{Type: token.RightParentheses, Value: ")"},
	)
	immediate, err := codec.BuildInstruction(
		c,
		cpu6502.LdaName,
		cpu6502parser.Operands{cpu6502parser.ImmediateOperand(expression)},
	)
	assert.NoError(t, err)
	formattedImmediate, err := c.FormatInstruction(immediate)
	assert.NoError(t, err)
	assert.Equal(t, "lda #(value-0x1)", formattedImmediate)
	parsedImmediate, err := c.ParseInstruction(t.Context(), strings.NewReader(formattedImmediate))
	assert.NoError(t, err)
	assert.NoError(t, c.ValidateInstruction(parsedImmediate))
}

func TestCPU6502Codec_CompatibilityProjectionPreservesModifiers(t *testing.T) {
	t.Parallel()

	c := newCPU6502AssemblyCodec(t)
	modifier := ast.Modifier{Operator: ast.NewOperator("+"), Value: "2"}
	operand := cpu6502parser.WithModifiers(
		cpu6502parser.MemoryOperand(
			cpu6502parser.OperandAddress,
			cpu6502parser.AddressAbsolute,
			ast.NewNumber(0x1200),
		),
		modifier,
	)
	instruction, err := codec.BuildInstruction(c, cpu6502.LdaName, cpu6502parser.Operands{operand})
	assert.NoError(t, err)
	assert.Len(t, instruction.Modifier, 1)
	_, ok := ast.NumberValue(instruction.Argument)
	assert.True(t, ok)

	resolved, err := cpu6502parser.ResolveInstruction(instruction, cpu6502.LdaInst)
	assert.NoError(t, err)
	assert.Equal(t, cpu6502parser.OperandAddress, resolved.Operands[0].Kind)
	assert.Equal(t, "+", resolved.Operands[0].Modifiers[0].Operator.Operator)

	formatted, err := c.FormatInstruction(instruction)
	assert.NoError(t, err)
	assert.Equal(t, "lda a:$1200+$2", formatted)
	parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
	assert.NoError(t, err)
	assert.NoError(t, c.ValidateInstruction(parsed))
}

func TestCPU6502Codec_RejectsInvalidTypedOperands(t *testing.T) {
	t.Parallel()

	c := newCPU6502AssemblyCodec(t)
	tests := []struct {
		mnemonic string
		operands cpu6502parser.Operands
	}{
		{
			mnemonic: cpu6502.LdaName,
			operands: cpu6502parser.Operands{cpu6502parser.ImmediateOperand(ast.NewNumber(0x100))},
		},
		{
			mnemonic: cpu6502.LdaName,
			operands: cpu6502parser.Operands{cpu6502parser.MemoryOperand(
				cpu6502parser.OperandAddress, cpu6502parser.AddressZeroPage, ast.NewNumber(0x100),
			)},
		},
		{
			mnemonic: cpu6502.JmpName,
			operands: cpu6502parser.Operands{cpu6502parser.MemoryOperand(
				cpu6502parser.OperandIndexedX, cpu6502parser.AddressAbsolute, ast.NewNumber(0x1000),
			)},
		},
	}
	for _, test := range tests {
		_, err := codec.BuildInstruction(c, test.mnemonic, test.operands)
		assert.Error(t, err)
	}
}

func TestCPU6502TypedInstructionFormattingOptions(t *testing.T) {
	t.Parallel()

	c := newCPU6502AssemblyCodec(t)
	instruction, err := codec.BuildInstruction(
		c,
		cpu6502.LdaName,
		cpu6502parser.Operands{cpu6502parser.MemoryOperand(
			cpu6502parser.OperandIndexedX,
			cpu6502parser.AddressAbsolute,
			ast.NewNumber(0x42),
		)},
	)
	assert.NoError(t, err)
	formatted, err := cpu6502parser.FormatInstructionWithOptions(
		instruction,
		cpu6502.LdaInst,
		cpu6502parser.FormatOptions{Indent: "  ", Uppercase: true, WordHexDigits: 6},
	)
	assert.NoError(t, err)
	assert.Equal(t, "  LDA A:$000042,X", formatted)
}

func newCPU6502AssemblyCodec(t *testing.T) *codec.Codec[*cpu6502.Instruction] {
	t.Helper()
	return newCPU6502AssemblyCodecWithConfig(t, asmcpu6502.New())
}

func newCPU6502AssemblyCodecForVariant(
	t *testing.T,
	variant cpu6502.CPUVariant,
) *codec.Codec[*cpu6502.Instruction] {

	t.Helper()
	return newCPU6502AssemblyCodecWithConfig(t, asmcpu6502.New(asmcpu6502.WithVariant(variant)))
}

func newCPU6502AssemblyCodecWithConfig(
	t *testing.T,
	configuration *config.Config[*cpu6502.Instruction],
) *codec.Codec[*cpu6502.Instruction] {

	t.Helper()
	segment := &config.Segment{
		Memory:      config.Memory{Name: "code", Start: 0, Size: 0x10000},
		SegmentName: "code",
	}
	configuration.Segments = map[string]*config.Segment{"code": segment}
	configuration.SegmentsOrdered = []*config.Segment{segment}
	c, err := codec.New(configuration)
	assert.NoError(t, err)
	return c
}
