package ast

import (
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

func TestStreamRewriteRetainedNodesKeepOldView(t *testing.T) {
	stream := NewStreamFromNodes(NewLabel("entry"), NewInstruction("lda", 0, NewNumber(1), nil))
	view, err := stream.EditNodes()
	assert.NoError(t, err)
	before := view.Nodes()
	assert.NoError(t, stream.Rewrite([]EntryEdit{{SourceIndex: 1}, {SourceIndex: 0}}))
	assert.Equal(t, before, view.Nodes())
	assert.NoError(t, stream.RenameSymbols(map[string]string{"entry": "renamed"}))
	assert.Equal(t, before, view.Nodes())
	assert.Equal(t, "renamed", stream.At(1).Node.(Label).Name)
	assert.ErrorContains(t, view.Commit(before), "stale")
}

func TestStreamRewriteRetainedDuplicatesOwnMetadata(t *testing.T) {
	stream := NewStreamFromNodes(NewInstruction("lda", 0, NewNumber(1), nil))
	assert.NoError(t, stream.Rewrite([]EntryEdit{{SourceIndex: 0}, {SourceIndex: 0}}))
	stream.entries[0].Node.SetComment("first")
	stream.entries[0].Node.(Instruction).Argument.SetComment("first operand")
	assert.Equal(t, "", InlineComment(stream.entries[1].Node))
	assert.Equal(t, "", InlineComment(stream.entries[1].Node.(Instruction).Argument))
}

func TestStreamRewriteRetainedSourceClearsHandle(t *testing.T) {
	node := NewInstruction("lda", 0, NewNumber(1), nil)
	node.setEntryHandle(&entryHandle{})
	stream := NewStreamFromNodes(node)
	assert.NoError(t, stream.Rewrite([]EntryEdit{{SourceIndex: 0}}))
	assert.Nil(t, stream.At(0).Node.(Instruction).entryHandle())
}

func TestStreamRewriteManyCopiesKeepRelocations(t *testing.T) {
	stream := streamJoinFixture("entry", "input.asm")
	before := stream.Copy()
	edits := make([]EntryEdit, 0, 27)
	edits = append(edits, EntryEdit{SourceIndex: 0}, EntryEdit{SourceIndex: 1})
	for range 25 {
		edits = append(edits, EntryEdit{SourceIndex: 2})
	}
	assert.NoError(t, stream.Rewrite(edits))
	assert.Len(t, stream.Relocations(), 25)
	for index, relocation := range stream.Relocations() {
		assert.Equal(t, index+2, relocation.EntryIndex)
		assert.Equal(t, before.Relocations()[0].Expression, relocation.Expression)
		assert.Equal(t, before.At(2), stream.At(index+2))
	}
	assert.Equal(t, before.SegmentChanges(), stream.SegmentChanges())
}

func TestNodeEditCommitUnchangedNodesKeepIndependentReads(t *testing.T) {
	instruction := NewInstruction("lda", 0, NewNumber(1), nil)
	instruction.SetComment("source")
	instruction.Argument.SetComment("operand")
	stream := NewStreamFromNodes(instruction, NewInstruction("nop", 0, nil, nil))
	view, err := stream.EditNodes()
	assert.NoError(t, err)
	nodes := view.Nodes()
	assert.NoError(t, view.Commit([]Node{nodes[0], nodes[0], nodes[1]}))
	before := stream.Copy()
	nodes[0].SetComment("caller")
	nodes[0].(Instruction).Argument.SetComment("caller operand")
	assert.Equal(t, before, stream)
	stream.entries[0].Node.SetComment("first")
	stream.entries[0].Node.(Instruction).Argument.SetComment("first operand")
	assert.Equal(t, "source", InlineComment(stream.entries[1].Node))
	assert.Equal(t, "operand", InlineComment(stream.entries[1].Node.(Instruction).Argument))
}

func BenchmarkNodeEditSingleMutation(b *testing.B) {
	for _, commit := range []bool{false, true} {
		name := "range"
		if commit {
			name = "commit"
		}
		b.Run(name, func(b *testing.B) {
			nodes := make([]Node, 512)
			for index := range nodes {
				nodes[index] = NewInstruction("lda", 0, NewNumber(uint64(index)), nil)
			}
			stream := NewStreamFromNodes(nodes...)
			var value uint64
			b.ReportAllocs()
			for b.Loop() {
				view, err := stream.EditNodes()
				assert.NoError(b, err)
				var candidates []Node
				if commit {
					candidates = view.Nodes()
				}
				native := view.At(256).(Instruction)
				native.Argument = NewNumber(value)
				value++
				if commit {
					candidates[256] = native
					assert.NoError(b, view.Commit(candidates))
				} else {
					assert.NoError(b, view.Replace(256, 257, []Node{native}))
				}
			}
		})
	}
}
