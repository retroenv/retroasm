package ast

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/assert"
)

func TestStreamRenameSymbolsRetainsMetadata(t *testing.T) {
	stream := streamJoinFixture("entry", "input.asm")
	stream.RecordState(streamJoinState{values: []int{1}}, streamJoinState{values: []int{2}})
	before := stream.Copy()
	assert.NoError(t, stream.RenameSymbols(map[string]string{"entry": "renamed"}))
	assert.Equal(t, "renamed", stream.Symbols()[0].Name)
	assert.Equal(t, "renamed", stream.Relocations()[0].Expression.Symbol)
	for index, entry := range before.Entries() {
		actual := stream.At(index)
		assert.Equal(t, entry.Position, actual.Position)
		assert.Equal(t, entry.Annotations, actual.Annotations)
		assert.Equal(t, entry.Boundary, actual.Boundary)
	}
	assert.NoError(t, stream.RenameSymbols(map[string]string{"renamed": "entry"}))
	assert.Equal(t, before, stream)
	stream.entries[1].Annotations[0].(*streamTestAnnotation).Value = "changed"
	assert.Equal(t, "entry", before.At(1).Annotations[0].(*streamTestAnnotation).Value)
}

func TestStreamRenameSymbolsSimultaneous(t *testing.T) {
	stream := NewStreamFromNodes(NewLabel("first"), NewLabel("second"))
	assert.NoError(t, stream.RenameSymbols(map[string]string{"first": "second", "second": "first"}))
	assert.Equal(t, "second", stream.At(0).Node.(Label).Name)
	assert.Equal(t, "first", stream.At(1).Node.(Label).Name)
	assert.NoError(t, stream.RenameSymbols(map[string]string{"first": "second", "second": "third"}))
	assert.Equal(t, "third", stream.At(0).Node.(Label).Name)
	assert.Equal(t, "second", stream.At(1).Node.(Label).Name)
}

func TestStreamRenameSymbolsRejectsAtomically(t *testing.T) {
	for _, names := range []map[string]string{{"first": "second"}, {"first": "same", "second": "same"}, {"first": ""}, {"": "other"}} {
		stream := NewStreamFromNodes(NewLabel("first"), NewLabel("second"))
		before := stream.Copy()
		assert.Error(t, stream.RenameSymbols(names))
		assert.Equal(t, before, stream)
	}
	stream := streamJoinFixture("entry", "input.asm")
	stream.entries[0].Position.Line = -1
	before := stream.Copy()
	assert.Error(t, stream.RenameSymbols(map[string]string{"entry": "other"}))
	assert.Equal(t, before, stream)
	var absent *Stream
	assert.Error(t, absent.RenameSymbols(nil))
	assert.NoError(t, NewStream().RenameSymbols(nil))
}

func TestStreamRenameSymbolsAliasExpressions(t *testing.T) {
	alias := NewAlias("alias")
	alias.Expression = expression.New(token.Token{
		Type:  token.Identifier,
		Value: "entry",
	}, token.Token{
		Type:  token.Plus,
		Value: "+",
	}, token.Token{
		Type:  token.Number,
		Value: "1",
	})
	alias.Expression.SetEvaluateOnce(true)
	stream := NewStreamFromNodes(NewLabel("entry"), alias)
	assert.NoError(t, stream.RebuildSymbols())
	before := stream.Copy()
	assert.NoError(t, stream.RenameSymbols(map[string]string{"entry": "target", "alias": "renamed", "1": "invalid"}))
	renamed := stream.At(1).Node.(Alias)
	assert.Equal(t, "renamed", renamed.Name)
	assert.Equal(t, "target", renamed.Expression.Tokens()[0].Value)
	assert.Equal(t, "1", renamed.Expression.Tokens()[2].Value)
	assert.True(t, renamed.Expression.IsEvaluatedOnce())
	assert.Equal(t, renamed.Expression, stream.Symbols()[1].Expression.Definition)
	assert.NoError(t, stream.RenameSymbols(map[string]string{"target": "entry", "renamed": "alias"}))
	assert.Equal(t, before, stream)
}

func TestStreamRenameSymbolsRejectsCaptureAndOpaqueNodes(t *testing.T) {
	for _, nodes := range [][]Node{
		{NewLabel("entry"), NewIdentifier("external")},
		{NewLabel("entry"), NewInstructionArgument("opaque")},
		{NewLabel("entry"), NewScope("scope")},
	} {
		stream := NewStreamFromNodes(nodes...)
		before := stream.Copy()
		assert.Error(t, stream.RenameSymbols(map[string]string{"entry": "external"}))
		assert.Equal(t, before, stream)
	}
	_, err := RewriteNodeSymbols(NewIdentifier("entry"), nil)
	assert.Error(t, err)
}
