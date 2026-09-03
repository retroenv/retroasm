package ast

import (
	"errors"
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

type streamTestAnnotation struct {
	Value string
}

type invalidStreamMetadataCase struct {
	name   string
	stream *Stream
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

func TestStream_ValidateAcceptsCompleteMetadata(t *testing.T) {
	stream := NewStream(NewEntry(NewLabel("entry"), SourcePosition{Source: "input.asm", Line: 1, Column: 1}))
	stream.RecordSymbol(Symbol{
		Name:       "entry",
		Segment:    "code",
		Expression: NewAbsoluteSymbolExpression(0x8000),
		Position:   SourcePosition{Source: "input.asm", Line: 1, Column: 1},
	})
	stream.RecordRelocation(Relocation{
		EntryIndex: 0,
		Kind:       AbsoluteRelocation,
		Expression: NewSymbolExpression("entry", 0, FullAddress),
		Width:      WidthWord,
		ByteOrder:  ByteOrderLittle,
	})
	stream.RecordSegmentChange(SegmentChange{
		EntryIndex: 0,
		Name:       "code",
		Alignment:  16,
		ByteOrder:  ByteOrderLittle,
	})

	assert.NoError(t, stream.Validate())
}

func TestStream_ValidateRejectsInvalidMetadata(t *testing.T) {
	for _, test := range invalidStreamMetadataCases() {
		t.Run(test.name, func(t *testing.T) {
			err := test.stream.Validate()
			assert.Error(t, err)
			assert.True(t, errors.Is(err, ErrInvalidStream))
		})
	}
}

func invalidStreamMetadataCases() []invalidStreamMetadataCase {
	return []invalidStreamMetadataCase{
		{
			name:   "nil stream",
			stream: nil,
		},
		{
			name:   "nil node",
			stream: NewStream(NewEntry(nil, SourcePosition{})),
		},
		{
			name: "negative source position",
			stream: NewStream(NewEntry(
				NewLabel("entry"),
				SourcePosition{Line: -1},
			)),
		},
		{
			name: "unknown boundary",
			stream: NewStream(Entry{
				Node:     NewLabel("entry"),
				Boundary: OptimizationBoundary(4),
			}),
		},
		{
			name: "invalid symbol expression",
			stream: streamWithSymbols(Symbol{
				Name:       "entry",
				Expression: SymbolExpression{},
			}),
		},
		{
			name: "duplicate symbol",
			stream: streamWithSymbols(
				Symbol{Name: "entry", Expression: NewAbsoluteSymbolExpression(0)},
				Symbol{Name: "entry", Expression: NewAbsoluteSymbolExpression(1)},
			),
		},
		{
			name: "relocation outside stream",
			stream: streamWithRelocation(Relocation{
				EntryIndex: 1,
				Kind:       AbsoluteRelocation,
				Expression: NewSymbolExpression("entry", 0, FullAddress),
				Width:      WidthWord,
				ByteOrder:  ByteOrderLittle,
			}),
		},
		{
			name: "invalid segment alignment",
			stream: streamWithSegmentChange(SegmentChange{
				EntryIndex: 0,
				Name:       "code",
				Alignment:  3,
				ByteOrder:  ByteOrderLittle,
			}),
		},
	}
}

func streamWithSymbols(symbols ...Symbol) *Stream {
	stream := NewStreamFromNodes(NewLabel("entry"))
	for _, symbol := range symbols {
		stream.RecordSymbol(symbol)
	}
	return stream
}

func streamWithRelocation(relocation Relocation) *Stream {
	stream := NewStreamFromNodes(NewLabel("entry"))
	stream.RecordRelocation(relocation)
	return stream
}

func streamWithSegmentChange(change SegmentChange) *Stream {
	stream := NewStreamFromNodes(NewLabel("entry"))
	stream.RecordSegmentChange(change)
	return stream
}
