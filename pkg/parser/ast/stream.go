package ast

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
)

// ErrInvalidStream indicates that stream nodes or metadata violate the typed stream contract.
var ErrInvalidStream = errors.New("invalid typed stream")

// Alignment is a byte alignment. Zero means that no alignment is specified.
type Alignment uint64

// Annotation is typed metadata that belongs to one stream entry.
type Annotation interface {
	CopyStreamAnnotation() Annotation
}

// StateCopier provides independent copies for mutable target state.
type StateCopier interface {
	CopyStreamState() any
}

// ByteOrder identifies the byte order for data and relocations.
type ByteOrder uint8

const (
	ByteOrderUnknown ByteOrder = iota
	ByteOrderLittle
	ByteOrderBig
)

// DataWidth is the encoded width of one data item in bytes.
type DataWidth uint8

const (
	WidthUnknown    DataWidth = 0
	WidthByte       DataWidth = 1
	WidthWord       DataWidth = 2
	WidthLong       DataWidth = 3
	WidthDoubleWord DataWidth = 4
	WidthQuadWord   DataWidth = 8
)

// OptimizationBoundary identifies which sides of an entry block optimization.
type OptimizationBoundary uint8

const BoundaryNone OptimizationBoundary = 0

const (
	BoundaryBefore OptimizationBoundary = 1 << iota
	BoundaryAfter
)

// RelocationKind identifies how an assembler resolves a symbol reference.
type RelocationKind uint8

const (
	RelocationInvalid RelocationKind = iota
	AbsoluteRelocation
	RelativeRelocation
)

// SymbolExpressionKind identifies the base value of a symbol expression.
type SymbolExpressionKind uint8

const (
	SymbolExpressionInvalid SymbolExpressionKind = iota
	SymbolExpressionAbsolute
	SymbolExpressionReference
)

// SourcePosition identifies one location in an input source.
type SourcePosition struct {
	Source string
	Line   int
	Column int
	Offset int
}

// SymbolExpression is an absolute value or a symbol reference with an addend.
type SymbolExpression struct {
	Kind          SymbolExpressionKind
	Value         uint64
	Symbol        string
	Addend        int64
	ReferenceType ReferenceType
}

// Symbol is a typed symbol definition in a stream.
type Symbol struct {
	Name       string
	Segment    string
	Expression SymbolExpression
	Position   SourcePosition
}

// Relocation identifies a value that assembly must resolve in encoded data.
type Relocation struct {
	EntryIndex int
	ByteOffset uint64
	Kind       RelocationKind
	Expression SymbolExpression
	Width      DataWidth
	ByteOrder  ByteOrder
}

// SegmentChange records segment state at one stream entry.
type SegmentChange struct {
	EntryIndex int
	Name       string
	Alignment  Alignment
	ByteOrder  ByteOrder
}

// Entry stores one node and its lossless stream metadata.
type Entry struct {
	Node        Node
	Position    SourcePosition
	Annotations []Annotation
	Boundary    OptimizationBoundary
}

// Stream owns ordered AST entries and their assembly metadata.
type Stream struct {
	entries        []Entry
	initialState   any
	finalState     any
	symbols        []Symbol
	relocations    []Relocation
	segmentChanges []SegmentChange
}

// NewAbsoluteSymbolExpression returns an expression with a fixed value.
func NewAbsoluteSymbolExpression(value uint64) SymbolExpression {
	return SymbolExpression{
		Kind:          SymbolExpressionAbsolute,
		Value:         value,
		ReferenceType: FullAddress,
	}
}

// NewEntry returns a stream entry for node and position.
func NewEntry(node Node, position SourcePosition) Entry {
	return Entry{Node: node, Position: position}
}

// NewStream returns a stream that owns copies of entries.
func NewStream(entries ...Entry) *Stream {
	return &Stream{entries: copyEntries(entries)}
}

// NewStreamFromNodes returns a stream for nodes without source metadata.
func NewStreamFromNodes(nodes ...Node) *Stream {
	entries := make([]Entry, len(nodes))

	for index, node := range nodes {
		entries[index] = Entry{Node: node}
	}
	return NewStream(entries...)
}

// NewSymbolExpression returns an expression for a symbol reference.
func NewSymbolExpression(symbol string, addend int64, referenceType ReferenceType) SymbolExpression {
	return SymbolExpression{
		Kind:          SymbolExpressionReference,
		Symbol:        symbol,
		Addend:        addend,
		ReferenceType: referenceType,
	}
}

// BlocksAfter reports whether optimization cannot cross the entry after it.
func (bnd OptimizationBoundary) BlocksAfter() bool {
	return bnd&BoundaryAfter != 0
}

// BlocksBefore reports whether optimization cannot cross the entry before it.
func (bnd OptimizationBoundary) BlocksBefore() bool {
	return bnd&BoundaryBefore != 0
}

// Valid reports whether alignment is zero or a power of two.
func (aln Alignment) Valid() bool {
	return aln == 0 || aln&(aln-1) == 0
}

// Valid reports whether width identifies at least one byte.
func (wid DataWidth) Valid() bool {
	return wid > 0
}

// Copy returns an independent entry.
func (ent Entry) Copy() Entry {
	result := Entry{
		Position: ent.Position,
		Boundary: ent.Boundary,
	}
	if ent.Node != nil {
		result.Node = ent.Node.Copy()
	}
	if ent.Annotations != nil {
		result.Annotations = make([]Annotation, len(ent.Annotations))
		for index, annotation := range ent.Annotations {
			if annotation != nil {
				result.Annotations[index] = annotation.CopyStreamAnnotation()
			}
		}
	}
	return result
}

// Append appends copies of entries to the stream.
func (stm *Stream) Append(entries ...Entry) {
	stm.entries = append(stm.entries, copyEntries(entries)...)
}

// At returns an independent copy of one entry.
func (stm *Stream) At(index int) Entry {
	return stm.entries[index].Copy()
}

// Copy returns an independent stream.
func (stm *Stream) Copy() *Stream {
	if stm == nil {
		return nil
	}
	return &Stream{
		entries:        copyEntries(stm.entries),
		initialState:   copyStreamState(stm.initialState),
		finalState:     copyStreamState(stm.finalState),
		symbols:        slices.Clone(stm.symbols),
		relocations:    slices.Clone(stm.relocations),
		segmentChanges: slices.Clone(stm.segmentChanges),
	}
}

// Entries returns independent copies of all entries.
func (stm *Stream) Entries() []Entry {
	return copyEntries(stm.entries)
}

// Len returns the number of stream entries.
func (stm *Stream) Len() int {
	return len(stm.entries)
}

// Nodes returns independent copies of all stream nodes.
func (stm *Stream) Nodes() []Node {
	nodes := make([]Node, len(stm.entries))

	for index, entry := range stm.entries {
		if entry.Node != nil {
			nodes[index] = entry.Node.Copy()
		}
	}
	return nodes
}

// RecordRelocation appends a typed relocation to the stream.
func (stm *Stream) RecordRelocation(relocation Relocation) {
	stm.relocations = append(stm.relocations, relocation)
}

// RecordSegmentChange appends a typed segment change to the stream.
func (stm *Stream) RecordSegmentChange(change SegmentChange) {
	stm.segmentChanges = append(stm.segmentChanges, change)
}

// RecordState stores independent initial and final target-state snapshots.
func (stm *Stream) RecordState(initial, final any) {
	stm.initialState = copyStreamState(initial)
	stm.finalState = copyStreamState(final)
}

// RecordSymbol appends a typed symbol definition to the stream.
func (stm *Stream) RecordSymbol(symbol Symbol) {
	stm.symbols = append(stm.symbols, symbol)
}

// Relocations returns a copy of the typed relocations.
func (stm *Stream) Relocations() []Relocation {
	return slices.Clone(stm.relocations)
}

// Replace replaces a half-open range with copies of replacement entries.
func (stm *Stream) Replace(start, end int, replacement []Entry) {
	entries := make([]Entry, 0, len(stm.entries)-(end-start)+len(replacement))
	entries = append(entries, stm.entries[:start]...)
	entries = append(entries, copyEntries(replacement)...)
	entries = append(entries, stm.entries[end:]...)
	stm.entries = entries
}

// SegmentChanges returns a copy of the typed segment changes.
func (stm *Stream) SegmentChanges() []SegmentChange {
	return slices.Clone(stm.segmentChanges)
}

// Symbols returns a copy of the typed symbol definitions.
func (stm *Stream) Symbols() []Symbol {
	return slices.Clone(stm.symbols)
}

// Validate checks stream entries and assembly metadata for structural consistency.
func (stm *Stream) Validate() error {
	if stm == nil {
		return fmt.Errorf("%w: stream is nil", ErrInvalidStream)
	}
	if err := validateEntries(stm.entries); err != nil {
		return err
	}
	if err := validateSymbols(stm.symbols); err != nil {
		return err
	}
	if err := validateRelocations(stm.relocations, len(stm.entries)); err != nil {
		return err
	}
	return validateSegmentChanges(stm.segmentChanges, len(stm.entries))
}

// StateSnapshots returns typed copies of the initial and final stream state.
func StateSnapshots[S any](stream *Stream) (S, S, bool) {
	var zero S
	if stream == nil {
		return zero, zero, false
	}
	initial, initialOK := stream.initialState.(S)
	final, finalOK := stream.finalState.(S)
	if !initialOK || !finalOK {
		return zero, zero, false
	}
	return copyStreamState(initial).(S), copyStreamState(final).(S), true
}

func validateEntries(entries []Entry) error {
	for index, entry := range entries {
		if entry.Node == nil {
			return fmt.Errorf("%w: entry %d has no node", ErrInvalidStream, index)
		}
		if !validSourcePosition(entry.Position) {
			return fmt.Errorf("%w: entry %d has a negative source position", ErrInvalidStream, index)
		}
		if entry.Boundary&^(BoundaryBefore|BoundaryAfter) != 0 {
			return fmt.Errorf("%w: entry %d has boundary %d", ErrInvalidStream, index, entry.Boundary)
		}
	}
	return nil
}

func validateSymbols(symbols []Symbol) error {
	names := make(map[string]struct{}, len(symbols))

	for index, symbol := range symbols {
		if symbol.Name == "" {
			return fmt.Errorf("%w: symbol %d has no name", ErrInvalidStream, index)
		}
		if _, exists := names[symbol.Name]; exists {
			return fmt.Errorf("%w: symbol %q is duplicated", ErrInvalidStream, symbol.Name)
		}
		if !validSourcePosition(symbol.Position) {
			return fmt.Errorf("%w: symbol %q has a negative source position", ErrInvalidStream, symbol.Name)
		}
		if err := validateSymbolExpression(symbol.Expression); err != nil {
			return fmt.Errorf("%w: symbol %q has %w", ErrInvalidStream, symbol.Name, err)
		}
		names[symbol.Name] = struct{}{}
	}
	return nil
}

func validateRelocations(relocations []Relocation, entryCount int) error {
	for index, relocation := range relocations {
		if relocation.EntryIndex < 0 || relocation.EntryIndex >= entryCount {
			return fmt.Errorf("%w: relocation %d has entry index %d", ErrInvalidStream, index, relocation.EntryIndex)
		}
		if relocation.Kind != AbsoluteRelocation && relocation.Kind != RelativeRelocation {
			return fmt.Errorf("%w: relocation %d has kind %d", ErrInvalidStream, index, relocation.Kind)
		}
		if err := validateSymbolExpression(relocation.Expression); err != nil {
			return fmt.Errorf("%w: relocation %d has %w", ErrInvalidStream, index, err)
		}
		if !relocation.Width.Valid() {
			return fmt.Errorf("%w: relocation %d has width %d", ErrInvalidStream, index, relocation.Width)
		}
		if !validByteOrder(relocation.ByteOrder) {
			return fmt.Errorf("%w: relocation %d has byte order %d", ErrInvalidStream, index, relocation.ByteOrder)
		}
	}
	return nil
}

func validateSegmentChanges(changes []SegmentChange, entryCount int) error {
	for index, change := range changes {
		if change.EntryIndex < 0 || change.EntryIndex >= entryCount {
			return fmt.Errorf("%w: segment change %d has entry index %d", ErrInvalidStream, index, change.EntryIndex)
		}
		if change.Name == "" {
			return fmt.Errorf("%w: segment change %d has no name", ErrInvalidStream, index)
		}
		if !change.Alignment.Valid() {
			return fmt.Errorf("%w: segment change %d has alignment %d", ErrInvalidStream, index, change.Alignment)
		}
		if !validByteOrder(change.ByteOrder) {
			return fmt.Errorf("%w: segment change %d has byte order %d", ErrInvalidStream, index, change.ByteOrder)
		}
	}
	return nil
}

func validateSymbolExpression(expression SymbolExpression) error {
	if expression.ReferenceType < FullAddress || expression.ReferenceType > BankAddressByte {
		return fmt.Errorf("reference type %d", expression.ReferenceType)
	}

	switch expression.Kind {
	case SymbolExpressionAbsolute:
		if expression.Symbol != "" {
			return errors.New("an absolute expression with a symbol")
		}
	case SymbolExpressionReference:
		if expression.Symbol == "" {
			return errors.New("a reference expression without a symbol")
		}
	default:
		return fmt.Errorf("expression kind %d", expression.Kind)
	}
	return nil
}

func validByteOrder(order ByteOrder) bool {
	return order >= ByteOrderLittle && order <= ByteOrderBig
}

func validSourcePosition(position SourcePosition) bool {
	return position.Line >= 0 && position.Column >= 0 && position.Offset >= 0
}

func copyEntries(entries []Entry) []Entry {
	if entries == nil {
		return nil
	}
	result := make([]Entry, len(entries))

	for index, entry := range entries {
		result[index] = entry.Copy()
	}
	return result
}

func copyStreamState(state any) any {
	if state == nil {
		return nil
	}
	if copier, ok := state.(StateCopier); ok {
		return copier.CopyStreamState()
	}
	if immutableStreamValue(reflect.ValueOf(state)) {
		return state
	}
	panic(fmt.Sprintf("ast: mutable stream state %T must implement CopyStreamState", state))
}

func immutableStreamValue(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}

	switch value.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.String:
		return true
	case reflect.Array:
		for index := range value.Len() {
			if !immutableStreamValue(value.Index(index)) {
				return false
			}
		}
		return true
	case reflect.Struct:
		for index := range value.NumField() {
			if !immutableStreamValue(value.Field(index)) {
				return false
			}
		}
		return true
	case reflect.Interface:
		return value.IsNil() || immutableStreamValue(value.Elem())
	default:
		return false
	}
}
