package ast

import (
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

func TestStreamRewriteMovesCopiesAndInsertsEntries(t *testing.T) {
	stream := streamJoinFixture("entry", "input.asm")
	stream.RecordState(streamJoinState{values: []int{1}}, streamJoinState{values: []int{2}})
	before := stream.Copy()
	assert.NoError(t, stream.Rewrite([]EntryEdit{
		{SourceIndex: 0}, {SourceIndex: 2}, {SourceIndex: 1}, {SourceIndex: 2},
		{SourceIndex: NoSourceEntry, Node: NewLabel("inserted")},
	}))
	assert.Equal(t, before.At(0), stream.At(0))
	assert.Equal(t, before.At(2), stream.At(1))
	assert.Equal(t, before.At(1), stream.At(2))
	assert.Equal(t, before.At(2), stream.At(3))
	assert.Equal(t, SourcePosition{}, stream.At(4).Position)
	assert.Equal(t, 2, stream.Symbols()[0].EntryIndex)
	assert.Equal(t, before.Symbols()[0].Position, stream.Symbols()[0].Position)
	assert.Equal(t, 4, stream.Symbols()[1].EntryIndex)
	assert.Equal(t, 1, stream.Relocations()[0].EntryIndex)
	assert.Equal(t, 3, stream.Relocations()[1].EntryIndex)
	assert.Equal(t, before.SegmentChanges(), stream.SegmentChanges())
	initial, final, ok := StateSnapshots[streamJoinState](stream)
	assert.True(t, ok)
	assert.Equal(t, []int{1}, initial.values)
	assert.Equal(t, []int{2}, final.values)
	assert.Empty(t, stream.RemovedEntries())
}

func TestStreamRewriteRetainsRemovedEntries(t *testing.T) {
	stream := streamJoinFixture("entry", "input.asm")
	before := stream.Copy()
	assert.NoError(t, stream.Rewrite([]EntryEdit{{SourceIndex: 0}, {SourceIndex: 2}}))
	assert.Equal(t, []Entry{before.At(1)}, stream.RemovedEntries())
	assert.Empty(t, stream.Symbols())
	assert.Equal(t, 1, stream.Relocations()[0].EntryIndex)
	assert.NoError(t, stream.Rewrite(nil))
	assert.Equal(t, []Entry{before.At(1), before.At(0), before.At(2)}, stream.RemovedEntries())
	assert.Equal(t, 0, stream.Len())
	assert.Empty(t, stream.Relocations())
	assert.Empty(t, stream.SegmentChanges())
	copied := stream.Copy()
	copied.removedEntries[0].Annotations[0].(*streamTestAnnotation).Value = "copy"
	assert.Equal(t, "entry", stream.RemovedEntries()[0].Annotations[0].(*streamTestAnnotation).Value)
	joined := NewStream()
	assert.NoError(t, joined.AppendStream(stream))
	assert.Equal(t, stream, joined)
	assert.NoError(t, joined.AppendStream(stream))
	assert.Len(t, joined.RemovedEntries(), 6)
}

func TestStreamRewriteRejectsInvalidCorrespondenceAtomically(t *testing.T) {
	for _, edits := range [][]EntryEdit{
		{{SourceIndex: -2}}, {{SourceIndex: 3}}, {{SourceIndex: NoSourceEntry}},
		{{SourceIndex: 0}, {SourceIndex: 1}, {SourceIndex: 2, Node: NewLabel("wrong")}},
		{{SourceIndex: 0, Node: NewLabel("wrong")}, {SourceIndex: 1}, {SourceIndex: 2}},
	} {
		stream := streamJoinFixture("entry", "input.asm")
		before := stream.Copy()
		assert.Error(t, stream.Rewrite(edits))
		assert.Equal(t, before, stream)
	}
}

func TestStreamRewriteInvalidatesAddressesAndRetainsReplacementSource(t *testing.T) {
	stream := streamJoinFixture("entry", "input.asm")
	assert.NoError(t, stream.ResolveSymbolValues(map[string]uint64{"entry": 0x8000}))
	before := stream.Copy()
	replacement := stream.At(2).Node.Copy()
	replacement.SetComment("replacement")
	assert.NoError(t, stream.Rewrite([]EntryEdit{{SourceIndex: 0}, {SourceIndex: 2, Node: replacement}, {SourceIndex: 1}}))
	assert.Equal(t, SymbolExpressionLocation, stream.Symbols()[0].Expression.Kind)
	assert.Equal(t, before.At(2).Position, stream.At(1).Position)
	assert.Equal(t, "replacement", InlineComment(stream.At(1).Node))
	assert.Equal(t, before.At(1), stream.At(2))
}

func TestStreamRewriteKeepsOnlyRemovedSourceRecords(t *testing.T) {
	plain := NewLabel("plain")
	commented := NewLabel("commented")
	commented.SetComment("source comment")
	stream := NewStreamFromNodes(plain, commented, &Comment{Message: "standalone"})
	before := stream.Copy()
	assert.NoError(t, stream.Rewrite(nil))
	assert.Equal(t, []Entry{before.At(1), before.At(2)}, stream.RemovedEntries())
	assert.NoError(t, stream.Rewrite(nil))
	assert.Len(t, stream.RemovedEntries(), 2)
	var absent *Stream
	assert.Error(t, absent.Rewrite(nil))
}

func TestStreamRewriteRetainsSourceComments(t *testing.T) {
	label := NewLabel("entry")
	label.SetComment("source")
	stream := NewStreamFromNodes(label)
	assert.NoError(t, stream.Rewrite([]EntryEdit{{SourceIndex: 0, Node: NewLabel("renamed")}}))
	assert.Equal(t, "source", InlineComment(stream.At(0).Node))
	before := stream.Copy()
	replacement := NewLabel("renamed")
	replacement.SetComment("conflict")
	assert.Error(t, stream.Rewrite([]EntryEdit{{SourceIndex: 0, Node: replacement}}))
	assert.Equal(t, before, stream)
}

func TestStreamRewriteReplacementOwnership(t *testing.T) {
	stream := streamJoinFixture("entry", "input.asm")
	stream.RecordState(streamJoinState{values: []int{1}}, streamJoinState{values: []int{2}})
	source := stream.entries[1]
	replacement := NewLabel("changed")
	assert.NoError(t, stream.Rewrite([]EntryEdit{
		{SourceIndex: 1, Node: replacement},
		{SourceIndex: 1, Node: replacement},
		{SourceIndex: 0},
		{SourceIndex: 2},
	}))

	replacement.SetComment("caller")
	source.Annotations[0].(*streamTestAnnotation).Value = "source"
	stream.entries[0].Node.SetComment("first")
	stream.entries[0].Annotations[0].(*streamTestAnnotation).Value = "first"
	assert.Equal(t, "", InlineComment(stream.entries[1].Node))
	assert.Equal(t, "entry", stream.entries[1].Annotations[0].(*streamTestAnnotation).Value)
	assert.Equal(t, source.Position, stream.entries[0].Position)
	assert.Equal(t, source.Boundary, stream.entries[0].Boundary)
}

func TestStreamRewriteRemovedPlainNodesAllocations(t *testing.T) {
	allocations := make([]float64, 2)
	for index, count := range []int{8, 4096} {
		entries := make([]Entry, count)
		for i := range entries {
			entries[i].Node = NewInstruction("lda", 0, NewNumber(uint64(i)), nil)
		}
		allocations[index] = testing.AllocsPerRun(10, func() {
			stream := &Stream{entries: entries}
			assert.NoError(t, stream.Rewrite(nil))
			assert.Equal(t, 0, stream.Len())
			assert.Empty(t, stream.RemovedEntries())
		})
	}
	assert.True(t, allocations[1] <= allocations[0], "removing plain nodes must not allocate per input node")
}

func BenchmarkStreamRewriteReplacements(b *testing.B) {
	nodes := make([]Node, 256)
	for index := range nodes {
		nodes[index] = NewInstruction("lda", 0, NewNumber(uint64(index)), nil)
	}
	stream := NewStreamFromNodes(nodes...)
	edits := make([]EntryEdit, len(nodes))
	for index, native := range nodes {
		edits[index] = EntryEdit{
			SourceIndex: index,
			Node:        native,
		}
	}
	b.ReportAllocs()

	for b.Loop() {
		assert.NoError(b, stream.Rewrite(edits))
	}
}
