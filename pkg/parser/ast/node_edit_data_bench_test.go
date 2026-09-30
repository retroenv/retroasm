package ast

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/assert"
)

func TestNodeEditComplexDataPublicationKeepsOwnership(t *testing.T) {
	data := NewData(DataType, 1)
	data.SetComment("table")
	data.Values = []*expression.Expression{expression.New(token.Token{
		Type:  token.Number,
		Value: "0",
	})}
	stream := NewStreamFromNodes(data)
	view, err := stream.EditNodes()
	assert.NoError(t, err)
	candidate := view.At(0).(Data)
	candidate.Values[0] = expression.New(token.Token{
		Type:  token.Number,
		Value: "1",
	})
	assert.NoError(t, view.Commit([]Node{candidate}))
	assert.Equal(t, "1", stream.At(0).Node.(Data).Values[0].Tokens()[0].Value)
	before := stream.Copy()
	candidate.Values[0] = expression.New(token.Token{
		Type:  token.Number,
		Value: "2",
	})
	candidate.SetComment("caller")
	assert.Equal(t, before, stream)
	assert.Equal(t, "0", view.At(0).(Data).Values[0].Tokens()[0].Value)
}

func BenchmarkNodeEditDataCommit(b *testing.B) {
	data := NewData(DataType, 1)
	data.Values = make([]*expression.Expression, 4096)
	for index := range data.Values {
		data.Values[index] = expression.New(token.Token{
			Type:  token.Number,
			Value: "0",
		})
	}
	stream := NewStreamFromNodes(data)
	b.ReportAllocs()
	for b.Loop() {
		view, err := stream.EditNodes()
		assert.NoError(b, err)
		assert.NoError(b, view.Commit(view.Nodes()))
	}
}
