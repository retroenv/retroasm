package ast

import (
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

type unsupportedEditNode struct{}

func (unsupportedEditNode) Copy() Node        { return unsupportedEditNode{} }
func (unsupportedEditNode) SetComment(string) {}

func TestNodeEditRetainsExplicitSources(t *testing.T) {
	stream := streamJoinFixture("entry", "input.asm")
	before := stream.Copy()
	edit, err := stream.EditNodes()
	assert.NoError(t, err)
	assert.Equal(t, 3, edit.Len())
	nodes := edit.Nodes()
	assert.NoError(t, edit.Commit([]Node{nodes[0], nodes[2].Copy(), nodes[1], nodes[2], NewLabel("inserted")}))
	assert.Equal(t, before.At(0), stream.At(0))
	assert.Equal(t, before.At(2), stream.At(1))
	assert.Equal(t, before.At(1), stream.At(2))
	assert.Equal(t, before.At(2), stream.At(3))
	assert.Equal(t, SourcePosition{}, stream.At(4).Position)
	assert.Equal(t, 1, stream.Relocations()[0].EntryIndex)
	assert.Equal(t, 3, stream.Relocations()[1].EntryIndex)
	assert.Error(t, edit.Commit(nodes))
	next, err := stream.EditNodes()
	assert.NoError(t, err)
	assert.NoError(t, next.Commit([]Node{next.At(0), next.At(1), next.At(3), next.At(4)}))
	assert.Equal(t, []Entry{before.At(1)}, stream.RemovedEntries())
}

func TestNodeEditCandidateIndependenceAndRejection(t *testing.T) {
	stream := NewStream(NewEntry(NewInstruction("lda", 0, NewNumber(1), nil), SourcePosition{Source: "input.asm", Line: 7}))
	before := stream.Copy()
	edit, err := stream.EditNodes()
	assert.NoError(t, err)
	candidate := edit.At(0).(Instruction)
	candidate.SetComment("candidate")
	assert.Equal(t, "", InlineComment(edit.At(0)))
	assert.Equal(t, before, stream)
	assert.NoError(t, edit.Commit([]Node{candidate}))
	candidate.SetComment("later")
	assert.Equal(t, "candidate", InlineComment(stream.At(0).Node))
	assert.Equal(t, before.At(0).Position, stream.At(0).Position)

	current, err := stream.EditNodes()
	assert.NoError(t, err)
	foreign, err := stream.Copy().EditNodes()
	assert.NoError(t, err)
	before = stream.Copy()
	assert.Error(t, current.Commit(foreign.Nodes()))
	assert.Error(t, current.Commit(edit.Nodes()))
	assert.Error(t, current.Commit([]Node{nil}))
	var absent *Instruction
	assert.Error(t, current.Commit([]Node{absent}))
	assert.Equal(t, before, stream)
	assert.NoError(t, current.Commit(current.Nodes()))
}

func TestNodeEditRevisions(t *testing.T) {
	for _, mutation := range []string{"append", "replace", "rewrite", "rename", "symbols", "resolve", "symbol", "relocation", "segment", "state", "join"} {
		t.Run(mutation, func(t *testing.T) {
			stream := streamJoinFixture("entry", "input.asm")
			edit, err := stream.EditNodes()
			assert.NoError(t, err)
			mutateNodeEditStream(t, stream, mutation)
			before := stream.Copy()
			assert.ErrorContains(t, edit.Commit(edit.Nodes()), "stale")
			assert.Equal(t, before, stream)
		})
	}
}

func TestNodeEditCopiesHaveIndependentRevisions(t *testing.T) {
	stream := NewStreamFromNodes(NewLabel("entry"))
	edit, err := stream.EditNodes()
	assert.NoError(t, err)
	copied := stream.Copy()
	copied.Append(NewEntry(NewLabel("copy"), SourcePosition{}))
	assert.NoError(t, edit.Commit(edit.Nodes()))
	assert.Equal(t, 2, copied.Len())
	assert.Equal(t, 1, stream.Len())
	var empty Stream
	emptyEdit, err := empty.EditNodes()
	assert.NoError(t, err)
	assert.NoError(t, emptyEdit.Commit([]Node{NewLabel("new")}))
	var absent *Stream
	_, err = absent.EditNodes()
	assert.Error(t, err)
}

func mutateNodeEditStream(t *testing.T, stream *Stream, mutation string) {
	t.Helper()
	switch mutation {
	case "append":
		stream.Append(NewEntry(NewLabel("new"), SourcePosition{}))
	case "replace":
		assert.NoError(t, stream.Replace(1, 2, []Entry{stream.At(1)}))
	case "rewrite":
		assert.NoError(t, stream.Rewrite([]EntryEdit{{SourceIndex: 0}, {SourceIndex: 1}, {SourceIndex: 2}}))
	case "rename":
		assert.NoError(t, stream.RenameSymbols(map[string]string{"entry": "renamed"}))
	case "symbols":
		assert.NoError(t, stream.RebuildSymbols())
	case "resolve":
		assert.NoError(t, stream.ResolveSymbolValues(map[string]uint64{"entry": 0x8000}))
	case "symbol":
		stream.RecordSymbol(stream.Symbols()[0])
	case "relocation":
		stream.RecordRelocation(stream.Relocations()[0])
	case "segment":
		stream.RecordSegmentChange(stream.SegmentChanges()[0])
	case "state":
		stream.RecordState(1, 2)
	case "join":
		assert.NoError(t, stream.AppendStream(NewStreamFromNodes(NewLabel("new"))))
	}
}

func TestNodeEditRejectedMutationsKeepViewValid(t *testing.T) {
	stream := streamJoinFixture("entry", "input.asm")
	edit, err := stream.EditNodes()
	assert.NoError(t, err)
	before := stream.Copy()
	assert.Error(t, stream.Replace(-1, 0, nil))
	assert.Error(t, stream.RenameSymbols(map[string]string{"entry": ""}))
	assert.Error(t, stream.AppendStream(nil))
	assert.Equal(t, before, stream)
	assert.NoError(t, edit.Commit(edit.Nodes()))
}

func TestNodeEditRejectsInvalidTypedNilSource(t *testing.T) {
	var absent *Instruction
	stream := &Stream{entries: []Entry{{Node: absent}}}
	_, err := stream.EditNodes()
	assert.ErrorIs(t, err, ErrInvalidStream)
}

func TestNodeEditRejectsCommentLoss(t *testing.T) {
	instruction := NewInstruction("lda", 0, NewNumber(1), nil)
	instruction.SetComment("original")
	stream := NewStreamFromNodes(instruction)
	edit, err := stream.EditNodes()
	assert.NoError(t, err)
	before := stream.Copy()
	candidate := edit.At(0)
	candidate.SetComment("conflict")
	assert.Error(t, edit.Commit([]Node{candidate}))
	assert.Equal(t, before, stream)
	candidate.SetComment("")
	assert.NoError(t, edit.Commit([]Node{candidate}))
	assert.Equal(t, "original", InlineComment(stream.At(0).Node))
}

func TestNodeEditRejectsUnsupportedNodes(t *testing.T) {
	stream := NewStreamFromNodes(unsupportedEditNode{})
	_, err := stream.EditNodes()
	assert.ErrorIs(t, err, ErrInvalidStream)
	stream = NewStreamFromNodes(NewLabel("entry"))
	edit, err := stream.EditNodes()
	assert.NoError(t, err)
	before := stream.Copy()
	assert.Error(t, edit.Commit([]Node{unsupportedEditNode{}}))
	assert.Equal(t, before, stream)
}
