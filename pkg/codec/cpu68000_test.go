package codec_test

import (
	"strings"
	"testing"

	asmcpu68000 "github.com/retroenv/retroasm/pkg/arch/cpu68000"
	cpu68000parser "github.com/retroenv/retroasm/pkg/arch/cpu68000/parser"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
	"github.com/retroenv/retrogolib/assert"
)

func TestCPU68000Codec_BuildValidateFormatAndAssemble(t *testing.T) {
	t.Parallel()

	c := newCPU68000Codec(t)
	built, err := codec.BuildInstruction(
		c,
		"move.l",
		cpu68000parser.BinaryOperands(
			cpu68000.SizeLong,
			cpu68000parser.DataRegister(0),
			cpu68000parser.DataRegister(1),
		),
	)
	assert.NoError(t, err)
	assert.True(t, built.OpcodeID.ValidFor(arch.CPU68000))
	assert.NoError(t, c.ValidateInstruction(built))

	formatted, err := c.FormatInstruction(built)
	assert.NoError(t, err)
	assert.Equal(t, "move.l d0,d1", formatted)

	parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
	assert.NoError(t, err)
	assert.NoError(t, c.ValidateInstruction(parsed))

	builtAssembly, err := c.Assemble(t.Context(), []ast.Node{built})
	assert.NoError(t, err)
	parsedAssembly, err := c.Assemble(t.Context(), []ast.Node{parsed})
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x22, 0x00}, builtAssembly.Binary)
	assert.Equal(t, builtAssembly.Binary, parsedAssembly.Binary)
}

//nolint:funlen // Effective-address coverage is intentionally one auditable table.
func TestCPU68000Codec_EffectiveAddressFamiliesRoundTrip(t *testing.T) {
	t.Parallel()

	c := newCPU68000Codec(t)
	tests := []struct {
		name     string
		mnemonic string
		operands cpu68000parser.Operands
		wantText string
	}{
		{name: "implied", mnemonic: cpu68000.NOPName, wantText: "nop"},
		{
			name: "immediate", mnemonic: "move.w",
			operands: cpu68000parser.BinaryOperands(
				cpu68000.SizeWord,
				cpu68000parser.Immediate(ast.NewNumber(0x1234)),
				cpu68000parser.DataRegister(0),
			),
			wantText: "move.w #0x1234,d0",
		},
		{
			name: "address register", mnemonic: "movea.l",
			operands: cpu68000parser.BinaryOperands(
				cpu68000.SizeLong,
				cpu68000parser.DataRegister(0),
				cpu68000parser.AddressRegister(1),
			),
			wantText: "movea.l d0,a1",
		},
		{
			name: "indirect", mnemonic: "move.w",
			operands: cpu68000parser.BinaryOperands(
				cpu68000.SizeWord,
				cpu68000parser.Indirect(0),
				cpu68000parser.DataRegister(1),
			),
			wantText: "move.w (a0),d1",
		},
		{
			name: "post increment", mnemonic: "move.w",
			operands: cpu68000parser.BinaryOperands(
				cpu68000.SizeWord,
				cpu68000parser.PostIncrement(0),
				cpu68000parser.DataRegister(1),
			),
			wantText: "move.w (a0)+,d1",
		},
		{
			name: "pre decrement", mnemonic: "move.w",
			operands: cpu68000parser.BinaryOperands(
				cpu68000.SizeWord,
				cpu68000parser.DataRegister(0),
				cpu68000parser.PreDecrement(1),
			),
			wantText: "move.w d0,-(a1)",
		},
		{
			name: "displacement", mnemonic: "move.w",
			operands: cpu68000parser.BinaryOperands(
				cpu68000.SizeWord,
				cpu68000parser.Displacement(0, ast.NewNumber(0x10)),
				cpu68000parser.DataRegister(1),
			),
			wantText: "move.w 0x0010(a0),d1",
		},
		{
			name: "indexed", mnemonic: "move.w",
			operands: cpu68000parser.BinaryOperands(
				cpu68000.SizeWord,
				cpu68000parser.Indexed(0, 1, false, cpu68000.SizeLong, ast.NewNumber(0x10)),
				cpu68000parser.DataRegister(2),
			),
			wantText: "move.w 0x10(a0,d1.l),d2",
		},
		{
			name: "absolute short", mnemonic: "clr.w",
			operands: cpu68000parser.UnaryOperands(
				cpu68000.SizeWord,
				cpu68000parser.Absolute(false, ast.NewNumber(0x1234)),
			),
			wantText: "clr.w 0x1234.w",
		},
		{
			name: "absolute long", mnemonic: "clr.l",
			operands: cpu68000parser.UnaryOperands(
				cpu68000.SizeLong,
				cpu68000parser.Absolute(true, ast.NewNumber(0x123456)),
			),
			wantText: "clr.l 0x00123456.l",
		},
		{
			name: "pc displacement", mnemonic: "lea.l",
			operands: cpu68000parser.BinaryOperands(
				cpu68000.SizeLong,
				cpu68000parser.PCDisplacement(ast.NewLabel("table")),
				cpu68000parser.AddressRegister(0),
			),
			wantText: "lea.l table(pc),a0",
		},
		{
			name: "pc indexed", mnemonic: "lea.l",
			operands: cpu68000parser.BinaryOperands(
				cpu68000.SizeLong,
				cpu68000parser.PCIndexed(1, true, cpu68000.SizeWord, ast.NewNumber(4)),
				cpu68000parser.AddressRegister(0),
			),
			wantText: "lea.l 0x04(pc,a1.w),a0",
		},
		{
			name: "quick immediate", mnemonic: "addq.w",
			operands: cpu68000parser.BinaryOperands(
				cpu68000.SizeWord,
				cpu68000parser.Immediate(ast.NewNumber(1)),
				cpu68000parser.DataRegister(0),
			),
			wantText: "addq.w #0x01,d0",
		},
		{
			name: "register list to memory", mnemonic: "movem.w",
			operands: cpu68000parser.BinaryOperands(
				cpu68000.SizeWord,
				cpu68000parser.RegisterList(0x0103),
				cpu68000parser.Indirect(0),
			),
			wantText: "movem.w d0/d1/a0,(a0)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			built, err := codec.BuildInstruction(c, test.mnemonic, test.operands)
			assert.NoError(t, err)
			assert.NoError(t, c.ValidateInstruction(built))

			formatted, err := c.FormatInstruction(built)
			assert.NoError(t, err)
			assert.Equal(t, test.wantText, formatted)

			parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
			assert.NoError(t, err)
			assert.NoError(t, c.ValidateInstruction(parsed))
			if !strings.Contains(test.wantText, "table") {
				builtAssembly, err := c.Assemble(t.Context(), []ast.Node{built})
				assert.NoError(t, err)
				parsedAssembly, err := c.Assemble(t.Context(), []ast.Node{parsed})
				assert.NoError(t, err)
				assert.Equal(t, builtAssembly.Binary, parsedAssembly.Binary)
			}
		})
	}
}

func TestCPU68000Codec_ConditionCodesRoundTrip(t *testing.T) {
	t.Parallel()

	c := newCPU68000Codec(t)
	tests := []struct {
		mnemonic string
		operands cpu68000parser.Operands
		want     string
	}{
		{
			mnemonic: "beq.w",
			operands: cpu68000parser.UnaryOperands(
				cpu68000.SizeWord,
				cpu68000parser.PCDisplacement(ast.NewLabel("loop")),
			),
			want: "beq.w loop(pc)",
		},
		{
			mnemonic: "dbne",
			operands: cpu68000parser.BinaryOperands(
				cpu68000.SizeWord,
				cpu68000parser.DataRegister(0),
				cpu68000parser.PCDisplacement(ast.NewLabel("loop")),
			),
			want: "dbne d0,loop(pc)",
		},
		{
			mnemonic: "seq",
			operands: cpu68000parser.UnaryOperands(
				cpu68000.SizeByte,
				cpu68000parser.DataRegister(0),
			),
			want: "seq d0",
		},
	}

	for _, test := range tests {
		built, err := codec.BuildInstruction(c, test.mnemonic, test.operands)
		assert.NoError(t, err)
		formatted, err := c.FormatInstruction(built)
		assert.NoError(t, err)
		assert.Equal(t, test.want, formatted)
		parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
		assert.NoError(t, err)
		assert.NoError(t, c.ValidateInstruction(parsed))
	}
}

func TestCPU68000Codec_RejectsInvalidTypedOperands(t *testing.T) {
	t.Parallel()

	c := newCPU68000Codec(t)
	tests := []struct {
		mnemonic string
		operands cpu68000parser.Operands
	}{
		{
			mnemonic: "move.w",
			operands: cpu68000parser.BinaryOperands(
				cpu68000.SizeWord,
				cpu68000parser.DataRegister(8),
				cpu68000parser.DataRegister(0),
			),
		},
		{
			mnemonic: "addq.w",
			operands: cpu68000parser.BinaryOperands(
				cpu68000.SizeWord,
				cpu68000parser.Immediate(ast.NewNumber(9)),
				cpu68000parser.DataRegister(0),
			),
		},
		{
			mnemonic: "trap",
			operands: cpu68000parser.Operands{
				Source: cpu68000parser.Immediate(ast.NewNumber(16)),
			},
		},
		{
			mnemonic: "movem.w",
			operands: cpu68000parser.BinaryOperands(
				cpu68000.SizeWord,
				cpu68000parser.RegisterList(1),
				cpu68000parser.RegisterList(2),
			),
		},
	}

	for _, test := range tests {
		_, err := codec.BuildInstruction(c, test.mnemonic, test.operands)
		assert.Error(t, err)
	}
}

func TestCPU68000TypedInstructionFormattingOptions(t *testing.T) {
	t.Parallel()

	c := newCPU68000Codec(t)
	instruction, err := codec.BuildInstruction(
		c,
		"move.l",
		cpu68000parser.BinaryOperands(
			cpu68000.SizeLong,
			cpu68000parser.Absolute(true, ast.NewNumber(0x42)),
			cpu68000parser.DataRegister(0),
		),
	)
	assert.NoError(t, err)

	formatted, err := cpu68000parser.FormatInstructionWithOptions(
		instruction,
		cpu68000parser.FormatOptions{Indent: "  ", Uppercase: true},
	)
	assert.NoError(t, err)
	assert.Equal(t, "  MOVE.L 0x00000042.l,D0", formatted)
}

func newCPU68000Codec(t *testing.T) *codec.Codec[*cpu68000.Instruction] {
	t.Helper()
	configuration := asmcpu68000.New()
	segment := &config.Segment{
		Memory: config.Memory{
			Name:  "code",
			Start: 0,
			Size:  0x1000000,
		},
		SegmentName: "code",
	}
	configuration.Segments = map[string]*config.Segment{"code": segment}
	configuration.SegmentsOrdered = []*config.Segment{segment}

	c, err := codec.New(configuration)
	assert.NoError(t, err)
	return c
}
