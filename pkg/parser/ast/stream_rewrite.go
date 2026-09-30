package ast

import (
	"errors"
	"fmt"
)

// NoSourceEntry identifies an inserted entry without a source entry.
const NoSourceEntry = -1

// EntryEdit gives the source of one result entry before a stream rewrite.
// Repeated source indices copy an entry. Omitted indices remove an entry.
type EntryEdit struct {
	// SourceIndex is an index in the unchanged input stream, or NoSourceEntry.
	SourceIndex int
	// Node replaces the source node. Nil retains the source node.
	// A replacement inherits the source comment; a different comment rejects the edit.
	// An inserted entry requires a node and has an unknown source position.
	Node Node
}

// Rewrite atomically applies an explicit source-to-result entry map.
// It retains source metadata through moves, copies, and node replacements.
// Removed entries with source metadata remain available through RemovedEntries.
// They do not emit code.
// Symbols are rebuilt because edits can change their addresses and segments.
// Retained relocations and segment changes must still match their result nodes.
// The caller must also validate target state and instruction legality with its codec.
// Owned node payloads are read-only. Unchanged entries can retain these payloads.
// Duplicate entries and replacement nodes have independent copies.
// Result nodes have no native edit handles.
func (stm *Stream) Rewrite(edits []EntryEdit) error {
	if err := stm.Validate(); err != nil {
		return err
	}
	return stm.rewriteValidatedSource(edits)
}

// RemovedEntries returns independent copies of entries removed by explicit rewrites.
// Entries are ordered by rewrite, then by their order in that rewrite's input.
func (stm *Stream) RemovedEntries() []Entry {
	return copyEntries(stm.removedEntries)
}

// rewriteValidatedSource requires a validated, unchanged source revision.
// It validates the complete output before publication.
func (stm *Stream) rewriteValidatedSource(edits []EntryEdit) error {
	candidate := &Stream{
		entries:        make([]Entry, len(edits)),
		removedEntries: copyEntries(stm.removedEntries),
		initialState:   copyStreamState(stm.initialState),
		finalState:     copyStreamState(stm.finalState),
	}
	destinations := make([]rewriteDestinations, stm.Len())
	for index, edit := range edits {
		if edit.SourceIndex < NoSourceEntry || edit.SourceIndex >= stm.Len() {
			return fmt.Errorf("%w: edit %d has source index %d", ErrInvalidStream, index, edit.SourceIndex)
		}
		var source Entry
		if edit.SourceIndex != NoSourceEntry {
			source = stm.entries[edit.SourceIndex]
			destinations[edit.SourceIndex].add(index)
		}
		replacement := edit.Node
		if replacement == nil && edit.SourceIndex != NoSourceEntry && len(destinations[edit.SourceIndex].indices) > 1 {
			replacement = source.Node
		}
		entry, err := copyRewriteEntry(source, replacement)
		if err != nil {
			return fmt.Errorf("%w: edit %d: %w", ErrInvalidStream, index, err)
		}
		candidate.entries[index] = entry
	}
	candidate.rewriteMetadata(stm, destinations)
	if err := candidate.RebuildSymbols(); err != nil {
		return fmt.Errorf("rebuilding rewritten symbols: %w", err)
	}
	if err := candidate.Validate(); err != nil {
		return fmt.Errorf("validating rewritten stream: %w", err)
	}
	candidate.revision = &streamRevision{}
	*stm = *candidate
	return nil
}

func (stm *Stream) rewriteMetadata(source *Stream, destinations []rewriteDestinations) {
	stm.relocations = nil
	for _, relocation := range source.relocations {
		for _, index := range destinations[relocation.EntryIndex].indices {
			copied := relocation
			copied.EntryIndex = index
			copied.Expression = relocation.Expression.Copy()
			stm.relocations = append(stm.relocations, copied)
		}
	}
	stm.segmentChanges = nil
	for _, change := range source.segmentChanges {
		for _, index := range destinations[change.EntryIndex].indices {
			copied := change
			copied.EntryIndex = index
			stm.segmentChanges = append(stm.segmentChanges, copied)
		}
	}
	for index, entry := range source.entries {
		if len(destinations[index].indices) == 0 && entryHasSourceMetadata(entry) {
			stm.removedEntries = append(stm.removedEntries, entry.Copy())
		}
	}
}

type rewriteDestinations struct {
	indices []int
	// first stores one destination without a separate allocation.
	first [1]int
}

func (destinations *rewriteDestinations) add(index int) {
	if len(destinations.indices) == 0 {
		destinations.first[0] = index
		destinations.indices = destinations.first[:]
		return
	}
	destinations.indices = append(destinations.indices, index)
}

func copyRewriteEntry(source Entry, replacement Node) (Entry, error) {
	original := source.Node
	if replacement == nil {
		if carrier, ok := original.(entryCarrier); ok && carrier.entryHandle() == nil {
			source.Node = nil
			copied := source.Copy()
			copied.Node = original
			if instruction, ok := original.(Instruction); ok {
				// Instruction payload sharing did not improve the full build.
				copied.Node = instruction.Copy()
			}
			return copied, nil
		}
	}
	if replacement != nil {
		source.Node = replacement
	}
	copied := source.Copy()
	if carrier, ok := copied.Node.(entryCarrier); ok {
		carrier.setEntryHandle(nil)
	}
	if replacement != nil {
		if err := retainEntryComment(original, copied.Node); err != nil {
			return Entry{}, err
		}
	}
	return copied, nil
}

func entryHasSourceMetadata(entry Entry) bool {
	if entry.Position != (SourcePosition{}) || len(entry.Annotations) != 0 || entry.Boundary != BoundaryNone {
		return true
	}
	if comment, ok := entry.Node.(*Comment); ok {
		return comment.Message != ""
	}
	return InlineComment(entry.Node) != ""
}

func retainEntryComment(source, replacement Node) error {
	comment := entryNodeComment(source)
	if comment == "" {
		return nil
	}
	if next := entryNodeComment(replacement); next != "" && next != comment {
		return errors.New("replacement changes the source comment")
	}
	replacement.SetComment(comment)
	return nil
}

func entryNodeComment(node Node) string {
	if comment, ok := node.(*Comment); ok {
		return comment.Message
	}
	return InlineComment(node)
}
