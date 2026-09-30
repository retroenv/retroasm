package ast

import (
	"fmt"
	"reflect"
)

// These nonzero-size tokens identify edit sources and stream revisions.
// Their values have no assembly meaning.
type entryHandle struct{ _ byte }
type streamRevision struct{ _ byte }

// NodeEdit owns a native view with explicit handles for its input entries.
// Node.Copy and value copies retain handles. New nodes have no source entry.
// An accepted commit invalidates the view. Any other stream mutation also invalidates it.
type NodeEdit struct {
	handles map[*entryHandle]int
	// nodes retains read-only payloads from the source revision.
	nodes    []Node
	revision *streamRevision
	stream   *Stream
	// tokens gives each source index a stable handle.
	tokens []entryHandle
}

// EditNodes returns an independent native view of the current stream.
// Each entry must use an AST node with the private source-handle contract.
func (stm *Stream) EditNodes() (*NodeEdit, error) {
	if err := stm.Validate(); err != nil {
		return nil, err
	}
	edit := &NodeEdit{
		handles:  make(map[*entryHandle]int, stm.Len()),
		nodes:    make([]Node, stm.Len()),
		revision: stm.revision,
		stream:   stm,
		tokens:   make([]entryHandle, stm.Len()),
	}
	for index, entry := range stm.entries {
		_, ok := entry.Node.(entryCarrier)
		if !ok {
			return nil, fmt.Errorf("%w: node %T has no entry handle contract", ErrInvalidStream, entry.Node)
		}
		edit.nodes[index] = entry.Node
		edit.handles[&edit.tokens[index]] = index
	}
	return edit, nil
}

// At returns an independent native node with its source handle.
func (edit *NodeEdit) At(index int) Node {
	copied := edit.nodes[index].Copy()
	copied.(entryCarrier).setEntryHandle(&edit.tokens[index])
	return copied
}

// Commit publishes nodes with their explicit source-entry correspondence.
// Handles from a different view are rejected, including handles from stale views.
func (edit *NodeEdit) Commit(nodes []Node) error {
	if edit.stream.revision != edit.revision {
		return fmt.Errorf("%w: native edit view is stale", ErrInvalidStream)
	}
	edits := make([]EntryEdit, len(nodes))
	for index, node := range nodes {
		entry, err := edit.entryEdit(node, index)
		if err != nil {
			return err
		}
		edits[index] = entry
	}
	return edit.stream.rewriteValidatedSource(edits)
}

// Len returns the number of input entries.
func (edit *NodeEdit) Len() int { return len(edit.nodes) }

// Nodes returns independent native nodes with their source handles.
func (edit *NodeEdit) Nodes() []Node {
	nodes := make([]Node, edit.Len())
	for index := range nodes {
		nodes[index] = edit.At(index)
	}
	return nodes
}

// Replace publishes a half-open node range with the source checks from Commit.
// A rejected edit leaves the stream and this view unchanged.
func (edit *NodeEdit) Replace(start, end int, replacement []Node) error {
	if start < 0 || end < start || end > edit.Len() {
		return fmt.Errorf("%w: node replacement range %d:%d is outside %d nodes", ErrInvalidStream, start, end, edit.Len())
	}
	if edit.stream.revision != edit.revision {
		return fmt.Errorf("%w: native edit view is stale", ErrInvalidStream)
	}
	edits := make([]EntryEdit, edit.Len()-(end-start)+len(replacement))
	for index := range start {
		edits[index].SourceIndex = index
	}
	for index, node := range replacement {
		entry, err := edit.entryEdit(node, start+index)
		if err != nil {
			return err
		}
		edits[start+index] = entry
	}
	for index := end; index < edit.Len(); index++ {
		edits[start+len(replacement)+index-end].SourceIndex = index
	}
	return edit.stream.rewriteValidatedSource(edits)
}

func (edit *NodeEdit) entryEdit(node Node, index int) (EntryEdit, error) {
	if node == nil || reflect.ValueOf(node).Kind() == reflect.Pointer && reflect.ValueOf(node).IsNil() {
		return EntryEdit{}, fmt.Errorf("%w: node %d is nil", ErrInvalidStream, index)
	}
	carrier, ok := node.(entryCarrier)
	if !ok {
		return EntryEdit{}, fmt.Errorf("%w: node %T has no entry handle contract", ErrInvalidStream, node)
	}
	source := NoSourceEntry
	if handle := carrier.entryHandle(); handle != nil {
		var exists bool
		source, exists = edit.handles[handle]
		if !exists {
			return EntryEdit{}, fmt.Errorf("%w: node %d has a foreign entry handle", ErrInvalidStream, index)
		}
	}
	// The source handle establishes correspondence. Equality only avoids a copy.
	if source != NoSourceEntry && equalRetainedNode(node, edit.nodes[source]) {
		node = nil
	}
	return EntryEdit{
		SourceIndex: source,
		Node:        node,
	}, nil
}

type entryCarrier interface {
	entryHandle() *entryHandle
	setEntryHandle(*entryHandle)
}

// Use scalar comparisons here. Reflection can cost more than a node copy.
func equalRetainedNode(left, right Node) bool {
	value := left
	if instruction, ok := InstructionFromNode(left); ok {
		value = instruction.Argument
	}
	switch value.(type) {
	case nil, Number, *Number, Label, *Label, Identifier, *Identifier, Operator, *Operator, *Comment:
		return Equal(left, right)
	default:
		return false
	}
}
