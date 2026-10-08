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
	stream := NewStream(NewEntry(NewInstruction("lda", 0, NewNumber(1), nil), SourcePosition{
		Source: "input.asm",
		Line:   7,
	}))
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

func TestNodeEditCommitClearsPublishedHandles(t *testing.T) {
	stream := NewStreamFromNodes(NewInstruction("lda", 0, NewNumber(1), nil))
	edit, err := stream.EditNodes()
	assert.NoError(t, err)
	candidate := edit.At(0)
	handle := candidate.(entryCarrier).entryHandle()
	assert.NotNil(t, handle)
	assert.NoError(t, edit.Commit([]Node{candidate, candidate}))
	assert.True(t, candidate.(entryCarrier).entryHandle() == handle)
	assert.Nil(t, stream.entries[0].Node.(entryCarrier).entryHandle())
	assert.Nil(t, stream.entries[1].Node.(entryCarrier).entryHandle())

	candidate.SetComment("caller")
	stream.entries[0].Node.SetComment("first")
	assert.Equal(t, "", InlineComment(stream.entries[1].Node))
}

func TestNodeEditReplaceMatchesCommit(t *testing.T) {
	for _, operation := range []string{"remove", "insert", "move", "copy", "replace", "empty", "all"} {
		t.Run(operation, func(t *testing.T) {
			stream := streamJoinFixture("entry", "input.asm")
			stream.RecordState(streamJoinState{values: []int{1}}, streamJoinState{values: []int{2}})
			reference := stream.Copy()
			edit, err := stream.EditNodes()
			assert.NoError(t, err)
			expectedEdit, err := reference.EditNodes()
			assert.NoError(t, err)
			start, end, replacement := nodeEditReplacement(edit, operation)
			assert.NoError(t, edit.Replace(start, end, replacement))
			start, end, replacement = nodeEditReplacement(expectedEdit, operation)
			nodes := expectedEdit.Nodes()
			result := append([]Node(nil), nodes[:start]...)
			result = append(result, replacement...)
			result = append(result, nodes[end:]...)
			assert.NoError(t, expectedEdit.Commit(result))
			assert.Equal(t, reference, stream)
			assert.ErrorContains(t, edit.Replace(0, 0, nil), "stale")
		})
	}
}

func TestNodeEditReplaceRejectsAtomically(t *testing.T) {
	stream := streamJoinFixture("entry", "input.asm")
	edit, err := stream.EditNodes()
	assert.NoError(t, err)
	foreign, err := stream.Copy().EditNodes()
	assert.NoError(t, err)
	before := stream.Copy()
	assert.ErrorIs(t, edit.Replace(-1, 0, nil), ErrInvalidStream)
	assert.ErrorIs(t, edit.Replace(2, 1, nil), ErrInvalidStream)
	assert.ErrorIs(t, edit.Replace(0, 4, nil), ErrInvalidStream)
	assert.ErrorIs(t, edit.Replace(0, 1, foreign.Nodes()), ErrInvalidStream)
	assert.ErrorIs(t, edit.Replace(0, 1, []Node{nil}), ErrInvalidStream)
	var absent *Instruction
	assert.ErrorIs(t, edit.Replace(0, 1, []Node{absent}), ErrInvalidStream)
	assert.Equal(t, before, stream)
	assert.NoError(t, edit.Replace(0, 0, nil))
}

func nodeEditReplacement(edit *NodeEdit, operation string) (int, int, []Node) {
	switch operation {
	case "remove":
		return 1, 2, nil
	case "insert":
		return 2, 2, []Node{NewLabel("inserted")}
	case "move":
		return 1, 3, []Node{edit.At(2), edit.At(1)}
	case "copy":
		return 2, 3, []Node{edit.At(2), edit.At(2)}

	case "replace":
		replacement := edit.At(1).(Label)
		replacement.Name = "renamed"
		return 1, 2, []Node{replacement}

	case "empty":
		return 2, 2, nil
	default:
		return 0, edit.Len(), nil
	}
}
