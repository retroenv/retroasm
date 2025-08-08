package ast

import (
	"github.com/retroenv/retroasm/expression"
)

// ReferenceType defines the type of address reference.
type ReferenceType int

const (
	// InvalidReferenceType represents an uninitialized or invalid reference type.
	InvalidReferenceType ReferenceType = iota
	// FullAddress represents a full address reference (both high and low bytes).
	FullAddress
	// LowAddressByte represents only the low byte of an address.
	LowAddressByte
	// HighAddressByte represents only the high byte of an address.
	HighAddressByte
)

// DataContentType defines the type of the data node.
type DataContentType int

const (
	InvalidDataType DataContentType = iota
	AddressType
	DataType
)

// Data ...
type Data struct {
	*node

	Type          DataContentType
	Width         int // byte width of a data item
	ReferenceType ReferenceType

	Fill bool

	Size   *expression.Expression
	Values *expression.Expression
}

// NewData returns a new data node.
func NewData(typ DataContentType, width int) Data {
	return Data{
		node:  &node{},
		Type:  typ,
		Width: width,
		Size:  expression.New(),
	}
}

// Copy returns a copy of the data node.
func (d Data) Copy() Node {
	var sizeCopy, valuesCopy *expression.Expression
	if d.Size != nil {
		sizeCopy = d.Size.Copy()
	}
	if d.Values != nil {
		valuesCopy = d.Values.Copy()
	}

	return Data{
		node:          d.node,
		Type:          d.Type,
		Width:         d.Width,
		ReferenceType: d.ReferenceType,
		Fill:          d.Fill,
		Size:          sizeCopy,
		Values:        valuesCopy,
	}
}
