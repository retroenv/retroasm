package ast

import (
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

func TestNodeEditPublicationValidatesOutput(t *testing.T) {
	for _, commit := range []bool{false, true} {
		name := "range"
		if commit {
			name = "commit"
		}
		t.Run(name, func(t *testing.T) {
			stream := NewStreamFromNodes(NewLabel("entry"))
			assert.NoError(t, stream.RebuildSymbols())
			before := stream.Copy()
			view, err := stream.EditNodes()
			assert.NoError(t, err)
			invalid := NewLabel("")
			if commit {
				err = view.Commit([]Node{view.At(0), invalid})
			} else {
				err = view.Replace(1, 1, []Node{invalid})
			}
			assert.Error(t, err)
			assert.Equal(t, before, stream)
			assert.NoError(t, view.Commit(view.Nodes()))
			assert.Equal(t, before, stream)
		})
	}
}

func TestNodeEditPublicationValidatesRelocations(t *testing.T) {
	for _, commit := range []bool{false, true} {
		name := "range"
		if commit {
			name = "commit"
		}
		t.Run(name, func(t *testing.T) {
			stream := streamJoinFixture("entry", "input.asm")
			before := stream.Copy()
			view, err := stream.EditNodes()
			assert.NoError(t, err)
			candidate := view.At(2).(Data)
			candidate.Width = 1
			if commit {
				nodes := view.Nodes()
				nodes[2] = candidate
				err = view.Commit(nodes)
			} else {
				err = view.Replace(2, 3, []Node{candidate})
			}
			assert.ErrorContains(t, err, "relocation")
			assert.Equal(t, before, stream)
			assert.NoError(t, view.Commit(view.Nodes()))
			assert.Equal(t, before, stream)
		})
	}
}
