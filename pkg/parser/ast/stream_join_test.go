package ast

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/assert"
)

type streamJoinState struct {
	values []int
}

func (state streamJoinState) CopyStreamState() any {
	return streamJoinState{values: append([]int(nil), state.values...)}
}

func TestStreamAppendStreamRetainsOwnedMetadata(t *testing.T) {
	t.Parallel()
	first := streamJoinFixture("first", "first.asm")
	second := streamJoinFixture("second", "second.asm")
	first.RecordState("entry", "middle")
	second.RecordState("middle", "exit")
	before := first.Copy()
	suffix := second.Copy()

	assert.NoError(t, first.AppendStream(second))
	assert.NoError(t, first.Validate())
	assert.Equal(t, before.Entries(), first.Entries()[:before.Len()])
	assert.Equal(t, suffix.Entries(), first.Entries()[before.Len():])
	assert.Equal(t, suffix, second)
	assert.Equal(t, before.Symbols()[0], first.Symbols()[0])
	symbol := suffix.Symbols()[0]
	symbol.EntryIndex += before.Len()
	assert.Equal(t, symbol, first.Symbols()[1])
	relocation := suffix.Relocations()[0]
	relocation.EntryIndex += before.Len()
	assert.Equal(t, relocation, first.Relocations()[1])
	change := suffix.SegmentChanges()[0]
	change.EntryIndex += before.Len()
	assert.Equal(t, change, first.SegmentChanges()[1])
	initial, final, ok := StateSnapshots[string](first)
	assert.True(t, ok)
	assert.Equal(t, "entry", initial)
	assert.Equal(t, "exit", final)

	second.entries[1].Annotations[0].(*streamTestAnnotation).Value = "changed"
	assert.Equal(t, "second", first.At(before.Len() + 1).Annotations[0].(*streamTestAnnotation).Value)
	first.entries[before.Len()+1].Annotations[0].(*streamTestAnnotation).Value = "destination"
	assert.Equal(t, "changed", second.At(1).Annotations[0].(*streamTestAnnotation).Value)
}

func TestStreamAppendStreamRejectsInvalidJoinAtomically(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"state", "missing-state", "entry", "symbol", "relocation", "segment", "nil"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			first := streamJoinFixture("first", "first.asm")
			second := streamJoinFixture("second", "second.asm")
			first.RecordState("entry", "middle")
			second.RecordState("middle", "exit")
			switch name {
			case "state":
				second.RecordState("different", "exit")
			case "missing-state":
				second.RecordState(nil, nil)
			case "entry":
				second.entries[0].Position.Line = -1
			case "symbol":
				second.symbols[0].EntryIndex = second.Len()
			case "relocation":
				second.relocations[0].EntryIndex = second.Len()
			case "segment":
				second.segmentChanges[0].EntryIndex = second.Len()
			case "nil":
				second = nil
			}
			before := first.Copy()
			suffix := second.Copy()
			assert.Error(t, first.AppendStream(second))
			assert.Equal(t, before, first)
			assert.Equal(t, suffix, second)
		})
	}
}

func TestStreamAppendStreamEmptyAndSelf(t *testing.T) {
	t.Parallel()
	stream := streamJoinFixture("entry", "entry.asm")
	stream.RecordState("entry", "entry")
	before := stream.Copy()
	joined := NewStream()
	assert.NoError(t, joined.AppendStream(stream))
	assert.Equal(t, before, joined)
	assert.NoError(t, joined.AppendStream(NewStream()))
	assert.Equal(t, before, joined)
	assert.NoError(t, joined.AppendStream(joined))
	assert.Equal(t, before.Entries(), joined.Entries()[before.Len():])
	assert.Equal(t, before.Len()+1, joined.Symbols()[1].EntryIndex)

	emptyState := NewStream()
	emptyState.RecordState("entry", "exit")
	assert.NoError(t, stream.AppendStream(emptyState))
	initial, final, ok := StateSnapshots[string](stream)
	assert.True(t, ok)
	assert.Equal(t, "entry", initial)
	assert.Equal(t, "exit", final)
	assert.Equal(t, before.Entries(), stream.Entries())
	var missing *Stream
	assert.Error(t, missing.AppendStream(before))
	invalid := NewStream(Entry{})
	assert.Error(t, invalid.AppendStream(before))
}

func TestStreamAppendStreamCopiesMutableState(t *testing.T) {
	t.Parallel()
	first := streamJoinFixture("first", "first.asm")
	second := streamJoinFixture("second", "second.asm")
	first.RecordState(streamJoinState{values: []int{1}}, streamJoinState{values: []int{2}})
	second.RecordState(streamJoinState{values: []int{2}}, streamJoinState{values: []int{3}})
	assert.NoError(t, first.AppendStream(second))
	second.finalState.(streamJoinState).values[0] = 9
	initial, final, ok := StateSnapshots[streamJoinState](first)
	assert.True(t, ok)
	assert.Equal(t, []int{1}, initial.values)
	assert.Equal(t, []int{3}, final.values)
	final.values[0] = 8
	assert.Equal(t, []int{3}, first.finalState.(streamJoinState).values)
}

func streamJoinFixture(name, source string) *Stream {
	address := NewData(AddressType, 2)
	address.ReferenceType = FullAddress
	address.Values = []*expression.Expression{expression.New(token.Token{
		Type:  token.Identifier,
		Value: name,
	})}
	label := NewEntry(NewLabel(name), SourcePosition{
		Source: source,
		Line:   2,
		Column: 1,
	})
	label.Annotations = []Annotation{&streamTestAnnotation{Value: name}}
	label.Boundary = BoundaryBefore | BoundaryAfter
	stream := NewStream(
		NewEntry(NewSegment("code"), SourcePosition{
			Source: source,
			Line:   1,
			Column: 1,
		}),
		label,
		NewEntry(address, SourcePosition{
			Source: source,
			Line:   3,
			Column: 1,
		}),
	)
	stream.RecordSymbol(Symbol{
		EntryIndex: 1,
		Kind:       LabelSymbol,
		Name:       name,
		Segment:    "code",
		Expression: NewLocationSymbolExpression(),
		Position:   label.Position,
	})
	stream.RecordRelocation(Relocation{
		EntryIndex: 2,
		Kind:       AbsoluteRelocation,
		Expression: NewSymbolExpression(name, 0, FullAddress),
		Width:      WidthWord,
		ByteOrder:  ByteOrderLittle,
	})
	stream.RecordSegmentChange(SegmentChange{
		EntryIndex: 0,
		Name:       "code",
		Alignment:  16,
		ByteOrder:  ByteOrderLittle,
	})
	return stream
}
