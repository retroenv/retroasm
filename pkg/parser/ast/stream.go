package ast

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"

	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/number"
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

// PackedField describes a relocation that occupies part of its encoded byte span.
// Bit offsets and masks use the unsigned value after byte-order decoding.
type PackedField struct {
	BitOffset    uint8
	BitWidth     uint8
	PreserveMask uint64
}

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

// SymbolKind identifies how a stream node defines a symbol.
type SymbolKind uint8

const (
	SymbolInvalid SymbolKind = iota
	AliasSymbol
	EquSymbol
	LabelSymbol
	FunctionSymbol
)

// SymbolExpressionKind identifies the base value of a symbol expression.
type SymbolExpressionKind uint8

const (
	SymbolExpressionInvalid SymbolExpressionKind = iota
	SymbolExpressionAbsolute
	SymbolExpressionReference
	SymbolExpressionDefinition
	SymbolExpressionLocation
)

// SourcePosition identifies one location in an input source.
type SourcePosition struct {
	Source string
	Line   int
	Column int
	Offset int
}

// SymbolExpression is a source definition, stream location, absolute value, or symbol reference.
type SymbolExpression struct {
	Kind          SymbolExpressionKind
	Value         uint64
	Symbol        string
	Addend        int64
	ReferenceType ReferenceType
	Definition    *expression.Expression
}

// Symbol is a typed symbol definition in a stream.
type Symbol struct {
	EntryIndex int
	Kind       SymbolKind
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
	Field      PackedField
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

// NewDefinitionSymbolExpression returns an owned source expression.
func NewDefinitionSymbolExpression(definition *expression.Expression) SymbolExpression {
	var copied *expression.Expression
	if definition != nil {
		copied = definition.Copy()
	}
	return SymbolExpression{
		Kind:          SymbolExpressionDefinition,
		ReferenceType: FullAddress,
		Definition:    copied,
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

// NewLocationSymbolExpression returns an unresolved stream location expression.
func NewLocationSymbolExpression() SymbolExpression {
	return SymbolExpression{
		Kind:          SymbolExpressionLocation,
		ReferenceType: FullAddress,
	}
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

// Copy returns an independent symbol expression.
func (exp SymbolExpression) Copy() SymbolExpression {
	if exp.Definition != nil {
		exp.Definition = exp.Definition.Copy()
	}
	return exp
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
		symbols:        copySymbols(stm.symbols),
		relocations:    copyRelocations(stm.relocations),
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
	relocation.Expression = relocation.Expression.Copy()
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
	symbol.Expression = symbol.Expression.Copy()
	stm.symbols = append(stm.symbols, symbol)
}

// Relocations returns a copy of the typed relocations.
func (stm *Stream) Relocations() []Relocation {
	return copyRelocations(stm.relocations)
}

// Replace atomically replaces a half-open range and reindexes compatible metadata.
func (stm *Stream) Replace(start, end int, replacement []Entry) error {
	if stm == nil {
		return fmt.Errorf("%w: stream is nil", ErrInvalidStream)
	}
	if start < 0 || end < start || end > len(stm.entries) {
		return fmt.Errorf("%w: replacement range %d:%d is outside %d entries", ErrInvalidStream, start, end, len(stm.entries))
	}

	candidate := stm.Copy()
	entries := make([]Entry, 0, len(stm.entries)-(end-start)+len(replacement))
	entries = append(entries, stm.entries[:start]...)
	entries = append(entries, copyEntries(replacement)...)
	entries = append(entries, stm.entries[end:]...)
	candidate.entries = entries

	if err := candidate.reindexMetadata(start, end, len(replacement)); err != nil {
		return err
	}
	if err := candidate.Validate(); err != nil {
		return fmt.Errorf("%w: replacement metadata is incompatible: %w", ErrInvalidStream, err)
	}

	*stm = *candidate
	return nil
}

// ResolveSymbolValues atomically records resolved label and function addresses.
func (stm *Stream) ResolveSymbolValues(values map[string]uint64) error {
	if stm == nil {
		return fmt.Errorf("%w: stream is nil", ErrInvalidStream)
	}

	candidate := stm.Copy()
	for index := range candidate.symbols {
		symbol := &candidate.symbols[index]
		if symbol.Kind != LabelSymbol && symbol.Kind != FunctionSymbol {
			continue
		}
		value, ok := values[symbol.Name]
		if ok {
			symbol.Expression = NewAbsoluteSymbolExpression(value)
		}
	}
	if err := candidate.Validate(); err != nil {
		return fmt.Errorf("%w: resolved symbol metadata is incompatible: %w", ErrInvalidStream, err)
	}

	*stm = *candidate
	return nil
}

// SegmentChanges returns a copy of the typed segment changes.
func (stm *Stream) SegmentChanges() []SegmentChange {
	return slices.Clone(stm.segmentChanges)
}

// Symbols returns a copy of the typed symbol definitions.
func (stm *Stream) Symbols() []Symbol {
	return copySymbols(stm.symbols)
}

// Validate checks stream entries and assembly metadata for structural consistency.
func (stm *Stream) Validate() error {
	if stm == nil {
		return fmt.Errorf("%w: stream is nil", ErrInvalidStream)
	}
	if err := validateEntries(stm.entries); err != nil {
		return err
	}
	if err := validateSymbols(stm.symbols, stm.entries); err != nil {
		return err
	}
	if err := validateRelocations(stm.relocations, stm.entries); err != nil {
		return err
	}
	return validateSegmentChanges(stm.segmentChanges, stm.entries)
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

func validateSymbols(symbols []Symbol, entries []Entry) error {
	for index, symbol := range symbols {
		if symbol.EntryIndex < 0 || symbol.EntryIndex >= len(entries) {
			return fmt.Errorf("%w: symbol %d has entry index %d", ErrInvalidStream, index, symbol.EntryIndex)
		}
		if symbol.Kind < AliasSymbol || symbol.Kind > FunctionSymbol {
			return fmt.Errorf("%w: symbol %d has kind %d", ErrInvalidStream, index, symbol.Kind)
		}
		if symbol.Name == "" {
			return fmt.Errorf("%w: symbol %d has no name", ErrInvalidStream, index)
		}
		if !validSourcePosition(symbol.Position) {
			return fmt.Errorf("%w: symbol %q has a negative source position", ErrInvalidStream, symbol.Name)
		}
		if err := validateSymbolExpression(symbol.Expression); err != nil {
			return fmt.Errorf("%w: symbol %q has %w", ErrInvalidStream, symbol.Name, err)
		}
		if err := validateSymbolNode(symbol, entries[symbol.EntryIndex]); err != nil {
			return fmt.Errorf("%w: symbol %q has %w", ErrInvalidStream, symbol.Name, err)
		}
	}
	return nil
}

func validateRelocations(relocations []Relocation, entries []Entry) error {
	for index, relocation := range relocations {
		if relocation.EntryIndex < 0 || relocation.EntryIndex >= len(entries) {
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
		if err := validatePackedField(relocation.Field, relocation.Width); err != nil {
			return fmt.Errorf("%w: relocation %d has %w", ErrInvalidStream, index, err)
		}
		if relocation.Expression.Kind != SymbolExpressionReference {
			return fmt.Errorf("%w: relocation %d does not reference a symbol", ErrInvalidStream, index)
		}
		if err := validateRelocationNode(relocation, entries[relocation.EntryIndex].Node); err != nil {
			return fmt.Errorf("%w: relocation %d has %w", ErrInvalidStream, index, err)
		}
	}
	return nil
}

func validateSegmentChanges(changes []SegmentChange, entries []Entry) error {
	for index, change := range changes {
		if change.EntryIndex < 0 || change.EntryIndex >= len(entries) {
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
		segment, ok := entries[change.EntryIndex].Node.(Segment)
		if !ok || segment.Name != change.Name {
			return fmt.Errorf("%w: segment change %d does not match its entry", ErrInvalidStream, index)
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
		return validateAbsoluteSymbolExpression(expression)
	case SymbolExpressionReference:
		return validateReferenceSymbolExpression(expression)
	case SymbolExpressionDefinition:
		return validateDefinitionSymbolExpression(expression)
	case SymbolExpressionLocation:
		return validateLocationSymbolExpression(expression)
	default:
		return fmt.Errorf("expression kind %d", expression.Kind)
	}
}

func validateAbsoluteSymbolExpression(expression SymbolExpression) error {
	if expression.Symbol != "" || expression.Addend != 0 || expression.ReferenceType != FullAddress || expression.Definition != nil {
		return errors.New("an absolute expression with reference data")
	}
	return nil
}

func validateReferenceSymbolExpression(expression SymbolExpression) error {
	if expression.Symbol == "" || expression.Value != 0 || expression.Definition != nil {
		return errors.New("a reference expression with invalid symbol data")
	}
	return nil
}

func validateDefinitionSymbolExpression(expression SymbolExpression) error {
	if expression.Definition == nil || len(expression.Definition.Tokens()) == 0 {
		return errors.New("a definition expression without tokens")
	}
	if expression.Symbol != "" || expression.Value != 0 || expression.Addend != 0 || expression.ReferenceType != FullAddress {
		return errors.New("a definition expression with a resolved value")
	}
	return nil
}

func validateLocationSymbolExpression(expression SymbolExpression) error {
	if expression.Symbol != "" || expression.Value != 0 || expression.Addend != 0 || expression.ReferenceType != FullAddress || expression.Definition != nil {
		return errors.New("an unresolved location expression with a value")
	}
	return nil
}

func (stm *Stream) reindexMetadata(start, end, replacementCount int) error {
	for index := range stm.symbols {
		entryIndex, err := replacementEntryIndex(stm.symbols[index].EntryIndex, start, end, replacementCount)
		if err != nil {
			return fmt.Errorf("%w: symbol %q: %w", ErrInvalidStream, stm.symbols[index].Name, err)
		}
		stm.symbols[index].EntryIndex = entryIndex
	}
	for index := range stm.relocations {
		entryIndex, err := replacementEntryIndex(stm.relocations[index].EntryIndex, start, end, replacementCount)
		if err != nil {
			return fmt.Errorf("%w: relocation %d: %w", ErrInvalidStream, index, err)
		}
		stm.relocations[index].EntryIndex = entryIndex
	}
	for index := range stm.segmentChanges {
		entryIndex, err := replacementEntryIndex(stm.segmentChanges[index].EntryIndex, start, end, replacementCount)
		if err != nil {
			return fmt.Errorf("%w: segment change %q: %w", ErrInvalidStream, stm.segmentChanges[index].Name, err)
		}
		stm.segmentChanges[index].EntryIndex = entryIndex
	}
	return nil
}

func replacementEntryIndex(entryIndex, start, end, replacementCount int) (int, error) {
	if entryIndex < start {
		return entryIndex, nil
	}
	if entryIndex >= end {
		return entryIndex + replacementCount - (end - start), nil
	}
	if replacementCount != end-start {
		return 0, errors.New("replacement removes its metadata entry")
	}
	return entryIndex, nil
}

func validateSymbolNode(symbol Symbol, entry Entry) error {
	if symbol.Position != entry.Position {
		return errors.New("a source position that differs from its entry")
	}

	switch symbol.Kind {
	case AliasSymbol, EquSymbol:
		alias, ok := entry.Node.(Alias)
		expectedReusable := symbol.Kind == AliasSymbol
		if !ok || alias.Name != symbol.Name || alias.SymbolReusable != expectedReusable {
			return errors.New("a definition that differs from its entry")
		}
		if symbol.Expression.Kind != SymbolExpressionDefinition || !sameDefinition(symbol.Expression.Definition, alias.Expression) {
			return errors.New("an expression that differs from its entry")
		}
	case LabelSymbol:
		label, ok := entry.Node.(Label)
		if !ok || label.Name != symbol.Name || !locationExpression(symbol.Expression) {
			return errors.New("a label that differs from its entry")
		}
	case FunctionSymbol:
		function, ok := entry.Node.(Function)
		if !ok || function.Name != symbol.Name || !locationExpression(symbol.Expression) {
			return errors.New("a function that differs from its entry")
		}
	}
	return nil
}

func sameDefinition(left, right *expression.Expression) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.IsEvaluatedOnce() == right.IsEvaluatedOnce() && slices.Equal(left.Tokens(), right.Tokens())
}

func locationExpression(expression SymbolExpression) bool {
	return expression.Kind == SymbolExpressionLocation || expression.Kind == SymbolExpressionAbsolute
}

func validateRelocationNode(relocation Relocation, node Node) error {
	if instruction, ok := InstructionFromNode(node); ok {
		if !instructionReferencesExpression(instruction, relocation.Expression) {
			return errors.New("an instruction operand that differs from its expression")
		}
		return nil
	}

	data, ok := DataFromNode(node)
	if !ok || data.Type != AddressType || relocation.Kind != AbsoluteRelocation {
		return errors.New("an entry that cannot own a relocation")
	}
	if relocation.Field != (PackedField{}) {
		return errors.New("a packed field on a data entry")
	}
	width := data.Width
	if data.ReferenceType != FullAddress {
		width = 1
	}
	if DataWidth(width) != relocation.Width || relocation.ByteOffset%uint64(width) != 0 {
		return errors.New("a width or byte offset that differs from its data entry")
	}
	valueIndex := int(relocation.ByteOffset / uint64(width))
	if valueIndex >= len(data.Values) {
		return errors.New("a byte offset outside its data entry")
	}
	symbol, addend, ok := ParseSymbolReference(data.Values[valueIndex])
	if !ok || symbol != relocation.Expression.Symbol || addend != relocation.Expression.Addend ||
		data.ReferenceType != relocation.Expression.ReferenceType {

		return errors.New("a data value that differs from its symbol")
	}
	return nil
}

func validatePackedField(field PackedField, width DataWidth) error {
	if field == (PackedField{}) {
		return nil
	}
	if field.BitWidth == 0 {
		return errors.New("a packed field without a bit width")
	}

	encodedBits := uint16(width) * 8
	if encodedBits > 64 {
		return errors.New("a packed field in more than 64 encoded bits")
	}
	fieldEnd := uint16(field.BitOffset) + uint16(field.BitWidth)
	if fieldEnd > encodedBits {
		return errors.New("a packed field outside its encoded width")
	}

	fieldMask := lowBitMask(field.BitWidth) << field.BitOffset
	encodedMask := lowBitMask(uint8(encodedBits))
	expectedPreserveMask := encodedMask &^ fieldMask
	if field.PreserveMask != expectedPreserveMask {
		return errors.New("a packed field with an inconsistent preserve mask")
	}
	return nil
}

func lowBitMask(bits uint8) uint64 {
	if bits == 64 {
		return ^uint64(0)
	}
	return (uint64(1) << bits) - 1
}

func instructionReferencesExpression(instruction Instruction, expected SymbolExpression) bool {
	if len(instruction.Modifier) == 0 {
		return nodeReferencesExpression(instruction.Argument, expected)
	}

	return instructionReferenceMatchesExpression(InstructionReference{
		Value:         instruction.Argument,
		Modifiers:     instruction.Modifier,
		ReferenceType: FullAddress,
	}, expected)
}

func nodeReferencesExpression(node Node, expected SymbolExpression) bool {
	if symbol, addend, ok := nodeSymbolReference(node); ok {
		return symbol == expected.Symbol && addend == expected.Addend && expected.ReferenceType == FullAddress
	}

	switch argument := node.(type) {
	case InstructionArgument:
		if provider, ok := argument.Value.(InstructionReferenceProvider); ok {
			for _, reference := range provider.InstructionReferences() {
				if instructionReferenceMatchesExpression(reference, expected) {
					return true
				}
			}
		}
		nested, ok := argument.Value.(Node)
		return ok && nodeReferencesExpression(nested, expected)
	case InstructionArguments:
		for _, value := range argument.Values {
			if nodeReferencesExpression(value, expected) {
				return true
			}
		}
	}
	return false
}

func instructionReferenceMatchesExpression(reference InstructionReference, expected SymbolExpression) bool {
	symbol, addend, ok := nodeSymbolReference(reference.Value)
	if !ok {
		return false
	}
	modifierAddend, ok := instructionModifierAddend(reference.Modifiers)
	if !ok {
		return false
	}
	addend, ok = combineInstructionAddends(addend, modifierAddend)
	return ok && symbol == expected.Symbol && addend == expected.Addend && reference.ReferenceType == expected.ReferenceType
}

func nodeSymbolReference(node Node) (string, int64, bool) {
	if symbol := SymbolName(node); symbol != "" {
		return symbol, 0, true
	}
	if expression, ok := node.(Expression); ok {
		return ParseSymbolReference(expression.Value)
	}
	return "", 0, false
}

func instructionModifierAddend(modifiers []Modifier) (int64, bool) {
	var result int64

	for _, modifier := range modifiers {
		value, err := number.Parse(modifier.Value)
		if err != nil || value > math.MaxInt64 {
			return 0, false
		}
		delta := int64(value)
		switch modifier.Operator.Operator {
		case "+":
		case "-":
			delta = -delta
		default:
			return 0, false
		}
		var ok bool
		result, ok = combineInstructionAddends(result, delta)
		if !ok {
			return 0, false
		}
	}
	return result, true
}

func combineInstructionAddends(left, right int64) (int64, bool) {
	if right > 0 && left > math.MaxInt64-right || right < 0 && left < math.MinInt64-right {
		return 0, false
	}
	return left + right, true
}

func copySymbols(symbols []Symbol) []Symbol {
	result := slices.Clone(symbols)
	for index := range result {
		result[index].Expression = result[index].Expression.Copy()
	}
	return result
}

func copyRelocations(relocations []Relocation) []Relocation {
	result := slices.Clone(relocations)
	for index := range result {
		result[index].Expression = result[index].Expression.Copy()
	}
	return result
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
