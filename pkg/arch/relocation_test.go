package arch_test

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/assert"
)

func TestRecordInstructionRelocation(t *testing.T) {
	t.Parallel()

	assigner := &recordingAssigner{}
	encoding := arch.RelocationEncoding{
		ByteOffset:    1,
		Kind:          ast.RelativeRelocation,
		Width:         ast.WidthByte,
		ByteOrder:     ast.ByteOrderLittle,
		ReferenceType: ast.FullAddress,
	}
	arch.RecordInstructionRelocation(assigner, nil, "target", encoding)

	assert.Equal(t, "target", assigner.argument)
	assert.Equal(t, encoding, assigner.encoding)
}

type recordingAssigner struct {
	argument any
	encoding arch.RelocationEncoding
}

func (*recordingAssigner) ArgumentValue(any) (uint64, error) {
	return 0, nil
}

func (*recordingAssigner) ProgramCounter() uint64 {
	return 0
}

func (*recordingAssigner) RelativeOffset(uint64, uint64) (byte, error) {
	return 0, nil
}

func (ass *recordingAssigner) RecordInstructionRelocation(_ arch.Instruction, argument any, encoding arch.RelocationEncoding) {
	ass.argument = argument
	ass.encoding = encoding
}
