package ast

import (
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

func TestNodeEditReadsKeepSourceRevision(t *testing.T) {
	for _, mutation := range []string{"append", "replace", "rewrite", "rename", "symbols", "resolve", "symbol", "relocation", "segment", "state", "join"} {
		t.Run(mutation, func(t *testing.T) {
			stream := streamJoinFixture("entry", "input.asm")
			edit, err := stream.EditNodes()
			assert.NoError(t, err)
			before := edit.Nodes()
			mutateNodeEditStream(t, stream, mutation)
			assert.Equal(t, before, edit.Nodes())
			assert.ErrorContains(t, edit.Commit(edit.Nodes()), "stale")
		})
	}
}

func TestNodeEditReadsOwnOperandsAndHandles(t *testing.T) {
	instruction := NewInstruction("lda", 0, NewNumber(1), nil)
	instruction.SetComment("instruction")
	instruction.Argument.SetComment("operand")
	stream := NewStreamFromNodes(instruction)
	before := stream.Copy()
	edit, err := stream.EditNodes()
	assert.NoError(t, err)
	first := edit.At(0).(Instruction)
	second := edit.Nodes()[0].(Instruction)
	assert.True(t, first.entryHandle() == second.entryHandle())
	first.SetComment("changed instruction")
	first.Argument.SetComment("changed operand")
	first.setEntryHandle(nil)
	assert.Equal(t, "instruction", InlineComment(second))
	assert.Equal(t, "operand", InlineComment(second.Argument))
	assert.NotNil(t, second.entryHandle())
	assert.Equal(t, before, stream)
	assert.NoError(t, edit.Replace(0, 1, []Node{second}))
}

func BenchmarkNodeEditSnapshot(b *testing.B) {
	nodes := make([]Node, 512)
	for index := range nodes {
		nodes[index] = NewInstruction("lda", 0, NewNumber(uint64(index)), nil)
	}
	stream := NewStreamFromNodes(nodes...)
	for _, read := range []bool{false, true} {
		name := "create"
		if read {
			name = "create-and-read"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				edit, err := stream.EditNodes()
				assert.NoError(b, err)
				if read {
					for index := range edit.Len() {
						copiedInstruction = edit.At(index)
					}
				}
			}
		})
	}
}
