package ast

import (
	"fmt"
	"reflect"
)

// AppendStream atomically appends an independent copy of source and its metadata.
// Entry indices are relative to the combined stream. The final state of the
// destination must equal the initial state of source. An empty stream without
// state is an identity; it does not change the other stream's state.
func (stm *Stream) AppendStream(source *Stream) error {
	if err := stm.Validate(); err != nil {
		return fmt.Errorf("validating destination stream: %w", err)
	}
	if err := source.Validate(); err != nil {
		return fmt.Errorf("validating source stream: %w", err)
	}
	if source.emptyWithoutState() {
		return nil
	}
	if stm.emptyWithoutState() {
		*stm = *source.Copy()
		stm.revision = &streamRevision{}
		return nil
	}
	if !reflect.DeepEqual(stm.finalState, source.initialState) {
		return fmt.Errorf("%w: stream states differ at join", ErrInvalidStream)
	}

	candidate := stm.Copy()
	suffix := source.Copy()
	offset := len(candidate.entries)
	candidate.entries = append(candidate.entries, suffix.entries...)
	candidate.removedEntries = append(candidate.removedEntries, suffix.removedEntries...)
	for _, symbol := range suffix.symbols {
		symbol.EntryIndex += offset
		candidate.symbols = append(candidate.symbols, symbol)
	}
	for _, relocation := range suffix.relocations {
		relocation.EntryIndex += offset
		candidate.relocations = append(candidate.relocations, relocation)
	}
	for _, change := range suffix.segmentChanges {
		change.EntryIndex += offset
		candidate.segmentChanges = append(candidate.segmentChanges, change)
	}
	candidate.finalState = suffix.finalState
	if err := candidate.Validate(); err != nil {
		return fmt.Errorf("validating combined stream: %w", err)
	}

	candidate.revision = &streamRevision{}
	*stm = *candidate
	return nil
}

func (stm *Stream) emptyWithoutState() bool {
	return len(stm.entries) == 0 && len(stm.removedEntries) == 0 && stm.initialState == nil && stm.finalState == nil
}
