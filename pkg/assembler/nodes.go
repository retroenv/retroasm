package assembler

import (
	"maps"
	"slices"

	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retroasm/pkg/scope"
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

// Copy returns a copy of the data node.
func (d *data) Copy() ast.Node {
	return &data{
		address:    d.address,
		width:      d.width,
		fill:       d.fill,
		size:       d.size.Copy(),
		expression: d.expression.Copy(),
		values:     slices.Clone(d.values),
	}
}

func (d *data) SetComment(_ string) {
}

// instruction of the used architecture.
type instruction struct {
	address uint64 // assigned start address of the instruction
	size    int
	opcodes []byte

	name       string
	addressing int
	argument   any
}

func (i *instruction) Address() uint64 {
	return i.address
}

func (i *instruction) Opcodes() []byte {
	return i.opcodes
}

func (i *instruction) Size() int {
	return i.size
}

func (i *instruction) SetOpcodes(bytes []byte) {
	i.opcodes = bytes
}

func (i *instruction) SetSize(size int) {
	i.size = size
}

func (i *instruction) Addressing() int {
	return i.addressing
}

func (i *instruction) Argument() any {
	return i.argument
}

func (i *instruction) Name() string {
	return i.name
}

func (i *instruction) SetAddress(addr uint64) {
	i.address = addr
}

func (i *instruction) SetAddressing(addressing int) {
	i.addressing = addressing
}

// Copy returns a copy of the instruction node.
func (i *instruction) Copy() ast.Node {
	return &instruction{
		address:    i.address,
		size:       i.size,
		opcodes:    i.opcodes,
		name:       i.name,
		addressing: i.addressing,
		argument:   i.argument,
	}
}

func (i *instruction) SetComment(_ string) {
}

type variable struct {
	address uint64 // assigned start address of the instruction

	v ast.Variable
}

// Copy returns a copy of the variable node.
func (v *variable) Copy() ast.Node {
	return &variable{
		address: v.address,
		v:       v.v.Copy().(ast.Variable),
	}
}

func (v *variable) SetComment(_ string) {
}

type scopeChange struct {
	scope *scope.Scope
}

// Copy returns a copy of the scope change node.
func (s scopeChange) Copy() ast.Node {
	return scopeChange{
		scope: s.scope,
	}
}

func (s scopeChange) SetComment(_ string) {
}

type macro struct {
	name      string
	arguments map[string]int // maps name to position
	tokens    []token.Token
}

// Copy returns a copy of the macro node.
func (m macro) Copy() ast.Node {
	return macro{
		name:      m.name,
		arguments: maps.Clone(m.arguments),
		tokens:    slices.Clone(m.tokens),
	}
}

func (m macro) SetComment(_ string) {
}

// wrap symbol to implement ast.Node interface and avoid cyclic import.
type symbol struct {
	*scope.Symbol
}

// Copy returns a copy of the symbol node.
func (s *symbol) Copy() ast.Node {
	return &symbol{
		Symbol: s.Symbol.Copy(),
	}
}

func (s *symbol) SetComment(_ string) {
}
