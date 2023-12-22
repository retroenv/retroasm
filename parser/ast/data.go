package ast

import (
	"github.com/retroenv/assembler/expression"
)

// ReferenceType defines the type of address reference.
type ReferenceType int

const (
	invalidReferenceType ReferenceType = iota
	FullAddress
	LowAddressByte
	HighAddressByte
)

// Data ...
type Data struct {
	node

	Type          string
	Width         int // byte width of a data item
	ReferenceType ReferenceType

	Fill bool

	Size   *expression.Expression
	Values *expression.Expression
}
