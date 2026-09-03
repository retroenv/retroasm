package codec_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	asmchip8 "github.com/retroenv/retroasm/pkg/arch/chip8"
	asmcpu6502 "github.com/retroenv/retroasm/pkg/arch/cpu6502"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/arch/cpu/cpu6502"
	"github.com/retroenv/retrogolib/assert"
)

func TestNew_RejectsIncompleteConfiguration(t *testing.T) {
	t.Parallel()

	_, err := codec.New[*cpu6502.Instruction](nil)
	assert.ErrorIs(t, err, codec.ErrNilConfiguration)

	_, err = codec.New(&config.Config[*cpu6502.Instruction]{})
	assert.ErrorIs(t, err, codec.ErrNilArchitecture)
}

func TestCodec_ParseTypedStream(t *testing.T) {
	t.Parallel()

	c := newCPU6502Codec(t)
	stream, err := c.ParseStream(
		t.Context(),
		"input.asm",
		strings.NewReader("start:\n; before load\nlda #1 ; inline"),
	)
	assert.NoError(t, err)
	assert.Equal(t, 3, stream.Len())
	assert.Equal(t, ast.SourcePosition{Source: "input.asm", Line: 1, Column: 1}, stream.At(0).Position)
	assert.Equal(t, ast.SourcePosition{Source: "input.asm", Line: 2, Column: 1}, stream.At(1).Position)
	assert.Equal(t, ast.SourcePosition{Source: "input.asm", Line: 3, Column: 1}, stream.At(2).Position)
	nodes := stream.Nodes()
	assert.Len(t, nodes, 3)

	label, ok := nodes[0].(ast.Label)
	assert.True(t, ok)
	assert.Equal(t, "start", label.Name)
	_, ok = nodes[1].(*ast.Comment)
	assert.True(t, ok)
	instruction, ok := ast.InstructionFromNode(nodes[2])
	assert.True(t, ok)
	assert.True(t, instruction.OpcodeID.ValidFor(arch.CPU6502))
	assert.Equal(t, uint16(cpu6502.Lda), instruction.OpcodeID.Value)
}

func TestCodec_ParseInstruction(t *testing.T) {
	t.Parallel()

	c := newCPU6502Codec(t)
	instruction, err := c.ParseInstruction(t.Context(), strings.NewReader("lda #1"))
	assert.NoError(t, err)
	assert.Equal(t, cpu6502.LdaName, instruction.Name)
	assert.True(t, instruction.OpcodeID.ValidFor(arch.CPU6502))

	_, err = c.ParseInstruction(t.Context(), strings.NewReader("label:"))
	assert.ErrorIs(t, err, codec.ErrExpectedInstruction)
	_, err = c.ParseInstruction(t.Context(), strings.NewReader("lda #1\nsta $20"))
	assert.ErrorIs(t, err, codec.ErrExpectedInstruction)
	_, err = c.ParseInstruction(t.Context(), nil)
	assert.ErrorIs(t, err, codec.ErrNilSource)
}

func TestCodec_OpcodeID(t *testing.T) {
	t.Parallel()

	c := newCPU6502Codec(t)
	identity, err := c.OpcodeID(" LDA ")
	assert.NoError(t, err)
	assert.True(t, identity.ValidFor(arch.CPU6502))
	assert.Equal(t, uint16(cpu6502.Lda), identity.Value)

	_, err = c.OpcodeID("missing")
	assert.ErrorIs(t, err, codec.ErrUnknownInstruction)
}

func TestCodec_AssembleTypedStreamWithoutReparsing(t *testing.T) {
	t.Parallel()

	c, err := codec.New(asmchip8.New())
	assert.NoError(t, err)
	stream, err := c.ParseStream(t.Context(), "input.asm", strings.NewReader("entry:\ncls"))
	assert.NoError(t, err)

	result, err := c.AssembleStream(t.Context(), stream)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x00, 0xe0}, result.Binary)
	assert.Equal(t, uint64(0x200), result.Symbols["entry"])

	_, err = c.AssembleStream(t.Context(), nil)
	assert.ErrorIs(t, err, codec.ErrNilStream)
}

func TestCodec_ValidateStreamReportsInstructionPosition(t *testing.T) {
	t.Parallel()

	c := newCPU6502Codec(t)
	stream, err := c.ParseStream(t.Context(), "input.asm", strings.NewReader("entry:\nlda #1"))
	assert.NoError(t, err)
	assert.NoError(t, c.ValidateStream(stream))

	stream.Replace(1, 2, []ast.Entry{ast.NewEntry(
		ast.NewInstruction("missing", 0, nil, nil),
		ast.SourcePosition{Source: "input.asm", Line: 2, Column: 1},
	)})
	err = c.ValidateStream(stream)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "input.asm:2:1")
	assert.ErrorContains(t, err, "unknown CPU6502 instruction")

	err = c.ValidateStream(nil)
	assert.ErrorIs(t, err, codec.ErrNilStream)
}

func TestCodec_FormatStreamRoundTripsLabelsCommentsAndInstructions(t *testing.T) {
	t.Parallel()

	c := newCPU6502AssemblyCodec(t)
	stream, err := c.ParseStream(
		t.Context(),
		"input.asm",
		strings.NewReader("; prologue\nentry: ; target\nlda #1 ; load"),
	)
	assert.NoError(t, err)

	formatted, err := c.FormatStream(stream)
	assert.NoError(t, err)
	assert.Equal(t, "; prologue\nentry: ; target\nlda #$01 ; load", formatted)

	roundTripped, err := c.ParseStream(t.Context(), "formatted.asm", strings.NewReader(formatted))
	assert.NoError(t, err)
	assert.NoError(t, c.ValidateStream(roundTripped))
	assert.Equal(t, stream.Len(), roundTripped.Len())
	assert.Equal(t, "prologue", roundTripped.At(0).Node.(*ast.Comment).Message)
	assert.Equal(t, "target", ast.InlineComment(roundTripped.At(1).Node))
	assert.Equal(t, "load", ast.InlineComment(roundTripped.At(2).Node))

	originalAssembly, err := c.AssembleStream(t.Context(), stream)
	assert.NoError(t, err)
	roundTripAssembly, err := c.AssembleStream(t.Context(), roundTripped)
	assert.NoError(t, err)
	assert.Equal(t, originalAssembly.Binary, roundTripAssembly.Binary)
	assert.Equal(t, originalAssembly.Symbols, roundTripAssembly.Symbols)
}

func TestCodec_FormatStreamRejectsUnsupportedNodes(t *testing.T) {
	t.Parallel()

	c := newCPU6502Codec(t)
	stream := ast.NewStream(ast.NewEntry(
		ast.NewData(ast.DataType, 1),
		ast.SourcePosition{Source: "input.asm", Line: 4, Column: 1},
	))

	_, err := c.FormatStream(stream)
	assert.ErrorIs(t, err, codec.ErrFormattingUnsupported)
	assert.ErrorContains(t, err, "input.asm:4:1")

	_, err = c.FormatStream(nil)
	assert.ErrorIs(t, err, codec.ErrNilStream)
}

func TestCodec_ParseHonorsCancellation(t *testing.T) {
	t.Parallel()

	c := newCPU6502Codec(t)
	ctx := t.Context()
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	_, err := c.Parse(cancelled, strings.NewReader("lda #1"))
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func newCPU6502Codec(t *testing.T) *codec.Codec[*cpu6502.Instruction] {
	t.Helper()
	c, err := codec.New(asmcpu6502.New())
	assert.NoError(t, err)
	return c
}
