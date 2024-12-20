package ast

import (
	"github.com/retroenv/retroasm/expression"
)

// ReferenceType defines the type of address reference.
type ReferenceType int

const (
	invalidReferenceType ReferenceType = iota
	FullAddress
	LowAddressByte
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
	return Data{
		node:          d.node,
		Type:          d.Type,
		Width:         d.Width,
		ReferenceType: d.ReferenceType,
		Fill:          d.Fill,
		Size:          d.Size.Copy(),
		Values:        d.Values.Copy(),
	}
}
