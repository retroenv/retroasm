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
	handles  map[*entryHandle]int
	nodes    []Node
	revision *streamRevision
	stream   *Stream
}

// EditNodes returns an independent native view of the current stream.
// Each entry must use an AST node with the private source-handle contract.
func (stm *Stream) EditNodes() (*NodeEdit, error) {
	if err := stm.Validate(); err != nil {
		return nil, err
	}
	edit := &NodeEdit{
		handles:  make(map[*entryHandle]int, stm.Len()),
		nodes:    stm.Nodes(),
		revision: stm.revision,
		stream:   stm,
	}
	for index, node := range edit.nodes {
		carrier, ok := node.(entryCarrier)
		if !ok {
			return nil, fmt.Errorf("%w: node %T has no entry handle contract", ErrInvalidStream, node)
		}
		handle := &entryHandle{}
		carrier.setEntryHandle(handle)
		edit.handles[handle] = index
	}
	return edit, nil
}

// At returns an independent native node with its source handle.
func (edit *NodeEdit) At(index int) Node {
	return edit.nodes[index].Copy()
}

// Commit publishes nodes with their explicit source-entry correspondence.
// Handles from a different view are rejected, including handles from stale views.
func (edit *NodeEdit) Commit(nodes []Node) error {
	if edit.stream.revision != edit.revision {
		return fmt.Errorf("%w: native edit view is stale", ErrInvalidStream)
	}
	edits := make([]EntryEdit, len(nodes))
	for index, node := range nodes {
		if node == nil || reflect.ValueOf(node).Kind() == reflect.Pointer && reflect.ValueOf(node).IsNil() {
			return fmt.Errorf("%w: node %d is nil", ErrInvalidStream, index)
		}
		carrier, ok := node.(entryCarrier)
		if !ok {
			return fmt.Errorf("%w: node %T has no entry handle contract", ErrInvalidStream, node)
		}
		source := NoSourceEntry
		if handle := carrier.entryHandle(); handle != nil {
			var exists bool
			source, exists = edit.handles[handle]
			if !exists {
				return fmt.Errorf("%w: node %d has a foreign entry handle", ErrInvalidStream, index)
			}
		}
		edits[index] = EntryEdit{SourceIndex: source}
		copied := node.Copy()
		if carrier, ok := copied.(entryCarrier); ok {
			carrier.setEntryHandle(nil)
		}
		edits[index].Node = copied
	}
	return edit.stream.Rewrite(edits)
}

// Len returns the number of input entries.
func (edit *NodeEdit) Len() int { return len(edit.nodes) }

// Nodes returns independent native nodes with their source handles.
func (edit *NodeEdit) Nodes() []Node { return CopyNodes(edit.nodes) }

type entryCarrier interface {
	entryHandle() *entryHandle
	setEntryHandle(*entryHandle)
}
