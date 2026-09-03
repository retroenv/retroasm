package codec_test

import (
	"strings"
	"testing"

	asmsm83 "github.com/retroenv/retroasm/pkg/arch/sm83"
	sm83parser "github.com/retroenv/retroasm/pkg/arch/sm83/parser"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/arch/cpu/sm83"
	"github.com/retroenv/retrogolib/assert"
)

func TestSM83Codec_BuildValidateFormatAndAssemble(t *testing.T) {
	t.Parallel()

	c := newSM83Codec(t)
	built, err := codec.BuildInstruction(
		c,
		sm83.LdName,
		sm83parser.Operands{
			sm83parser.RegisterOperand(sm83.RegA),
			sm83parser.ValueOperand(ast.NewNumber(0x12)),
		},
	)
	assert.NoError(t, err)
	assert.True(t, built.OpcodeID.ValidFor(arch.SM83))
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

func TestSM83Codec_RecordsInstructionRelocations(t *testing.T) {
	t.Parallel()

	c := newSM83Codec(t)
	stream, err := c.ParseStream(t.Context(), "input.asm", strings.NewReader(strings.Join([]string{
		"target:",
		"ld a,target",
		"ld bc,target+1",
		"jp target+2",
		"jr target",
		"ld (hl),target+3",
		"ldh (target),a",
	}, "\n")))
	assert.NoError(t, err)
	assert.Empty(t, stream.Relocations())

	assembly, err := c.AssembleStream(t.Context(), stream)
	assert.NoError(t, err)
	assert.Equal(t, []byte{
		0x3e, 0x00,
		0x01, 0x01, 0x00,
		0xc3, 0x02, 0x00,
		0x18, 0xf6,
		0x36, 0x03,
		0xe0, 0x00,
	}, assembly.Binary)
	assert.Equal(t, []ast.Relocation{
		{EntryIndex: 1, ByteOffset: 1, Kind: ast.AbsoluteRelocation, Expression: ast.NewSymbolExpression("target", 0, ast.FullAddress), Width: ast.WidthByte, ByteOrder: ast.ByteOrderLittle},
		{EntryIndex: 2, ByteOffset: 1, Kind: ast.AbsoluteRelocation, Expression: ast.NewSymbolExpression("target", 1, ast.FullAddress), Width: ast.WidthWord, ByteOrder: ast.ByteOrderLittle},
		{EntryIndex: 3, ByteOffset: 1, Kind: ast.AbsoluteRelocation, Expression: ast.NewSymbolExpression("target", 2, ast.FullAddress), Width: ast.WidthWord, ByteOrder: ast.ByteOrderLittle},
		{EntryIndex: 4, ByteOffset: 1, Kind: ast.RelativeRelocation, Expression: ast.NewSymbolExpression("target", 0, ast.FullAddress), Width: ast.WidthByte, ByteOrder: ast.ByteOrderLittle},
		{EntryIndex: 5, ByteOffset: 1, Kind: ast.AbsoluteRelocation, Expression: ast.NewSymbolExpression("target", 3, ast.FullAddress), Width: ast.WidthByte, ByteOrder: ast.ByteOrderLittle},
		{EntryIndex: 6, ByteOffset: 1, Kind: ast.AbsoluteRelocation, Expression: ast.NewSymbolExpression("target", 0, ast.FullAddress), Width: ast.WidthByte, ByteOrder: ast.ByteOrderLittle},
	}, assembly.Stream.Relocations())
	assert.NoError(t, assembly.Stream.Validate())

	reassembled, err := c.AssembleStream(t.Context(), assembly.Stream)
	assert.NoError(t, err)
	assert.Equal(t, assembly.Binary, reassembled.Binary)
	assert.Equal(t, assembly.Stream.Relocations(), reassembled.Stream.Relocations())
}

//nolint:funlen // Emitted-form coverage is intentionally kept in one auditable table.
func TestSM83Codec_CurrentEmittedFormsRoundTrip(t *testing.T) {
	t.Parallel()

	c := newSM83Codec(t)
	tests := []struct {
		name     string
		mnemonic string
		operands sm83parser.Operands
		wantText string
		wantCode []byte
	}{
		{name: "implied", mnemonic: sm83.NopName, wantText: "nop", wantCode: []byte{0x00}},
		{
			name: "register load", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegA),
				sm83parser.RegisterOperand(sm83.RegB),
			},
			wantText: "ld a,b", wantCode: []byte{0x78},
		},
		{
			name: "ambiguous C register", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegC),
				sm83parser.RegisterOperand(sm83.RegA),
			},
			wantText: "ld c,a", wantCode: []byte{0x4f},
		},
		{
			name: "pair immediate", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegHL),
				sm83parser.ValueOperand(ast.NewNumber(0x1234)),
			},
			wantText: "ld hl,0x1234", wantCode: []byte{0x21, 0x34, 0x12},
		},
		{
			name: "indirect load", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegA),
				sm83parser.IndirectRegisterOperand(sm83.RegBC),
			},
			wantText: "ld a,(bc)", wantCode: []byte{0x0a},
		},
		{
			name: "indirect store", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.IndirectRegisterOperand(sm83.RegHL),
				sm83parser.RegisterOperand(sm83.RegA),
			},
			wantText: "ld (hl),a", wantCode: []byte{0x77},
		},
		{
			name: "indirect immediate", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.IndirectRegisterOperand(sm83.RegHL),
				sm83parser.ValueOperand(ast.NewNumber(0x2a)),
			},
			wantText: "ld (hl),0x2A", wantCode: []byte{0x36, 0x2a},
		},
		{
			name: "absolute store", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.IndirectValueOperand(ast.NewNumber(0xc123)),
				sm83parser.RegisterOperand(sm83.RegA),
			},
			wantText: "ld (0xC123),a", wantCode: []byte{0xea, 0x23, 0xc1},
		},
		{
			name: "absolute load", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegA),
				sm83parser.IndirectValueOperand(ast.NewNumber(0xc123)),
			},
			wantText: "ld a,(0xC123)", wantCode: []byte{0xfa, 0x23, 0xc1},
		},
		{
			name: "absolute stack store", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.IndirectValueOperand(ast.NewNumber(0xc123)),
				sm83parser.RegisterOperand(sm83.RegSP),
			},
			wantText: "ld (0xC123),sp", wantCode: []byte{0x08, 0x23, 0xc1},
		},
		{
			name: "post increment", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegA),
				sm83parser.HLIncrementOperand(),
			},
			wantText: "ld a,(hl+)", wantCode: []byte{0x2a},
		},
		{
			name: "post decrement", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.HLDecrementOperand(),
				sm83parser.RegisterOperand(sm83.RegA),
			},
			wantText: "ld (hl-),a", wantCode: []byte{0x32},
		},
		{
			name: "stack pair copy", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegSP),
				sm83parser.RegisterOperand(sm83.RegHL),
			},
			wantText: "ld sp,hl", wantCode: []byte{0xf9},
		},
		{
			name: "stack offset load", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegHL),
				sm83parser.SPOffsetOperand(ast.NewNumber(0xfc)),
			},
			wantText: "ld hl,sp-0x4", wantCode: []byte{0xf8, 0xfc},
		},
		{
			name: "stack offset add", mnemonic: sm83.AddName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegSP),
				sm83parser.ValueOperand(ast.NewNumber(0xfc)),
			},
			wantText: "add sp,-0x4", wantCode: []byte{0xe8, 0xfc},
		},
		{
			name: "high memory store", mnemonic: sm83.LdhName,
			operands: sm83parser.Operands{
				sm83parser.IndirectValueOperand(ast.NewNumber(0x12)),
				sm83parser.RegisterOperand(sm83.RegA),
			},
			wantText: "ldh (0x12),a", wantCode: []byte{0xe0, 0x12},
		},
		{
			name: "high memory load", mnemonic: sm83.LdhName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegA),
				sm83parser.IndirectValueOperand(ast.NewNumber(0x12)),
			},
			wantText: "ldh a,(0x12)", wantCode: []byte{0xf0, 0x12},
		},
		{
			name: "high C store", mnemonic: sm83.LdhName,
			operands: sm83parser.Operands{
				sm83parser.IndirectRegisterOperand(sm83.RegC),
				sm83parser.RegisterOperand(sm83.RegA),
			},
			wantText: "ldh (c),a", wantCode: []byte{0xe2},
		},
		{
			name: "high C load", mnemonic: sm83.LdhName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegA),
				sm83parser.IndirectRegisterOperand(sm83.RegC),
			},
			wantText: "ldh a,(c)", wantCode: []byte{0xf2},
		},
		{
			name: "conditional jump", mnemonic: sm83.JpName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegCondNZ),
				sm83parser.ValueOperand(ast.NewNumber(0x1234)),
			},
			wantText: "jp nz,0x1234", wantCode: []byte{0xc2, 0x34, 0x12},
		},
		{
			name: "carry return", mnemonic: sm83.RetName,
			operands: sm83parser.Operands{sm83parser.RegisterOperand(sm83.RegCondC)},
			wantText: "ret c", wantCode: []byte{0xd8},
		},
		{
			name: "restart", mnemonic: sm83.RstName,
			operands: sm83parser.Operands{sm83parser.ValueOperand(ast.NewNumber(0x18))},
			wantText: "rst 0x18", wantCode: []byte{0xdf},
		},
		{
			name: "bit memory", mnemonic: sm83.BitName,
			operands: sm83parser.Operands{
				sm83parser.ValueOperand(ast.NewNumber(3)),
				sm83parser.IndirectRegisterOperand(sm83.RegHL),
			},
			wantText: "bit 0x3,(hl)", wantCode: []byte{0xcb, 0x5e},
		},
		{
			name: "swap", mnemonic: sm83.SwapName,
			operands: sm83parser.Operands{sm83parser.RegisterOperand(sm83.RegA)},
			wantText: "swap a", wantCode: []byte{0xcb, 0x37},
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

			builtAssembly, err := c.Assemble(t.Context(), []ast.Node{built})
			assert.NoError(t, err)
			parsedAssembly, err := c.Assemble(t.Context(), []ast.Node{parsed})
			assert.NoError(t, err)
			assert.Equal(t, test.wantCode, builtAssembly.Binary)
			assert.Equal(t, builtAssembly.Binary, parsedAssembly.Binary)
		})
	}
}

//nolint:funlen // Formatting policies are clearer as one compact table.
func TestSM83TypedInstructionFormattingOptions(t *testing.T) {
	t.Parallel()

	c := newSM83Codec(t)
	options := sm83parser.FormatOptions{
		Indent:                 "  ",
		Uppercase:              true,
		MinimumHexDigits:       2,
		PairImmediateHexDigits: 4,
		DecimalBitIndexes:      true,
		DecimalSignedOffsets:   true,
	}
	tests := []struct {
		name     string
		mnemonic string
		operands sm83parser.Operands
		want     string
	}{
		{
			name: "byte immediate", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegA),
				sm83parser.ValueOperand(ast.NewNumber(5)),
			},
			want: "  LD A,0x05",
		},
		{
			name: "pair immediate", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegHL),
				sm83parser.ValueOperand(ast.NewNumber(4)),
			},
			want: "  LD HL,0x0004",
		},
		{
			name: "bit index", mnemonic: sm83.BitName,
			operands: sm83parser.Operands{
				sm83parser.ValueOperand(ast.NewNumber(3)),
				sm83parser.RegisterOperand(sm83.RegA),
			},
			want: "  BIT 3,A",
		},
		{
			name: "stack offset", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegHL),
				sm83parser.SPOffsetOperand(ast.NewNumber(0xfc)),
			},
			want: "  LD HL,SP-4",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			instruction, err := codec.BuildInstruction(c, test.mnemonic, test.operands)
			assert.NoError(t, err)

			formatted, err := sm83parser.FormatInstructionWithOptions(instruction, options)

			assert.NoError(t, err)
			assert.Equal(t, test.want, formatted)
		})
	}
}

func TestSM83Codec_RejectsInvalidTypedState(t *testing.T) {
	t.Parallel()

	c := newSM83Codec(t)
	tests := []struct {
		name     string
		mnemonic string
		operands sm83parser.Operands
	}{
		{
			name: "invalid absolute register", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegB),
				sm83parser.IndirectValueOperand(ast.NewNumber(0x1234)),
			},
		},
		{
			name: "wide immediate", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegA),
				sm83parser.ValueOperand(ast.NewNumber(0x100)),
			},
		},
		{
			name: "invalid bit", mnemonic: sm83.BitName,
			operands: sm83parser.Operands{
				sm83parser.ValueOperand(ast.NewNumber(8)),
				sm83parser.RegisterOperand(sm83.RegA),
			},
		},
		{
			name: "symbolic stack offset", mnemonic: sm83.LdName,
			operands: sm83parser.Operands{
				sm83parser.RegisterOperand(sm83.RegHL),
				sm83parser.SPOffsetOperand(ast.NewLabel("offset")),
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

	valid, err := codec.BuildInstruction(c, sm83.NopName, sm83parser.Operands{})
	assert.NoError(t, err)
	valid.OpcodeID = ast.NewOpcodeID(arch.Z80, valid.OpcodeID.Value)
	assert.Error(t, c.ValidateInstruction(valid))
}

func TestSM83Codec_AmbiguousCUsesInstructionContext(t *testing.T) {
	t.Parallel()

	c := newSM83Codec(t)
	register, err := c.ParseInstruction(t.Context(), strings.NewReader("ld c,a"))
	assert.NoError(t, err)
	assert.NoError(t, c.ValidateInstruction(register))
	formattedRegister, err := c.FormatInstruction(register)
	assert.NoError(t, err)
	assert.Equal(t, "ld c,a", formattedRegister)

	condition, err := c.ParseInstruction(t.Context(), strings.NewReader("ret c"))
	assert.NoError(t, err)
	assert.NoError(t, c.ValidateInstruction(condition))
	formattedCondition, err := c.FormatInstruction(condition)
	assert.NoError(t, err)
	assert.Equal(t, "ret c", formattedCondition)

	label, err := c.ParseInstruction(t.Context(), strings.NewReader("jp c"))
	assert.NoError(t, err)
	assert.NoError(t, c.ValidateInstruction(label))
	formattedLabel, err := c.FormatInstruction(label)
	assert.NoError(t, err)
	assert.Equal(t, "jp c", formattedLabel)
}

func newSM83Codec(t *testing.T) *codec.Codec[*asmsm83.InstructionGroup] {
	t.Helper()
	configuration := asmsm83.New()
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
