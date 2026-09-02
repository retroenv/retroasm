package codec_test

import (
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
	configuration := asmcpu6502.New()
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
