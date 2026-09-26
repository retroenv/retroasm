package codec

import (
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

type relocationLocation struct {
	byteOffset uint64
	entryIndex int
}

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

	segments := make(map[int]struct{})
	for _, change := range stream.SegmentChanges() {
		segments[change.EntryIndex] = struct{}{}
	}
	relocations := make(map[relocationLocation]struct{})
	for _, relocation := range stream.Relocations() {
		relocations[relocationLocation{entryIndex: relocation.EntryIndex, byteOffset: relocation.ByteOffset}] = struct{}{}
	}
	for index, entry := range stream.Entries() {
		switch node := entry.Node.(type) {
		case ast.Segment:
			if _, exists := segments[index]; !exists {
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

func recordDataRelocations(stream *ast.Stream, entryIndex int, data ast.Data, order ast.ByteOrder,
	recorded map[relocationLocation]struct{}) {

	if data.Type != ast.AddressType {
		return
	}

	width := data.Width
	if data.ReferenceType != ast.FullAddress {
		width = 1
	}
	for valueIndex, value := range data.Values {
		byteOffset := uint64(valueIndex * width)
		if _, exists := recorded[relocationLocation{entryIndex: entryIndex, byteOffset: byteOffset}]; exists {
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
