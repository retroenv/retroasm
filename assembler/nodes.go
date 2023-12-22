package assembler

import (
	"github.com/retroenv/assembler/expression"
	"github.com/retroenv/assembler/lexer/token"
	"github.com/retroenv/assembler/parser/ast"
	"github.com/retroenv/assembler/scope"
	"github.com/retroenv/retrogolib/addressing"
)

// referenceType defines the type of reference.
type referenceType int

const (
	invalidReferenceType referenceType = iota
	fullAddress
	lowAddressByte
	highAddressByte
)

// reference for a label or constant.
type reference struct {
	name string
	typ  referenceType
}

// data of type []byte or string.
type data struct {
	address uint64 // assigned start address of the data
	width   int    // data item width in bytes
	// flag whether data space is reserved and should be filled with the
	// optional fill bytes in values. If the fill values are shorter than
	// the reserved space, the fill values will be repeated.
	fill bool

	size       *expression.Expression // item count
	expression *expression.Expression
	// values will be filled by evaluating the expression.
	// each value can be of type []byte or reference.
	// since expressions are evaluated before addresses are assigned,
	// the references will be replaced by the resolved addresses at the
	// opcode generation step.
	values []any
}

// instruction of the used architecture.
type instruction struct {
	address uint64 // assigned start address of the instruction
	size    int
	opcodes []byte

	name       string
	addressing addressing.Mode
	argument   any
}

type base struct {
	address *expression.Expression
}

type variable struct {
	address uint64 // assigned start address of the instruction

	v *ast.Variable
}

type scopeChange struct {
	scope *scope.Scope
}

type macro struct {
	name      string
	arguments map[string]int // maps name to position
	tokens    []token.Token
}
