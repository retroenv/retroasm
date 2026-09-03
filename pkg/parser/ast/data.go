package ast

import (
	"errors"
	"fmt"

	"github.com/retroenv/retroasm/pkg/expression"
)

// ErrInvalidData indicates that a data node violates the typed data contract.
var ErrInvalidData = errors.New("invalid typed data")

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
	// BankAddressByte represents the bank byte (bits 16-23) of an address.
	BankAddressByte
)

// DataContentType defines the type of the data node.
type DataContentType int

const (
	InvalidDataType DataContentType = iota
	AddressType
	DataType
)

// Data represents a data definition directive (.byte, .word, .db, .dw).
type Data struct {
	*node

	Type          DataContentType
	Width         int // byte width of a data item
	ReferenceType ReferenceType

	Fill bool

	Size   *expression.Expression
	Values []*expression.Expression
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

// Validate checks the typed data shape before formatting or assembly.
func (d Data) Validate() error {
	if d.Type != AddressType && d.Type != DataType {
		return fmt.Errorf("%w: content type %d", ErrInvalidData, d.Type)
	}
	if d.Width <= 0 {
		return fmt.Errorf("%w: width %d", ErrInvalidData, d.Width)
	}
	if d.Size == nil {
		return fmt.Errorf("%w: size expression is nil", ErrInvalidData)
	}
	if d.Fill && len(d.Size.Tokens()) == 0 {
		return fmt.Errorf("%w: fill size expression is empty", ErrInvalidData)
	}
	if !d.Fill && len(d.Values) == 0 {
		return fmt.Errorf("%w: data values are empty", ErrInvalidData)
	}

	if err := d.validateContentType(); err != nil {
		return err
	}
	for index, value := range d.Values {
		if value == nil {
			return fmt.Errorf("%w: value %d is nil", ErrInvalidData, index)
		}
		tokenCount := len(value.Tokens())
		if tokenCount == 0 {
			return fmt.Errorf("%w: value %d is empty", ErrInvalidData, index)
		}
		if d.Type == AddressType && tokenCount != 1 {
			return fmt.Errorf("%w: address value %d has %d tokens", ErrInvalidData, index, tokenCount)
		}
	}
	return nil
}

// Copy returns a copy of the data node.
func (d Data) Copy() Node {
	var sizeCopy *expression.Expression
	if d.Size != nil {
		sizeCopy = d.Size.Copy()
	}

	return Data{
		node:          d.node.copyNode(),
		Type:          d.Type,
		Width:         d.Width,
		ReferenceType: d.ReferenceType,
		Fill:          d.Fill,
		Size:          sizeCopy,
		Values:        copyDataValues(d.Values),
	}
}

func (d Data) validateContentType() error {
	if d.Type == DataType {
		if d.ReferenceType != InvalidReferenceType {
			return fmt.Errorf("%w: plain data has reference type %d", ErrInvalidData, d.ReferenceType)
		}
		return nil
	}
	if d.ReferenceType < FullAddress || d.ReferenceType > BankAddressByte {
		return fmt.Errorf("%w: address reference type %d", ErrInvalidData, d.ReferenceType)
	}
	if d.Fill {
		return fmt.Errorf("%w: address data cannot reserve storage", ErrInvalidData)
	}
	return nil
}

// DataFromNode returns the data stored in node.
func DataFromNode(n Node) (Data, bool) {
	switch data := n.(type) {
	case Data:
		return data, true
	case *Data:
		if data != nil {
			return *data, true
		}
	}
	return Data{}, false
}

func copyDataValues(values []*expression.Expression) []*expression.Expression {
	if values == nil {
		return nil
	}

	copied := make([]*expression.Expression, len(values))
	for index, value := range values {
		if value != nil {
			copied[index] = value.Copy()
		}
	}
	return copied
}
