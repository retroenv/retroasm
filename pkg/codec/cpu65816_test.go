package codec_test

import (
	"strings"
	"testing"

	asmcpu65816 "github.com/retroenv/retroasm/pkg/arch/cpu65816"
	cpu65816parser "github.com/retroenv/retroasm/pkg/arch/cpu65816/parser"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/arch/cpu/cpu65816"
	"github.com/retroenv/retrogolib/assert"
)

func TestCPU65816Codec_BuildValidateFormatAndAssemble(t *testing.T) {
	t.Parallel()

	c := newCPU65816Codec(t)
	built, err := codec.BuildInstruction(
		c,
		cpu65816.LdaName,
		cpu65816parser.Operands{cpu65816parser.ImmediateOperand(ast.NewNumber(0x12))},
	)
	assert.NoError(t, err)
	assert.True(t, built.OpcodeID.ValidFor(arch.CPU65816))
	assert.NoError(t, c.ValidateInstruction(built))

	formatted, err := c.FormatInstruction(built)
	assert.NoError(t, err)
	assert.Equal(t, "lda #$12", formatted)

	parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
	assert.NoError(t, err)
	assert.NoError(t, c.ValidateInstruction(parsed))

	builtAssembly, err := c.Assemble(t.Context(), []ast.Node{built})
	assert.NoError(t, err)
	parsedAssembly, err := c.Assemble(t.Context(), []ast.Node{parsed})
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xa9, 0x12}, builtAssembly.Binary)
	assert.Equal(t, builtAssembly.Binary, parsedAssembly.Binary)
}

func TestCPU65816Codec_FormatStreamRoundTripsX816DataWidths(t *testing.T) {
	t.Parallel()

	configuration := asmcpu65816.New()
	configuration.CompatibilityMode = config.CompatX816
	c := newCPU65816CodecWithConfig(t, configuration)
	stream, err := c.ParseStream(
		t.Context(),
		"input.asm",
		strings.NewReader(".dcl $123456,1+2\n.dcd $12345678\n.dsl 2,$abcdef\n.dsd 2,$12345678"),
	)
	assert.NoError(t, err)

	formatted, err := c.FormatStream(stream)
	assert.NoError(t, err)
	assert.Equal(t, ".dcl 0x123456, 0x1+0x2\n.dcd 0x12345678\n.dsl 0x2, 0xABCDEF\n.dsd 0x2, 0x12345678", formatted)

	roundTripped, err := c.ParseStream(t.Context(), "formatted.asm", strings.NewReader(formatted))
	assert.NoError(t, err)
	originalAssembly, err := c.AssembleStream(t.Context(), stream)
	assert.NoError(t, err)
	roundTripAssembly, err := c.AssembleStream(t.Context(), roundTripped)
	assert.NoError(t, err)
	assert.Equal(t, []byte{
		0x56, 0x34, 0x12, 0x03, 0x00, 0x00,
		0x78, 0x56, 0x34, 0x12,
		0xef, 0xcd, 0xab, 0xef, 0xcd, 0xab,
		0x78, 0x56, 0x34, 0x12, 0x78, 0x56, 0x34, 0x12,
	}, originalAssembly.Binary)
	assert.Equal(t, originalAssembly.Binary, roundTripAssembly.Binary)
}

//nolint:funlen // Addressing-family coverage is intentionally one auditable table.
func TestCPU65816Codec_AddressingFamiliesRoundTrip(t *testing.T) {
	t.Parallel()

	c := newCPU65816Codec(t)
	tests := []struct {
		name     string
		mnemonic string
		operands cpu65816parser.Operands
		wantText string
		wantCode []byte
	}{
		{name: "implied", mnemonic: cpu65816.NopName, wantText: "nop", wantCode: []byte{0xea}},
		{
			name: "accumulator", mnemonic: cpu65816.AslName,
			operands: cpu65816parser.Operands{cpu65816parser.AccumulatorOperand()},
			wantText: "asl a", wantCode: []byte{0x0a},
		},
		{
			name: "direct page", mnemonic: cpu65816.LdaName,
			operands: cpu65816parser.Operands{cpu65816parser.MemoryOperand(
				cpu65816parser.OperandAddress, cpu65816parser.AddressDirectPage, ast.NewNumber(0x10),
			)},
			wantText: "lda z:$10", wantCode: []byte{0xa5, 0x10},
		},
		{
			name: "absolute", mnemonic: cpu65816.LdaName,
			operands: cpu65816parser.Operands{cpu65816parser.MemoryOperand(
				cpu65816parser.OperandAddress, cpu65816parser.AddressAbsolute, ast.NewNumber(0x1234),
			)},
			wantText: "lda a:$1234", wantCode: []byte{0xad, 0x34, 0x12},
		},
		{
			name: "absolute long", mnemonic: cpu65816.LdaName,
			operands: cpu65816parser.Operands{cpu65816parser.MemoryOperand(
				cpu65816parser.OperandAddress, cpu65816parser.AddressLong, ast.NewNumber(0x123456),
			)},
			wantText: "lda f:$123456", wantCode: []byte{0xaf, 0x56, 0x34, 0x12},
		},
		{
			name: "absolute long indexed X", mnemonic: cpu65816.LdaName,
			operands: cpu65816parser.Operands{cpu65816parser.MemoryOperand(
				cpu65816parser.OperandIndexedX, cpu65816parser.AddressLong, ast.NewNumber(0x123456),
			)},
			wantText: "lda f:$123456,x", wantCode: []byte{0xbf, 0x56, 0x34, 0x12},
		},
		{
			name: "direct page indexed X", mnemonic: cpu65816.LdaName,
			operands: cpu65816parser.Operands{cpu65816parser.MemoryOperand(
				cpu65816parser.OperandIndexedX, cpu65816parser.AddressDirectPage, ast.NewNumber(0x10),
			)},
			wantText: "lda z:$10,x", wantCode: []byte{0xb5, 0x10},
		},
		{
			name: "absolute indexed Y", mnemonic: cpu65816.LdaName,
			operands: cpu65816parser.Operands{cpu65816parser.MemoryOperand(
				cpu65816parser.OperandIndexedY, cpu65816parser.AddressAbsolute, ast.NewNumber(0x1234),
			)},
			wantText: "lda a:$1234,y", wantCode: []byte{0xb9, 0x34, 0x12},
		},
		{
			name: "direct page indirect", mnemonic: cpu65816.LdaName,
			operands: cpu65816parser.Operands{cpu65816parser.MemoryOperand(
				cpu65816parser.OperandIndirect, cpu65816parser.AddressDirectPage, ast.NewNumber(0x10),
			)},
			wantText: "lda z:($10)", wantCode: []byte{0xb2, 0x10},
		},
		{
			name: "direct page indexed X indirect", mnemonic: cpu65816.LdaName,
			operands: cpu65816parser.Operands{cpu65816parser.MemoryOperand(
				cpu65816parser.OperandIndexedXIndirect, cpu65816parser.AddressDirectPage, ast.NewNumber(0x10),
			)},
			wantText: "lda z:($10,x)", wantCode: []byte{0xa1, 0x10},
		},
		{
			name: "direct page indirect indexed Y", mnemonic: cpu65816.LdaName,
			operands: cpu65816parser.Operands{cpu65816parser.MemoryOperand(
				cpu65816parser.OperandIndirectIndexedY, cpu65816parser.AddressDirectPage, ast.NewNumber(0x10),
			)},
			wantText: "lda z:($10),y", wantCode: []byte{0xb1, 0x10},
		},
		{
			name: "direct page indirect long", mnemonic: cpu65816.LdaName,
			operands: cpu65816parser.Operands{cpu65816parser.MemoryOperand(
				cpu65816parser.OperandIndirectLong, cpu65816parser.AddressDirectPage, ast.NewNumber(0x10),
			)},
			wantText: "lda z:[$10]", wantCode: []byte{0xa7, 0x10},
		},
		{
			name: "direct page indirect long indexed Y", mnemonic: cpu65816.LdaName,
			operands: cpu65816parser.Operands{cpu65816parser.MemoryOperand(
				cpu65816parser.OperandIndirectLongIndexedY,
				cpu65816parser.AddressDirectPage,
				ast.NewNumber(0x10),
			)},
			wantText: "lda z:[$10],y", wantCode: []byte{0xb7, 0x10},
		},
		{
			name: "stack relative", mnemonic: cpu65816.LdaName,
			operands: cpu65816parser.Operands{cpu65816parser.MemoryOperand(
				cpu65816parser.OperandStackRelative,
				cpu65816parser.AddressDirectPage,
				ast.NewNumber(0x10),
			)},
			wantText: "lda $10,s", wantCode: []byte{0xa3, 0x10},
		},
		{
			name: "stack relative indirect indexed Y", mnemonic: cpu65816.LdaName,
			operands: cpu65816parser.Operands{cpu65816parser.MemoryOperand(
				cpu65816parser.OperandStackRelativeIndirectIndexedY,
				cpu65816parser.AddressDirectPage,
				ast.NewNumber(0x10),
			)},
			wantText: "lda ($10,s),y", wantCode: []byte{0xb3, 0x10},
		},
		{
			name: "absolute indirect jump", mnemonic: cpu65816.JmpName,
			operands: cpu65816parser.Operands{cpu65816parser.MemoryOperand(
				cpu65816parser.OperandIndirect, cpu65816parser.AddressAbsolute, ast.NewNumber(0x1234),
			)},
			wantText: "jmp a:($1234)", wantCode: []byte{0x6c, 0x34, 0x12},
		},
		{
			name: "absolute indexed indirect jump", mnemonic: cpu65816.JmpName,
			operands: cpu65816parser.Operands{cpu65816parser.MemoryOperand(
				cpu65816parser.OperandIndexedXIndirect,
				cpu65816parser.AddressAbsolute,
				ast.NewNumber(0x1234),
			)},
			wantText: "jmp a:($1234,x)", wantCode: []byte{0x7c, 0x34, 0x12},
		},
		{
			name: "absolute indirect long jump", mnemonic: cpu65816.JmlName,
			operands: cpu65816parser.Operands{cpu65816parser.MemoryOperand(
				cpu65816parser.OperandIndirectLong,
				cpu65816parser.AddressAbsolute,
				ast.NewNumber(0x1234),
			)},
			wantText: "jml a:[$1234]", wantCode: []byte{0xdc, 0x34, 0x12},
		},
		{
			name: "block move", mnemonic: cpu65816.MvnName,
			operands: cpu65816parser.BlockMoveOperands(ast.NewNumber(1), ast.NewNumber(2)),
			wantText: "mvn $01,$02", wantCode: []byte{0x54, 0x02, 0x01},
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

func TestCPU65816Codec_StatefulImmediateWidths(t *testing.T) {
	t.Parallel()

	c := newCPU65816Codec(t)
	source := strings.NewReader(strings.Join([]string{
		"rep #$30",
		"lda #$1234",
		"ldx #$5678",
		"sep #$20",
		"lda #$12",
		"ldx #$9ABC",
		"sep #$10",
	}, "\n"))

	stream, err := codec.ParseStreamWithState(
		t.Context(), c, "stateful.asm", source, cpu65816parser.DefaultState(),
	)
	assert.NoError(t, err)
	initialState, finalState, ok := ast.StateSnapshots[cpu65816parser.State](stream)
	assert.True(t, ok)
	assert.Equal(t, cpu65816parser.DefaultState(), initialState)
	assert.Equal(t, cpu65816parser.WidthByte, finalState.AccumulatorWidth)
	assert.Equal(t, cpu65816parser.WidthByte, finalState.IndexWidth)
	for _, node := range stream.Nodes() {
		instruction, ok := ast.InstructionFromNode(node)
		assert.True(t, ok)
		assert.NoError(t, c.ValidateInstruction(instruction))
	}

	assembly, err := c.AssembleStream(t.Context(), stream)
	assert.NoError(t, err)
	assert.Equal(t, []byte{
		0xc2, 0x30,
		0xa9, 0x34, 0x12,
		0xa2, 0x78, 0x56,
		0xe2, 0x20,
		0xa9, 0x12,
		0xa2, 0xbc, 0x9a,
		0xe2, 0x10,
	}, assembly.Binary)
}

func TestCPU65816Codec_RecordsInstructionRelocations(t *testing.T) {
	t.Parallel()

	c := newCPU65816Codec(t)
	stream, err := c.ParseStream(t.Context(), "input.asm", strings.NewReader(strings.Join([]string{
		"target:",
		"lda #target",
		"lda z:target",
		"lda a:target + 1",
		"jml f:target + 2",
		"bra target",
		"brl target",
	}, "\n")))
	assert.NoError(t, err)
	assert.Empty(t, stream.Relocations())

	assembly, err := c.AssembleStream(t.Context(), stream)
	assert.NoError(t, err)
	assert.Equal(t, []byte{
		0xa9, 0x00,
		0xa5, 0x00,
		0xad, 0x01, 0x00,
		0x5c, 0x02, 0x00, 0x00,
		0x80, 0xf3,
		0x82, 0xf0, 0xff,
	}, assembly.Binary)
	assert.Equal(t, []ast.Relocation{
		{EntryIndex: 1, ByteOffset: 1, Kind: ast.AbsoluteRelocation, Expression: ast.NewSymbolExpression("target", 0, ast.FullAddress), Width: ast.WidthByte, ByteOrder: ast.ByteOrderLittle},
		{EntryIndex: 2, ByteOffset: 1, Kind: ast.AbsoluteRelocation, Expression: ast.NewSymbolExpression("target", 0, ast.FullAddress), Width: ast.WidthByte, ByteOrder: ast.ByteOrderLittle},
		{EntryIndex: 3, ByteOffset: 1, Kind: ast.AbsoluteRelocation, Expression: ast.NewSymbolExpression("target", 1, ast.FullAddress), Width: ast.WidthWord, ByteOrder: ast.ByteOrderLittle},
		{EntryIndex: 4, ByteOffset: 1, Kind: ast.AbsoluteRelocation, Expression: ast.NewSymbolExpression("target", 2, ast.FullAddress), Width: ast.WidthLong, ByteOrder: ast.ByteOrderLittle},
		{EntryIndex: 5, ByteOffset: 1, Kind: ast.RelativeRelocation, Expression: ast.NewSymbolExpression("target", 0, ast.FullAddress), Width: ast.WidthByte, ByteOrder: ast.ByteOrderLittle},
		{EntryIndex: 6, ByteOffset: 1, Kind: ast.RelativeRelocation, Expression: ast.NewSymbolExpression("target", 0, ast.FullAddress), Width: ast.WidthWord, ByteOrder: ast.ByteOrderLittle},
	}, assembly.Stream.Relocations())
	assert.NoError(t, assembly.Stream.Validate())

	reassembled, err := c.AssembleStream(t.Context(), assembly.Stream)
	assert.NoError(t, err)
	assert.Equal(t, assembly.Binary, reassembled.Binary)
	assert.Equal(t, assembly.Stream.Relocations(), reassembled.Stream.Relocations())
}

func TestCPU65816Codec_RecordsStateSelectedImmediateRelocations(t *testing.T) {
	t.Parallel()

	c := newCPU65816Codec(t)
	stream, err := c.ParseStream(t.Context(), "stateful.asm", strings.NewReader(strings.Join([]string{
		"rep #$30",
		"target:",
		"lda #target",
		"ldx #target",
	}, "\n")))
	assert.NoError(t, err)

	assembly, err := c.AssembleStream(t.Context(), stream)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xc2, 0x30, 0xa9, 0x02, 0x00, 0xa2, 0x02, 0x00}, assembly.Binary)
	assert.Equal(t, []ast.Relocation{
		{EntryIndex: 2, ByteOffset: 1, Kind: ast.AbsoluteRelocation, Expression: ast.NewSymbolExpression("target", 0, ast.FullAddress), Width: ast.WidthWord, ByteOrder: ast.ByteOrderLittle},
		{EntryIndex: 3, ByteOffset: 1, Kind: ast.AbsoluteRelocation, Expression: ast.NewSymbolExpression("target", 0, ast.FullAddress), Width: ast.WidthWord, ByteOrder: ast.ByteOrderLittle},
	}, assembly.Stream.Relocations())
	assert.NoError(t, assembly.Stream.Validate())
}

func TestCPU65816Codec_RejectsStaleStateSelectedWidthRelocation(t *testing.T) {
	t.Parallel()

	c := newCPU65816Codec(t)
	byteState := cpu65816parser.DefaultState()
	byteLoad, _, err := codec.BuildInstructionWithState(c, cpu65816.LdaName, cpu65816parser.Operands{
		cpu65816parser.ImmediateOperand(ast.NewLabel("target")),
	}, byteState)
	assert.NoError(t, err)
	wordState := byteState
	wordState.AccumulatorWidth = cpu65816parser.WidthWord
	wordLoad, _, err := codec.BuildInstructionWithState(c, cpu65816.LdaName, cpu65816parser.Operands{
		cpu65816parser.ImmediateOperand(ast.NewLabel("target")),
	}, wordState)
	assert.NoError(t, err)

	assembly, err := c.Assemble(t.Context(), []ast.Node{byteLoad, ast.NewLabel("target")})
	assert.NoError(t, err)
	assert.Equal(t, ast.WidthByte, assembly.Stream.Relocations()[0].Width)
	err = assembly.Stream.Replace(0, 1, []ast.Entry{ast.NewEntry(wordLoad, ast.SourcePosition{})})
	assert.NoError(t, err)

	_, err = c.AssembleStream(t.Context(), assembly.Stream)
	assert.ErrorIs(t, err, codec.ErrInstructionRelocationMismatch)
}

func TestCPU65816Codec_ParallelStreamsKeepIndependentState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		want   cpu65816parser.State
	}{
		{
			name:   "wide accumulator",
			source: "rep #$20",
			want: cpu65816parser.State{
				AccumulatorWidth: cpu65816parser.WidthWord,
				IndexWidth:       cpu65816parser.WidthByte,
				Emulation:        cpu65816parser.StatusClear,
			},
		},
		{
			name:   "enter emulation",
			source: "sec\nxce",
			want: cpu65816parser.State{
				AccumulatorWidth: cpu65816parser.WidthByte,
				IndexWidth:       cpu65816parser.WidthByte,
				Carry:            cpu65816parser.StatusClear,
				Emulation:        cpu65816parser.StatusSet,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			c := newCPU65816Codec(t)
			stream, err := codec.ParseStreamWithState(
				t.Context(), c, test.name+".asm", strings.NewReader(test.source), cpu65816parser.DefaultState(),
			)
			assert.NoError(t, err)
			initial, final, ok := ast.StateSnapshots[cpu65816parser.State](stream)
			assert.True(t, ok)
			assert.Equal(t, cpu65816parser.DefaultState(), initial)
			assert.Equal(t, test.want, final)
		})
	}
}

func TestCPU65816Codec_StatefulBuilderTransitions(t *testing.T) {
	t.Parallel()

	c := newCPU65816Codec(t)
	state := cpu65816parser.DefaultState()
	rep, state, err := codec.BuildInstructionWithState(
		c,
		cpu65816.RepName,
		cpu65816parser.Operands{cpu65816parser.ImmediateOperand(ast.NewNumber(0x20))},
		state,
	)
	assert.NoError(t, err)
	assert.Equal(t, cpu65816parser.WidthWord, state.AccumulatorWidth)
	assert.Equal(t, cpu65816parser.WidthByte, state.IndexWidth)

	wideLoad, state, err := codec.BuildInstructionWithState(
		c,
		cpu65816.LdaName,
		cpu65816parser.Operands{cpu65816parser.ImmediateOperand(ast.NewNumber(0x1234))},
		state,
	)
	assert.NoError(t, err)
	assert.Equal(t, cpu65816parser.WidthWord, state.AccumulatorWidth)

	formatted, err := c.FormatInstruction(wideLoad)
	assert.NoError(t, err)
	assert.Equal(t, "lda #$1234", formatted)
	assembly, err := c.Assemble(t.Context(), []ast.Node{rep, wideLoad})
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xc2, 0x20, 0xa9, 0x34, 0x12}, assembly.Binary)
}

func TestCPU65816Codec_TracksNativeAndEmulationTransitions(t *testing.T) {
	t.Parallel()

	c := newCPU65816Codec(t)
	source := strings.NewReader(strings.Join([]string{
		"clc",
		"xce",
		"rep #$30",
		"lda #$1234",
		"ldx #$5678",
		"sec",
		"xce",
		"rep #$30",
		"lda #$12",
		"ldx #$34",
		"clc",
		"xce",
	}, "\n"))

	nodes, state, err := codec.ParseWithState(t.Context(), c, source, cpu65816parser.DefaultState())
	assert.NoError(t, err)
	assert.Equal(t, cpu65816parser.WidthByte, state.AccumulatorWidth)
	assert.Equal(t, cpu65816parser.WidthByte, state.IndexWidth)
	assert.Equal(t, cpu65816parser.StatusSet, state.Carry)
	assert.Equal(t, cpu65816parser.StatusClear, state.Emulation)

	assembly, err := c.Assemble(t.Context(), nodes)
	assert.NoError(t, err)
	assert.Equal(t, []byte{
		0x18, 0xfb, 0xc2, 0x30, 0xa9, 0x34, 0x12, 0xa2, 0x78, 0x56,
		0x38, 0xfb, 0xc2, 0x30, 0xa9, 0x12, 0xa2, 0x34, 0x18, 0xfb,
	}, assembly.Binary)
}

func TestCPU65816Codec_ParsesCompilerEntryPrologue(t *testing.T) {
	t.Parallel()

	c := newCPU65816Codec(t)
	source := strings.NewReader(strings.Join([]string{
		"sei",
		"clc",
		"xce",
		"lda #$1f",
		"xba",
		"lda #$ff",
		"tcs",
		"sep #$30",
		"rts",
	}, "\n"))

	nodes, state, err := codec.ParseWithState(t.Context(), c, source, cpu65816parser.DefaultState())
	assert.NoError(t, err)
	assert.Equal(t, cpu65816parser.WidthByte, state.AccumulatorWidth)
	assert.Equal(t, cpu65816parser.WidthByte, state.IndexWidth)
	assert.Equal(t, cpu65816parser.StatusClear, state.Emulation)

	assembly, err := c.Assemble(t.Context(), nodes)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x78, 0x18, 0xfb, 0xa9, 0x1f, 0xeb, 0xa9, 0xff, 0x1b, 0xe2, 0x30, 0x60}, assembly.Binary)
}

func TestCPU65816Codec_RejectsInvalidStateAndOperands(t *testing.T) {
	t.Parallel()

	c := newCPU65816Codec(t)
	_, _, err := codec.BuildInstructionWithState(
		c,
		cpu65816.LdaName,
		cpu65816parser.Operands{cpu65816parser.ImmediateOperand(ast.NewNumber(0x12))},
		cpu65816parser.State{},
	)
	assert.Error(t, err)

	_, err = codec.BuildInstruction(
		c,
		cpu65816.LdaName,
		cpu65816parser.Operands{cpu65816parser.ImmediateOperand(ast.NewNumber(0x100))},
	)
	assert.Error(t, err)

	_, err = codec.BuildInstruction(
		c,
		cpu65816.MvnName,
		cpu65816parser.BlockMoveOperands(ast.NewNumber(0x100), ast.NewNumber(2)),
	)
	assert.Error(t, err)

	_, _, err = codec.ParseWithState(
		t.Context(),
		c,
		strings.NewReader("lda #$12"),
		"invalid state",
	)
	assert.Error(t, err)

	_, _, err = codec.ParseWithState(
		t.Context(),
		c,
		strings.NewReader("plp\nlda #$12"),
		cpu65816parser.DefaultState(),
	)
	assert.Error(t, err)
}

func TestCPU65816TypedInstructionFormattingOptions(t *testing.T) {
	t.Parallel()

	c := newCPU65816Codec(t)
	instruction, err := codec.BuildInstruction(
		c,
		cpu65816.LdaName,
		cpu65816parser.Operands{cpu65816parser.MemoryOperand(
			cpu65816parser.OperandIndexedX,
			cpu65816parser.AddressAbsolute,
			ast.NewNumber(0x42),
		)},
	)
	assert.NoError(t, err)

	formatted, err := cpu65816parser.FormatInstructionWithOptions(
		instruction,
		cpu65816parser.FormatOptions{Indent: "  ", Uppercase: true, WordHexDigits: 6},
	)

	assert.NoError(t, err)
	assert.Equal(t, "  LDA A:$000042,X", formatted)
}

func newCPU65816Codec(t *testing.T) *codec.Codec[*cpu65816.Instruction] {
	t.Helper()
	return newCPU65816CodecWithConfig(t, asmcpu65816.New())
}

func newCPU65816CodecWithConfig(
	t *testing.T,
	configuration *config.Config[*cpu65816.Instruction],
) *codec.Codec[*cpu65816.Instruction] {

	t.Helper()
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
