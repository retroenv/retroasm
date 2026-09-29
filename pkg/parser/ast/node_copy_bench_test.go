package ast

import "testing"

var copiedInstruction Node

func BenchmarkInstructionCopy(b *testing.B) {
	instruction := NewInstruction("lda", 1, NewNumber(42), nil)
	instruction.SetComment("load value")
	instruction.setEntryHandle(&entryHandle{})
	b.ReportAllocs()

	for b.Loop() {
		copiedInstruction = instruction.Copy()
	}
}
