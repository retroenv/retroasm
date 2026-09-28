package codec

import (
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/set"
)

func (c *Codec[T]) recordAssemblyMetadata(stream *ast.Stream) error {
	orderer, ok := any(c.configuration.Arch).(arch.ByteOrderer)
	if !ok {
		return ErrByteOrderUnsupported
	}
	order := orderer.ByteOrder()
	if order != ast.ByteOrderLittle && order != ast.ByteOrderBig {
		return fmt.Errorf("%w: %d", ErrByteOrderUnsupported, order)
	}
	if err := stream.RebuildSymbols(); err != nil {
		return err //nolint:wrapcheck // the codec adds operation context
	}

	segments := set.New[int]()
	for _, change := range stream.SegmentChanges() {
		segments.Add(change.EntryIndex)
	}
	relocations := set.New[relocationLocation]()
	for _, relocation := range stream.Relocations() {
		relocations.Add(relocationLocation{
			entryIndex: relocation.EntryIndex,
			byteOffset: relocation.ByteOffset,
		})
	}
	for index, entry := range stream.Entries() {
		switch node := entry.Node.(type) {
		case ast.Segment:
			if !segments.Contains(index) {
				stream.RecordSegmentChange(c.segmentChange(index, node, order))
			}
		case ast.Data:
			recordDataRelocations(stream, index, node, order, relocations)
		}
	}
	if err := stream.Validate(); err != nil {
		return err //nolint:wrapcheck // the codec adds operation context
	}
	return nil
}

func (c *Codec[T]) segmentChange(entryIndex int, segment ast.Segment, order ast.ByteOrder) ast.SegmentChange {
	change := ast.SegmentChange{
		EntryIndex: entryIndex,
		Name:       segment.Name,
		ByteOrder:  order,
	}
	name := strings.Trim(segment.Name, "\"'")
	if configured, ok := c.configuration.Segments[name]; ok {
		change.Alignment = ast.Alignment(configured.Align)
	}
	return change
}

type relocationLocation struct {
	byteOffset uint64
	entryIndex int
}

func recordDataRelocations(stream *ast.Stream, entryIndex int, data ast.Data, order ast.ByteOrder,
	recorded set.Set[relocationLocation]) {

	if data.Type != ast.AddressType {
		return
	}

	width := data.Width
	if data.ReferenceType != ast.FullAddress {
		width = 1
	}
	for valueIndex, value := range data.Values {
		byteOffset := uint64(valueIndex * width)
		if recorded.Contains(relocationLocation{
			entryIndex: entryIndex,
			byteOffset: byteOffset,
		}) {

			continue
		}
		symbol, addend, ok := ast.ParseSymbolReference(value)
		if !ok {
			continue
		}
		stream.RecordRelocation(ast.Relocation{
			EntryIndex: entryIndex,
			ByteOffset: byteOffset,
			Kind:       ast.AbsoluteRelocation,
			Expression: ast.NewSymbolExpression(symbol, addend, data.ReferenceType),
			Width:      ast.DataWidth(width),
			ByteOrder:  order,
		})
	}
}
