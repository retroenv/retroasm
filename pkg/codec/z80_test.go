package codec_test

import (
	"strings"
	"testing"

	asmz80 "github.com/retroenv/retroasm/pkg/arch/z80"
	z80parser "github.com/retroenv/retroasm/pkg/arch/z80/parser"
	z80profile "github.com/retroenv/retroasm/pkg/arch/z80/profile"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/assert"
)

func TestZ80Codec_BuildValidateFormatAndAssemble(t *testing.T) {
	t.Parallel()

	c := newZ80Codec(t)
	built, err := codec.BuildInstruction(
		c,
		cpuz80.LdName,
		z80parser.Operands{
			z80parser.RegisterOperand(cpuz80.RegA),
			z80parser.ValueOperand(ast.NewNumber(0x12)),
		},
	)
	assert.NoError(t, err)
	assert.True(t, built.OpcodeID.ValidFor(arch.Z80))
	assert.NoError(t, c.ValidateInstruction(built))

	formatted, err := c.FormatInstruction(built)
	assert.NoError(t, err)
	assert.Equal(t, "ld a,0x12", formatted)

	parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
	assert.NoError(t, err)
	assert.NoError(t, c.ValidateInstruction(parsed))

	builtAssembly, err := c.Assemble(t.Context(), []ast.Node{built})
	assert.NoError(t, err)
	parsedAssembly, err := c.Assemble(t.Context(), []ast.Node{parsed})
	assert.NoError(t, err)
	assert.Equal(t, builtAssembly.Binary, parsedAssembly.Binary)
}

func TestZ80Codec_ParsedInstructionRoundTrips(t *testing.T) {
	t.Parallel()

	c := newZ80Codec(t)

	sources := []string{
		"nop",
		"ld a,b",
		"ld hl,0x1234",
		"jr nz,0x2",
		"jp (hl)",
		"ld a,(bc)",
		"ld (hl),a",
		"ld (hl),42",
		"ld a,(0x1234)",
		"ld (0x1234),a",
		"ld a,(ix-3)",
		"ld (iy+4),a",
		"ld (ix+5),7",
		"bit 3,(iy-2)",
		"in a,(3)",
		"in b,(c)",
		"out (c),a",
		"ex (sp),ix",
		"rst 0x38",
		"im 2",
		"ld i,a",
		"ld sp,iy",
		"ex de,hl",
	}

	for _, source := range sources {
		t.Run(source, func(t *testing.T) {
			t.Parallel()

			parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(source))
			assert.NoError(t, err)
			assert.NoError(t, c.ValidateInstruction(parsed))

			formatted, err := c.FormatInstruction(parsed)
			assert.NoError(t, err)
			roundTripped, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
			assert.NoError(t, err)
			assert.NoError(t, c.ValidateInstruction(roundTripped))

			originalAssembly, err := c.Assemble(t.Context(), []ast.Node{parsed})
			assert.NoError(t, err)
			roundTripAssembly, err := c.Assemble(t.Context(), []ast.Node{roundTripped})
			assert.NoError(t, err)
			assert.Equal(t, originalAssembly.Binary, roundTripAssembly.Binary)
		})
	}
}

func TestZ80Codec_SymbolAndConditionRoundTrips(t *testing.T) {
	t.Parallel()

	c := newZ80Codec(t)
	sources := []string{
		"jp c",
		"jp hl",
		"jp c,target",
		"jr nz,target",
		"ld a,(table+index)",
		"ld a,(ix+offset)",
		"ld a,(iy - offset)",
	}

	for _, source := range sources {
		t.Run(source, func(t *testing.T) {
			t.Parallel()

			parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(source))
			assert.NoError(t, err)
			assert.NoError(t, c.ValidateInstruction(parsed))
			formatted, err := c.FormatInstruction(parsed)
			assert.NoError(t, err)
			roundTripped, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
			assert.NoError(t, err)
			assert.NoError(t, c.ValidateInstruction(roundTripped))
		})
	}
}

func TestZ80Codec_RejectsProfileAndTypedMetadataMismatches(t *testing.T) {
	t.Parallel()

	strict := newZ80Codec(t, asmz80.WithProfile(z80profile.StrictDocumented))
	_, err := codec.BuildInstruction(strict, cpuz80.INF.Name, z80parser.Operands{})
	assert.Error(t, err)

	defaultCodec := newZ80Codec(t)
	instruction, err := codec.BuildInstruction(defaultCodec, cpuz80.NopName, z80parser.Operands{})
	assert.NoError(t, err)
	instruction.OpcodeID = ast.NewOpcodeID(arch.CPU6502, 1)
	assert.Error(t, defaultCodec.ValidateInstruction(instruction))
}

func TestZ80Codec_RejectsOutOfWidthTypedOperands(t *testing.T) {
	t.Parallel()

	c := newZ80Codec(t)
	tests := []struct {
		name     string
		mnemonic string
		operands z80parser.Operands
	}{
		{
			name: "byte immediate", mnemonic: cpuz80.LdName,
			operands: z80parser.Operands{
				z80parser.RegisterOperand(cpuz80.RegA),
				z80parser.ValueOperand(ast.NewNumber(0x100)),
			},
		},
		{
			name: "bit number", mnemonic: cpuz80.BitName,
			operands: z80parser.Operands{
				z80parser.ValueOperand(ast.NewNumber(8)),
				z80parser.RegisterOperand(cpuz80.RegA),
			},
		},
		{
			name: "port", mnemonic: cpuz80.InName,
			operands: z80parser.Operands{
				z80parser.RegisterOperand(cpuz80.RegA),
				z80parser.IndirectValueOperand(ast.NewNumber(0x100)),
			},
		},
		{
			name: "indexed displacement", mnemonic: cpuz80.LdName,
			operands: z80parser.Operands{
				z80parser.RegisterOperand(cpuz80.RegA),
				z80parser.IndexedOperand(cpuz80.RegIX, ast.NewNumber(0x100)),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := codec.BuildInstruction(c, test.mnemonic, test.operands)
			assert.Error(t, err)
		})
	}
}

func TestZ80Codec_RejectsInvalidTextRegisterCombinations(t *testing.T) {
	t.Parallel()

	c := newZ80Codec(t)
	for _, source := range []string{"ld a,sp", "bit c,a"} {
		t.Run(source, func(t *testing.T) {
			t.Parallel()

			_, err := c.ParseInstruction(t.Context(), strings.NewReader(source))
			assert.Error(t, err)
		})
	}
}

func TestCodec_BuildUnsupportedArchitecture(t *testing.T) {
	t.Parallel()

	c := newCPU6502Codec(t)
	_, err := codec.BuildInstruction(c, cpuz80.NopName, z80parser.Operand{})
	assert.ErrorIs(t, err, codec.ErrBuildUnsupported)
}

func newZ80Codec(t *testing.T, opts ...asmz80.Option) *codec.Codec[*asmz80.InstructionGroup] {
	t.Helper()

	configuration := asmz80.New(opts...)
	segment := &config.Segment{
		Memory: config.Memory{
			Name:  "code",
			Start: 0,
			Size:  0x10000,
		},
		SegmentName: "code",
	}
	configuration.Segments = map[string]*config.Segment{"code": segment}
	configuration.SegmentsOrdered = []*config.Segment{segment}

	c, err := codec.New(configuration)
	assert.NoError(t, err)
	return c
}
