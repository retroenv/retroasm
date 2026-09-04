package codec_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/assert"
)

const (
	equivalenceInputSource     = "input.asm"
	equivalenceFormattedSource = "formatted.asm"
)

type streamCodec interface {
	AssembleStream(context.Context, *ast.Stream) (*codec.Assembly, error)
	FormatStream(*ast.Stream) (string, error)
	ParseStream(context.Context, string, io.Reader) (*ast.Stream, error)
	ValidateStream(*ast.Stream) error
}

type streamEquivalenceCase struct {
	name     string
	newCodec func(*testing.T) streamCodec
	source   string
}

type instructionForm struct {
	opcodeID   ast.OpcodeID
	addressing int
	argument   string
}

type streamVariant struct {
	name   string
	stream *ast.Stream
}

func TestCodec_StreamEquivalenceAcrossArchitectures(t *testing.T) {
	t.Parallel()

	for _, test := range streamEquivalenceCases() {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			verifyStreamEquivalence(t, test)
		})
	}
}

func streamEquivalenceCases() []streamEquivalenceCase {
	return []streamEquivalenceCase{
		{
			name:     "CPU6502",
			newCodec: newCPU6502StreamCodec,
			source:   "; header\nentry:\nlda target+1\ntarget:\nnop",
		},
		{
			name:     "CPU65816",
			newCodec: newCPU65816StreamCodec,
			source:   "; header\ntarget:\nnop\nentry:\nlda target+1",
		},
		{
			name:     "Z80",
			newCodec: newZ80StreamCodec,
			source:   "; header\nentry:\nld hl,target+1\ntarget:\nnop",
		},
		{
			name:     "SM83",
			newCodec: newSM83StreamCodec,
			source:   "; header\nentry:\nld bc,target+1\ntarget:\nnop",
		},
		{
			name:     "CPU68000",
			newCodec: newCPU68000StreamCodec,
			source:   "; header\nentry:\nmove.w #target,d0\ntarget:\nnop",
		},
		{
			name:     "CHIP8",
			newCodec: newCHIP8StreamCodec,
			source:   "; header\nentry:\ncall target\ntarget:\ncls",
		},
	}
}

func verifyStreamEquivalence(t *testing.T, test streamEquivalenceCase) {
	t.Helper()

	c := test.newCodec(t)
	stream, err := c.ParseStream(t.Context(), equivalenceInputSource, strings.NewReader(test.source))
	assert.NoError(t, err)
	assert.NoError(t, c.ValidateStream(stream))

	copied := stream.Copy()
	assertStreamMetadataEqual(t, stream, copied)

	replaced := stream.Copy()
	assert.NoError(t, replaceFirstInstructionWithCopy(replaced))
	assertStreamMetadataEqual(t, stream, replaced)

	formatted, err := c.FormatStream(stream)
	assert.NoError(t, err)
	roundTripped, err := c.ParseStream(t.Context(), equivalenceFormattedSource, strings.NewReader(formatted))
	assert.NoError(t, err)
	assert.Equal(t, instructionForms(stream), instructionForms(roundTripped))

	baseline, err := c.AssembleStream(t.Context(), stream)
	assert.NoError(t, err)
	assert.NotEmpty(t, baseline.Stream.Relocations())

	streams := []streamVariant{
		{name: "copy", stream: copied},
		{name: "replacement", stream: replaced},
		{name: "format-parse", stream: roundTripped},
	}
	for _, variant := range streams {
		t.Run(variant.name, func(t *testing.T) {
			assembly, err := c.AssembleStream(t.Context(), variant.stream)
			assert.NoError(t, err)
			assertAssemblyEqual(t, baseline, assembly)

			repeated, err := c.AssembleStream(t.Context(), assembly.Stream)
			assert.NoError(t, err)
			assertAssemblyEqual(t, baseline, repeated)
		})
	}

	assertDiagnosticCopyEquivalence(t, c, stream)
}

func assertStreamMetadataEqual(t *testing.T, expected, actual *ast.Stream) {
	t.Helper()

	assert.Equal(t, expected.Entries(), actual.Entries())
	assert.Equal(t, expected.Symbols(), actual.Symbols())
	assert.Equal(t, expected.Relocations(), actual.Relocations())
	assert.Equal(t, expected.SegmentChanges(), actual.SegmentChanges())
	assert.Equal(t, instructionForms(expected), instructionForms(actual))
}

func assertAssemblyEqual(t *testing.T, expected, actual *codec.Assembly) {
	t.Helper()

	assert.Equal(t, expected.Binary, actual.Binary)
	assert.Equal(t, expected.Symbols, actual.Symbols)
	assert.Equal(t, expected.Stream.Relocations(), actual.Stream.Relocations())
	assert.Equal(t, instructionForms(expected.Stream), instructionForms(actual.Stream))
}

func assertDiagnosticCopyEquivalence(t *testing.T, c streamCodec, stream *ast.Stream) {
	t.Helper()

	invalid := stream.Copy()
	index, entry, instruction := firstInstruction(t, invalid)
	instruction.Name = "missing"
	instruction.OpcodeID = ast.OpcodeID{}
	entry.Node = instruction
	assert.NoError(t, invalid.Replace(index, index+1, []ast.Entry{entry}))

	validationErr := c.ValidateStream(invalid)
	copyValidationErr := c.ValidateStream(invalid.Copy())
	assert.Error(t, validationErr)
	assert.Error(t, copyValidationErr)
	assert.Equal(t, validationErr.Error(), copyValidationErr.Error())

	_, formattingErr := c.FormatStream(invalid)
	_, copyFormattingErr := c.FormatStream(invalid.Copy())
	assert.Error(t, formattingErr)
	assert.Error(t, copyFormattingErr)
	assert.Equal(t, formattingErr.Error(), copyFormattingErr.Error())
}

func instructionForms(stream *ast.Stream) []instructionForm {
	forms := make([]instructionForm, 0)

	for _, entry := range stream.Entries() {
		instruction, ok := ast.InstructionFromNode(entry.Node)
		if !ok {
			continue
		}
		forms = append(forms, instructionForm{
			opcodeID:   instruction.OpcodeID,
			addressing: instruction.Addressing,
			argument:   ast.InstructionArgumentForm(instruction.Argument),
		})
	}
	return forms
}

func replaceFirstInstructionWithCopy(stream *ast.Stream) error {
	for index, entry := range stream.Entries() {
		if ast.IsInstruction(entry.Node) {
			if err := stream.Replace(index, index+1, []ast.Entry{entry}); err != nil {
				return fmt.Errorf("replacing instruction copy: %w", err)
			}
			return nil
		}
	}
	return errors.New("stream has no instruction")
}

func firstInstruction(t *testing.T, stream *ast.Stream) (int, ast.Entry, ast.Instruction) {
	t.Helper()

	for index, entry := range stream.Entries() {
		instruction, ok := ast.InstructionFromNode(entry.Node)
		if ok {
			return index, entry, instruction
		}
	}
	t.Fatal("stream has no instruction")
	return 0, ast.Entry{}, ast.Instruction{}
}

func newCPU6502StreamCodec(t *testing.T) streamCodec {
	t.Helper()
	return newCPU6502AssemblyCodec(t)
}

func newCPU65816StreamCodec(t *testing.T) streamCodec {
	t.Helper()
	return newCPU65816Codec(t)
}

func newZ80StreamCodec(t *testing.T) streamCodec {
	t.Helper()
	return newZ80Codec(t)
}

func newSM83StreamCodec(t *testing.T) streamCodec {
	t.Helper()
	return newSM83Codec(t)
}

func newCPU68000StreamCodec(t *testing.T) streamCodec {
	t.Helper()
	return newCPU68000Codec(t)
}

func newCHIP8StreamCodec(t *testing.T) streamCodec {
	t.Helper()
	return newCHIP8Codec(t)
}
