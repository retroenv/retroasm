package ast

import (
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

type streamTestAnnotation struct {
	Value string
}

func (ann *streamTestAnnotation) CopyStreamAnnotation() Annotation {
	if ann == nil {
		return (*streamTestAnnotation)(nil)
	}
	copied := *ann
	return &copied
}

func TestStream_OwnsEntriesAndMetadata(t *testing.T) {
	position := SourcePosition{Source: "input.asm", Line: 3, Column: 5, Offset: 12}
	entry := NewEntry(NewLabel("entry"), position)
	entry.Annotations = []Annotation{&streamTestAnnotation{Value: "hot"}}
	entry.Boundary = BoundaryBefore | BoundaryAfter
	stream := NewStream(entry, NewEntry(NewInstruction("nop", 0, nil, nil), SourcePosition{}))

	entry.Position.Line = 99
	entry.Annotations[0].(*streamTestAnnotation).Value = "changed"

	assert.Equal(t, 2, stream.Len())
	stored := stream.At(0)
	assert.Equal(t, position, stored.Position)
	assert.Equal(t, "hot", stored.Annotations[0].(*streamTestAnnotation).Value)
	assert.True(t, stored.Boundary.BlocksBefore())
	assert.True(t, stored.Boundary.BlocksAfter())
	assert.False(t, BoundaryNone.BlocksBefore())
	assert.False(t, BoundaryNone.BlocksAfter())

	entries := stream.Entries()
	entries[0].Annotations[0].(*streamTestAnnotation).Value = "external"
	assert.Equal(t, "hot", stream.At(0).Annotations[0].(*streamTestAnnotation).Value)

	nodes := stream.Nodes()
	assert.Len(t, nodes, 2)
	assert.Equal(t, "entry", SymbolName(nodes[0]))
}

func TestStream_RecordsTypedAssemblyMetadata(t *testing.T) {
	stream := NewStreamFromNodes(NewLabel("entry"), NewData(DataType, 1), NewSegment("code"))
	stream.RecordState("native", "emulation")
	stream.RecordSymbol(Symbol{
		Name:       "entry",
		Segment:    "code",
		Expression: NewAbsoluteSymbolExpression(0x8000),
	})
	stream.RecordRelocation(Relocation{
		EntryIndex: 1,
		ByteOffset: 0,
		Kind:       AbsoluteRelocation,
		Expression: NewSymbolExpression("entry", 2, HighAddressByte),
		Width:      WidthWord,
		ByteOrder:  ByteOrderLittle,
	})
	stream.RecordSegmentChange(SegmentChange{
		EntryIndex: 2,
		Name:       "code",
		Alignment:  16,
		ByteOrder:  ByteOrderLittle,
	})

	initial, final, ok := StateSnapshots[string](stream)
	assert.True(t, ok)
	assert.Equal(t, "native", initial)
	assert.Equal(t, "emulation", final)
	assert.Equal(t, uint64(0x8000), stream.Symbols()[0].Expression.Value)
	assert.Equal(t, "entry", stream.Relocations()[0].Expression.Symbol)
	assert.Equal(t, Alignment(16), stream.SegmentChanges()[0].Alignment)
	assert.True(t, Alignment(16).Valid())
	assert.False(t, Alignment(3).Valid())
	assert.True(t, WidthLong.Valid())
	assert.False(t, WidthUnknown.Valid())

	copied := stream.Copy()
	copied.RecordSymbol(Symbol{Name: "copy-only"})
	copied.Replace(0, 1, []Entry{NewEntry(NewLabel("copy-entry"), SourcePosition{})})
	assert.Equal(t, 3, stream.Len())
	assert.Len(t, stream.Symbols(), 1)
	assert.Equal(t, "entry", SymbolName(stream.At(0).Node))
	assert.Equal(t, "copy-entry", SymbolName(copied.At(0).Node))
}

func TestStream_RejectsMutableStateWithoutCopyContract(t *testing.T) {
	stream := NewStream()
	assert.Panics(t, func() {
		stream.RecordState([]int{1}, []int{2})
	})
}
