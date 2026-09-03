package ast

import (
	"errors"
	"testing"

	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
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
		EntryIndex: 0,
		Kind:       LabelSymbol,
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
	copied.Append(NewEntry(NewLabel("copy-entry"), SourcePosition{}))
	assert.Equal(t, 3, stream.Len())
	assert.Len(t, stream.Symbols(), 1)
	assert.Equal(t, "entry", SymbolName(stream.At(0).Node))
	assert.Equal(t, "copy-entry", SymbolName(copied.At(3).Node))
}

func TestStream_RejectsMutableStateWithoutCopyContract(t *testing.T) {
	stream := NewStream()
	assert.Panics(t, func() {
		stream.RecordState([]int{1}, []int{2})
	})
}

func TestStream_ValidateAcceptsCompleteMetadata(t *testing.T) {
	position := SourcePosition{Source: "input.asm", Line: 1, Column: 1}
	address := NewData(AddressType, 2)
	address.ReferenceType = FullAddress
	address.Values = []*expression.Expression{expression.New(token.Token{Type: token.Identifier, Value: "entry"})}
	stream := NewStream(
		NewEntry(NewLabel("entry"), position),
		NewEntry(address, SourcePosition{Source: "input.asm", Line: 2, Column: 1}),
		NewEntry(NewSegment("code"), SourcePosition{Source: "input.asm", Line: 3, Column: 1}),
	)
	stream.RecordSymbol(Symbol{
		EntryIndex: 0,
		Kind:       LabelSymbol,
		Name:       "entry",
		Segment:    "code",
		Expression: NewAbsoluteSymbolExpression(0x8000),
		Position:   position,
	})
	stream.RecordRelocation(Relocation{
		EntryIndex: 1,
		Kind:       AbsoluteRelocation,
		Expression: NewSymbolExpression("entry", 0, FullAddress),
		Width:      WidthWord,
		ByteOrder:  ByteOrderLittle,
	})
	stream.RecordSegmentChange(SegmentChange{
		EntryIndex: 2,
		Name:       "code",
		Alignment:  16,
		ByteOrder:  ByteOrderLittle,
	})

	assert.NoError(t, stream.Validate())
}

func TestStream_ValidateInstructionRelocationMatchesExpression(t *testing.T) {
	t.Parallel()

	instruction := NewInstruction("jmp", 0, NewLabel("target"), []Modifier{{
		Operator: NewOperator("-"),
		Value:    "1",
	}})
	relocation := Relocation{
		Kind:       AbsoluteRelocation,
		Expression: NewSymbolExpression("target", -1, FullAddress),
		Width:      WidthWord,
		ByteOrder:  ByteOrderLittle,
	}
	stream := NewStreamFromNodes(instruction)
	stream.RecordRelocation(relocation)
	assert.NoError(t, stream.Validate())

	relocation.Expression = NewSymbolExpression("target", 0, FullAddress)
	stream = NewStreamFromNodes(instruction)
	stream.RecordRelocation(relocation)
	assert.ErrorIs(t, stream.Validate(), ErrInvalidStream)

	relocation.Expression = NewSymbolExpression("target", -1, LowAddressByte)
	stream = NewStreamFromNodes(instruction)
	stream.RecordRelocation(relocation)
	assert.ErrorIs(t, stream.Validate(), ErrInvalidStream)
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

func TestStream_ReplaceReindexesCompatibleMetadataAtomically(t *testing.T) {
	position := SourcePosition{Source: "input.asm", Line: 1, Column: 1}
	address := NewData(AddressType, 2)
	address.ReferenceType = FullAddress
	address.Values = []*expression.Expression{expression.New(token.Token{Type: token.Identifier, Value: "entry"})}
	stream := NewStream(
		NewEntry(NewLabel("entry"), position),
		NewEntry(address, SourcePosition{Source: "input.asm", Line: 2, Column: 1}),
		NewEntry(NewSegment("code"), SourcePosition{Source: "input.asm", Line: 3, Column: 1}),
	)
	stream.RecordSymbol(Symbol{
		EntryIndex: 0,
		Kind:       LabelSymbol,
		Name:       "entry",
		Expression: NewLocationSymbolExpression(),
		Position:   position,
	})
	stream.RecordRelocation(Relocation{
		EntryIndex: 1,
		Kind:       AbsoluteRelocation,
		Expression: NewSymbolExpression("entry", 0, FullAddress),
		Width:      WidthWord,
		ByteOrder:  ByteOrderLittle,
	})
	stream.RecordSegmentChange(SegmentChange{
		EntryIndex: 2,
		Name:       "code",
		ByteOrder:  ByteOrderLittle,
	})

	err := stream.Replace(0, 0, []Entry{NewEntry(&Comment{Message: "header"}, SourcePosition{})})
	assert.NoError(t, err)
	assert.Equal(t, 1, stream.Symbols()[0].EntryIndex)
	assert.Equal(t, 2, stream.Relocations()[0].EntryIndex)
	assert.Equal(t, 3, stream.SegmentChanges()[0].EntryIndex)
	assert.NoError(t, stream.Validate())

	before := stream.Copy()
	err = stream.Replace(2, 3, nil)
	assert.ErrorIs(t, err, ErrInvalidStream)
	assert.Equal(t, before.Entries(), stream.Entries())
	assert.Equal(t, before.Relocations(), stream.Relocations())

	err = stream.Replace(2, 3, []Entry{NewEntry(NewInstruction("nop", 0, nil, nil), stream.At(2).Position)})
	assert.ErrorIs(t, err, ErrInvalidStream)
	assert.Equal(t, before.Entries(), stream.Entries())
	assert.Equal(t, before.Relocations(), stream.Relocations())
}

func TestStream_ResolveSymbolValuesPreservesDefinitionsAndInput(t *testing.T) {
	alias := NewAlias("constant")
	alias.SymbolReusable = true
	alias.Expression = expression.New(token.Token{Type: token.Number, Value: "1"})
	alias.Expression.SetEvaluateOnce(true)
	labelPosition := SourcePosition{Source: "input.asm", Line: 2, Column: 1}
	stream := NewStream(
		NewEntry(alias, SourcePosition{Source: "input.asm", Line: 1, Column: 1}),
		NewEntry(NewLabel("entry"), labelPosition),
	)
	stream.RecordSymbol(Symbol{
		EntryIndex: 0,
		Kind:       AliasSymbol,
		Name:       "constant",
		Expression: NewDefinitionSymbolExpression(alias.Expression),
		Position:   stream.At(0).Position,
	})
	stream.RecordSymbol(Symbol{
		EntryIndex: 1,
		Kind:       LabelSymbol,
		Name:       "entry",
		Expression: NewLocationSymbolExpression(),
		Position:   labelPosition,
	})
	assert.NoError(t, stream.Validate())

	external := stream.Symbols()
	external[0].Expression.Definition.AddTokens(token.Token{Type: token.Number, Value: "2"})
	assert.Len(t, stream.Symbols()[0].Expression.Definition.Tokens(), 1)

	err := stream.ResolveSymbolValues(map[string]uint64{"entry": 0x8000})
	assert.NoError(t, err)
	assert.Equal(t, SymbolExpressionDefinition, stream.Symbols()[0].Expression.Kind)
	assert.Equal(t, SymbolExpressionAbsolute, stream.Symbols()[1].Expression.Kind)
	assert.Equal(t, uint64(0x8000), stream.Symbols()[1].Expression.Value)
	assert.NoError(t, stream.Validate())
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
				Kind:       LabelSymbol,
				Name:       "entry",
				Expression: SymbolExpression{},
			}),
		},
		{
			name: "absolute symbol expression with addend",
			stream: streamWithSymbols(Symbol{
				Kind:       LabelSymbol,
				Name:       "entry",
				Expression: SymbolExpression{Kind: SymbolExpressionAbsolute, Addend: 1, ReferenceType: FullAddress},
			}),
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
	entries := make([]Entry, len(symbols))
	for index := range symbols {
		symbols[index].EntryIndex = index
		entries[index] = NewEntry(NewLabel(symbols[index].Name), symbols[index].Position)
	}
	stream := NewStream(entries...)
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
