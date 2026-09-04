package codec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/retroenv/retroasm/pkg/assembler"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

var (
	// ErrNilConfiguration indicates that a codec was created without target configuration.
	ErrNilConfiguration = errors.New("codec configuration cannot be nil")
	// ErrNilArchitecture indicates that a codec configuration has no architecture implementation.
	ErrNilArchitecture = errors.New("codec architecture cannot be nil")
	// ErrNilSource indicates that a parse operation received no source reader.
	ErrNilSource = errors.New("codec source cannot be nil")
	// ErrNilStream indicates that an operation received no typed stream.
	ErrNilStream = errors.New("codec stream cannot be nil")
	// ErrUnknownInstruction indicates that the target does not register a mnemonic.
	ErrUnknownInstruction = errors.New("unknown target instruction")
	// ErrExpectedInstruction indicates that a single-instruction parse produced another node shape.
	ErrExpectedInstruction = errors.New("expected exactly one instruction")
	// ErrBuildUnsupported indicates that an architecture has no typed builder for the requested operand type.
	ErrBuildUnsupported = errors.New("typed instruction building is not supported by architecture")
	// ErrValidationUnsupported indicates that an architecture has no typed instruction validator.
	ErrValidationUnsupported = errors.New("typed instruction validation is not supported by architecture")
	// ErrFormattingUnsupported indicates that a node has no deterministic spelling for the active configuration.
	ErrFormattingUnsupported = errors.New("typed stream formatting is not supported by configuration")
	// ErrStateType indicates that an architecture returned a different stream-state type.
	ErrStateType = errors.New("unexpected architecture stream state type")
	// ErrByteOrderUnsupported indicates that an architecture does not report its native byte order.
	ErrByteOrderUnsupported = errors.New("architecture byte order is not supported")
	// ErrInstructionRelocationMismatch indicates that retained metadata differs from the selected encoding.
	ErrInstructionRelocationMismatch = errors.New("instruction relocation does not match selected encoding")

	dataDirectiveNames = map[dataDirectiveKey]string{
		{fill: false, width: 1}: ".byte",
		{fill: false, width: 2}: ".word",
		{fill: true, width: 1}:  ".dsb",
		{fill: true, width: 2}:  ".dsw",
	}
	x816DataDirectiveNames = map[dataDirectiveKey]string{
		{fill: false, width: 3}: ".dcl",
		{fill: false, width: 4}: ".dcd",
		{fill: true, width: 3}:  ".dsl",
		{fill: true, width: 4}:  ".dsd",
	}
	configurationDirectiveNames = map[ast.ConfigurationItem]string{
		ast.ConfigMapper:      ".inesmap",
		ast.ConfigSubMapper:   ".inessubmap",
		ast.ConfigPrg:         ".inesprg",
		ast.ConfigChr:         ".ineschr",
		ast.ConfigBattery:     ".inesbat",
		ast.ConfigMirror:      ".inesmir",
		ast.ConfigNes2ChrRAM:  ".nes2chrram",
		ast.ConfigNes2PrgRAM:  ".nes2prgram",
		ast.ConfigNes2Sub:     ".nes2sub",
		ast.ConfigNes2TV:      ".nes2tv",
		ast.ConfigNes2VS:      ".nes2vs",
		ast.ConfigNes2BRam:    ".nes2bram",
		ast.ConfigNes2ChrBRam: ".nes2chrbram",
	}
)

// Assembly is the result of assembling a typed node stream.
type Assembly struct {
	Binary  []byte
	Symbols map[string]uint64
	Stream  *ast.Stream
}

// Codec binds typed stream operations to one architecture configuration.
type Codec[T any] struct {
	configuration *config.Config[T]
}

// New creates a typed stream codec for configuration.
func New[T any](configuration *config.Config[T]) (*Codec[T], error) {
	if configuration == nil {
		return nil, ErrNilConfiguration
	}
	if configuration.Arch == nil {
		return nil, ErrNilArchitecture
	}
	return &Codec[T]{configuration: configuration}, nil
}

// Parse reads an assembly stream into architecture-resolved AST nodes.
func (c *Codec[T]) Parse(ctx context.Context, source io.Reader) ([]ast.Node, error) {
	stream, err := c.ParseStream(ctx, "", source)
	if err != nil {
		return nil, err
	}
	return stream.Nodes(), nil
}

// ParseStream reads assembly into an owned typed stream.
func (c *Codec[T]) ParseStream(ctx context.Context, sourceName string, source io.Reader) (*ast.Stream, error) {
	return c.parseStream(ctx, sourceName, source, nil)
}

// ParseInstruction resolves a source stream containing exactly one instruction.
func (c *Codec[T]) ParseInstruction(ctx context.Context, source io.Reader) (ast.Instruction, error) {
	nodes, err := c.Parse(ctx, source)
	if err != nil {
		return ast.Instruction{}, err
	}
	return singleInstruction(nodes)
}

// OpcodeID resolves mnemonic to its architecture-scoped identity.
func (c *Codec[T]) OpcodeID(mnemonic string) (ast.OpcodeID, error) {
	lookupName := strings.ToLower(strings.TrimSpace(mnemonic))
	instruction, ok := c.configuration.Arch.Instruction(lookupName)
	if !ok {
		return ast.OpcodeID{}, fmt.Errorf("%w: %q", ErrUnknownInstruction, mnemonic)
	}
	identity := c.configuration.Arch.OpcodeID(instruction)
	if identity.Value == 0 {
		return ast.OpcodeID{}, fmt.Errorf("%w: %q has no opcode identity", ErrUnknownInstruction, mnemonic)
	}
	return identity, nil
}

// ValidateInstruction verifies architecture identity, profile, variant, addressing,
// register combination, and operands without assembling the instruction.
func (c *Codec[T]) ValidateInstruction(instruction ast.Instruction) error {
	validator, ok := c.configuration.Arch.(instructionValidator)
	if !ok {
		return ErrValidationUnsupported
	}
	if err := validator.ValidateInstruction(instruction); err != nil {
		return fmt.Errorf("validating typed instruction: %w", err)
	}
	return nil
}

// ValidateStream checks stream metadata, data nodes, and architecture-specific instructions.
func (c *Codec[T]) ValidateStream(stream *ast.Stream) error {
	if stream == nil {
		return ErrNilStream
	}
	if err := stream.Validate(); err != nil {
		return fmt.Errorf("validating typed stream: %w", err)
	}

	for index, entry := range stream.Entries() {
		if data, ok := ast.DataFromNode(entry.Node); ok {
			if err := data.Validate(); err != nil {
				return streamEntryError("validating", index, entry.Position, err)
			}
			continue
		}
		instruction, ok := ast.InstructionFromNode(entry.Node)
		if !ok {
			continue
		}
		if err := c.ValidateInstruction(instruction); err != nil {
			return streamEntryError("validating", index, entry.Position, err)
		}
	}
	return nil
}

// FormatInstruction returns one deterministic, parseable instruction line.
func (c *Codec[T]) FormatInstruction(instruction ast.Instruction) (string, error) {
	formatter, ok := c.configuration.Arch.(instructionFormatter)
	if !ok {
		return "", ErrFormattingUnsupported
	}
	formatted, err := formatter.FormatInstruction(instruction)
	if err != nil {
		return "", fmt.Errorf("formatting typed instruction: %w", err)
	}
	return formatted, nil
}

// FormatStream formats supported typed stream nodes as deterministic assembly source.
func (c *Codec[T]) FormatStream(stream *ast.Stream) (string, error) {
	if stream == nil {
		return "", ErrNilStream
	}
	if err := c.ValidateStream(stream); err != nil {
		return "", err
	}

	lines := make([]string, 0, stream.Len())
	for index, entry := range stream.Entries() {
		line, err := c.formatStreamNode(entry.Node)
		if err != nil {
			return "", streamEntryError("formatting", index, entry.Position, err)
		}
		if comment := ast.InlineComment(entry.Node); comment != "" {
			line += " ; " + comment
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n"), nil
}

// Assemble assembles a copy of nodes directly, without formatting or reparsing text.
func (c *Codec[T]) Assemble(ctx context.Context, nodes []ast.Node) (*Assembly, error) {
	return c.AssembleStream(ctx, ast.NewStreamFromNodes(nodes...))
}

// AssembleStream assembles a typed stream directly, without formatting or reparsing text.
func (c *Codec[T]) AssembleStream(ctx context.Context, stream *ast.Stream) (*Assembly, error) {
	if stream == nil {
		return nil, ErrNilStream
	}

	assemblyStream := stream.Copy()
	if err := c.recordAssemblyMetadata(assemblyStream); err != nil {
		return nil, fmt.Errorf("recording typed stream metadata: %w", err)
	}

	var output bytes.Buffer
	asm := assembler.New(c.configuration, &output)
	if err := asm.ProcessAST(ctx, assemblyStream.Nodes()); err != nil {
		return nil, fmt.Errorf("assembling typed stream: %w", err)
	}
	if err := reconcileInstructionRelocations(assemblyStream, asm.InstructionRelocations()); err != nil {
		return nil, fmt.Errorf("recording instruction relocations: %w", err)
	}
	if err := assemblyStream.ResolveSymbolValues(asm.Symbols()); err != nil {
		return nil, fmt.Errorf("resolving typed stream symbols: %w", err)
	}
	return &Assembly{
		Binary:  output.Bytes(),
		Symbols: maps.Clone(asm.Symbols()),
		Stream:  assemblyStream,
	}, nil
}

func (c *Codec[T]) parseStream(
	ctx context.Context,
	sourceName string,
	source io.Reader,
	initialState any,
) (*ast.Stream, error) {

	if source == nil {
		return nil, ErrNilSource
	}

	p := parser.New(c.configuration.Arch, source, c.configuration.CompatibilityMode)
	p.SetArchitectureState(initialState)
	if err := p.Read(ctx); err != nil {
		return nil, fmt.Errorf("reading assembly stream: %w", err)
	}
	stream, err := p.TokensToStream(sourceName)
	if err != nil {
		return nil, fmt.Errorf("resolving assembly stream: %w", err)
	}
	if err := c.recordAssemblyMetadata(stream); err != nil {
		return nil, fmt.Errorf("recording assembly stream metadata: %w", err)
	}
	return stream, nil
}

func (c *Codec[T]) formatStreamNode(node ast.Node) (string, error) {
	if instruction, ok := ast.InstructionFromNode(node); ok {
		return c.FormatInstruction(instruction)
	}
	if label, ok := ast.LabelName(node); ok {
		return label + ":", nil
	}
	if data, ok := ast.DataFromNode(node); ok {
		return c.formatData(data)
	}
	if comment, ok := node.(*ast.Comment); ok {
		if comment.Message == "" {
			return ";", nil
		}
		return "; " + comment.Message, nil
	}
	return c.formatLayoutNode(node)
}

func (c *Codec[T]) formatLayoutNode(node ast.Node) (string, error) {
	switch typed := node.(type) {
	case ast.Alias:
		return formatAlias(typed)
	case ast.Base:
		return formatBase(typed)
	case ast.Bank:
		return formatBank(typed)
	case ast.Segment:
		return formatSegment(typed)
	case ast.OffsetCounter:
		return ".rsset " + strconv.FormatUint(typed.Number, 10), nil
	case ast.Variable:
		return formatVariable(typed)
	case ast.Configuration:
		return c.formatConfiguration(typed)
	default:
		return c.formatStructuralNode(node)
	}
}

func (c *Codec[T]) formatConfiguration(configuration ast.Configuration) (string, error) {
	if configuration.Item == ast.ConfigFillValue {
		if configuration.Value != 0 {
			return "", fmt.Errorf("%w: fill configuration has numeric value %d", ErrFormattingUnsupported, configuration.Value)
		}
		value, err := ast.FormatExpression(configuration.Expression)
		if err != nil {
			return "", fmt.Errorf("%w: formatting fill configuration: %w", ErrFormattingUnsupported, err)
		}
		return ".fillvalue " + value, nil
	}
	if configuration.Expression != nil {
		return "", fmt.Errorf("%w: numeric configuration has an expression", ErrFormattingUnsupported)
	}

	directive, ok := configurationDirectiveNames[configuration.Item]
	if !ok {
		return "", fmt.Errorf("%w: configuration item %d", ErrFormattingUnsupported, configuration.Item)
	}
	if isAsm6Configuration(configuration.Item) &&
		c.configuration.CompatibilityMode != config.CompatAsm6 {

		return "", fmt.Errorf(
			"%w: configuration item %d in %s mode",
			ErrFormattingUnsupported,
			configuration.Item,
			c.configuration.CompatibilityMode,
		)
	}

	value, ok := configurationSourceValue(configuration.Item, configuration.Value)
	if !ok {
		return "", fmt.Errorf(
			"%w: configuration item %d value %d",
			ErrFormattingUnsupported,
			configuration.Item,
			configuration.Value,
		)
	}
	return directive + " " + strconv.FormatUint(value, 10), nil
}

func (c *Codec[T]) formatData(data ast.Data) (string, error) {
	directive, err := c.dataDirective(data)
	if err != nil {
		return "", err
	}

	values := make([]string, 0, len(data.Values)+1)
	if data.Fill {
		size, err := ast.FormatExpression(data.Size)
		if err != nil {
			return "", fmt.Errorf("formatting data size: %w", err)
		}
		values = append(values, size)
	}
	for index, value := range data.Values {
		formatted, err := ast.FormatExpression(value)
		if err != nil {
			return "", fmt.Errorf("formatting data value %d: %w", index, err)
		}
		values = append(values, formatted)
	}
	if len(values) == 0 {
		return "", fmt.Errorf("%w: data node has no values", ErrFormattingUnsupported)
	}
	return directive + " " + strings.Join(values, ", "), nil
}

func (c *Codec[T]) dataDirective(data ast.Data) (string, error) {
	if data.Type == ast.AddressType {
		return c.addressDirective(data)
	}
	if data.Type != ast.DataType {
		return "", fmt.Errorf("%w: data type %d", ErrFormattingUnsupported, data.Type)
	}

	key := dataDirectiveKey{fill: data.Fill, width: data.Width}
	if directive, ok := dataDirectiveNames[key]; ok {
		return directive, nil
	}
	if c.configuration.CompatibilityMode == config.CompatX816 {
		if directive, ok := x816DataDirectiveNames[key]; ok {
			return directive, nil
		}
	}
	return "", fmt.Errorf(
		"%w: data width %d in %s mode",
		ErrFormattingUnsupported,
		data.Width,
		c.configuration.CompatibilityMode,
	)
}

func (c *Codec[T]) addressDirective(data ast.Data) (string, error) {
	switch data.ReferenceType {
	case ast.FullAddress:
		if data.Width*8 == c.configuration.Arch.AddressWidth() {
			return ".addr", nil
		}
		if data.Width == 3 && c.configuration.CompatibilityMode == config.CompatCa65 {
			return ".faraddr", nil
		}
	case ast.LowAddressByte:
		if c.configuration.CompatibilityMode != config.CompatX816 {
			return ".dl", nil
		}
	case ast.HighAddressByte:
		return ".dh", nil
	case ast.BankAddressByte:
		if c.configuration.CompatibilityMode == config.CompatCa65 {
			return ".bankbytes", nil
		}
	}
	return "", fmt.Errorf(
		"%w: address reference type %d with width %d in %s mode",
		ErrFormattingUnsupported,
		data.ReferenceType,
		data.Width,
		c.configuration.CompatibilityMode,
	)
}

// ParseWithState reads an assembly stream from an explicit target state and
// returns the state after the last parsed instruction.
func ParseWithState[T, S any](
	ctx context.Context,
	c *Codec[T],
	source io.Reader,
	initialState S,
) ([]ast.Node, S, error) {

	var zero S
	stream, err := ParseStreamWithState(ctx, c, "", source, initialState)
	if err != nil {
		return nil, zero, err
	}
	_, finalState, ok := ast.StateSnapshots[S](stream)
	if !ok {
		return nil, zero, ErrStateType
	}
	return stream.Nodes(), finalState, nil
}

// ParseStreamWithState reads assembly into an owned typed stream from an explicit target state.
func ParseStreamWithState[T, S any](
	ctx context.Context,
	c *Codec[T],
	sourceName string,
	source io.Reader,
	initialState S,
) (*ast.Stream, error) {

	stream, err := c.parseStream(ctx, sourceName, source, initialState)
	if err != nil {
		return nil, err
	}
	if _, _, ok := ast.StateSnapshots[S](stream); !ok {
		return nil, ErrStateType
	}
	return stream, nil
}

// ParseInstructionWithState resolves exactly one instruction from an explicit
// target state and returns the state after that instruction.
func ParseInstructionWithState[T, S any](
	ctx context.Context,
	c *Codec[T],
	source io.Reader,
	initialState S,
) (ast.Instruction, S, error) {

	var zero S
	nodes, finalState, err := ParseWithState(ctx, c, source, initialState)
	if err != nil {
		return ast.Instruction{}, zero, err
	}
	instruction, err := singleInstruction(nodes)
	if err != nil {
		return ast.Instruction{}, zero, err
	}
	return instruction, finalState, nil
}

// BuildInstruction constructs a typed instruction through the architecture's
// operand-specific builder. Architectures opt in with their concrete operand-set type.
func BuildInstruction[T, O any](c *Codec[T], mnemonic string, operands O) (ast.Instruction, error) {
	builder, ok := c.configuration.Arch.(instructionBuilder[O])
	if !ok {
		return ast.Instruction{}, ErrBuildUnsupported
	}
	instruction, err := builder.BuildInstruction(mnemonic, operands)
	if err != nil {
		return ast.Instruction{}, fmt.Errorf("building typed instruction: %w", err)
	}
	return instruction, nil
}

// BuildInstructionWithState constructs a typed instruction using explicit
// target stream state and returns the state after the instruction.
func BuildInstructionWithState[T, O, S any](
	c *Codec[T],
	mnemonic string,
	operands O,
	state S,
) (ast.Instruction, S, error) {

	var zero S
	builder, ok := c.configuration.Arch.(statefulInstructionBuilder[O, S])
	if !ok {
		return ast.Instruction{}, zero, ErrBuildUnsupported
	}
	instruction, nextState, err := builder.BuildInstructionWithState(mnemonic, operands, state)
	if err != nil {
		return ast.Instruction{}, zero, fmt.Errorf("building stateful typed instruction: %w", err)
	}
	return instruction, nextState, nil
}

type dataDirectiveKey struct {
	fill  bool
	width int
}

type instructionBuilder[O any] interface {
	BuildInstruction(string, O) (ast.Instruction, error)
}

type statefulInstructionBuilder[O, S any] interface {
	BuildInstructionWithState(string, O, S) (ast.Instruction, S, error)
}

type instructionValidator interface {
	ValidateInstruction(ast.Instruction) error
}

type instructionFormatter interface {
	FormatInstruction(ast.Instruction) (string, error)
}

func singleInstruction(nodes []ast.Node) (ast.Instruction, error) {
	if len(nodes) != 1 {
		return ast.Instruction{}, fmt.Errorf("%w: got %d nodes", ErrExpectedInstruction, len(nodes))
	}
	instruction, ok := ast.InstructionFromNode(nodes[0])
	if !ok {
		return ast.Instruction{}, fmt.Errorf("%w: got %T", ErrExpectedInstruction, nodes[0])
	}
	return instruction, nil
}

func reconcileInstructionRelocations(stream *ast.Stream, selected []ast.Relocation) error {
	entries := stream.Entries()
	existingByEntry := make([][]ast.Relocation, len(entries))
	selectedByEntry := make([][]ast.Relocation, len(entries))

	for _, relocation := range stream.Relocations() {
		if _, ok := ast.InstructionFromNode(entries[relocation.EntryIndex].Node); ok {
			existingByEntry[relocation.EntryIndex] = append(existingByEntry[relocation.EntryIndex], relocation)
		}
	}
	for _, relocation := range selected {
		if relocation.EntryIndex < 0 || relocation.EntryIndex >= len(entries) {
			return fmt.Errorf("%w: entry index %d", ErrInstructionRelocationMismatch, relocation.EntryIndex)
		}
		if _, ok := ast.InstructionFromNode(entries[relocation.EntryIndex].Node); !ok {
			return fmt.Errorf("%w: entry %d is not an instruction", ErrInstructionRelocationMismatch, relocation.EntryIndex)
		}
		selectedByEntry[relocation.EntryIndex] = append(selectedByEntry[relocation.EntryIndex], relocation)
	}

	for entryIndex, entry := range entries {
		if _, ok := ast.InstructionFromNode(entry.Node); !ok {
			continue
		}
		existing := existingByEntry[entryIndex]
		generated := selectedByEntry[entryIndex]
		if len(existing) == 0 {
			for _, relocation := range generated {
				stream.RecordRelocation(relocation)
			}
			continue
		}
		if !slices.Equal(existing, generated) {
			return fmt.Errorf("%w at entry %d", ErrInstructionRelocationMismatch, entryIndex)
		}
	}
	return nil
}

func formatAlias(alias ast.Alias) (string, error) {
	if !validIdentifier(alias.Name) {
		return "", fmt.Errorf("%w: alias name %q", ErrFormattingUnsupported, alias.Name)
	}

	value, err := ast.FormatExpression(alias.Expression)
	if err != nil {
		return "", fmt.Errorf("%w: formatting alias expression: %w", ErrFormattingUnsupported, err)
	}

	switch {
	case alias.SymbolReusable && alias.Expression.IsEvaluatedOnce():
		return alias.Name + " = " + value, nil
	case !alias.SymbolReusable && !alias.Expression.IsEvaluatedOnce():
		return alias.Name + " EQU " + value, nil
	default:
		return "", fmt.Errorf("%w: alias evaluation and reuse policy differ", ErrFormattingUnsupported)
	}
}

func formatBase(base ast.Base) (string, error) {
	value, err := ast.FormatExpression(base.Address)
	if err != nil {
		return "", fmt.Errorf("%w: formatting base expression: %w", ErrFormattingUnsupported, err)
	}
	return ".org " + value, nil
}

func formatBank(bank ast.Bank) (string, error) {
	if bank.Number < 0 {
		return "", fmt.Errorf("%w: negative bank %d", ErrFormattingUnsupported, bank.Number)
	}
	return ".bank " + strconv.Itoa(bank.Number), nil
}

func formatSegment(segment ast.Segment) (string, error) {
	if validIdentifier(segment.Name) || validQuotedValue(segment.Name) {
		return ".segment " + segment.Name, nil
	}
	return "", fmt.Errorf("%w: segment name %q", ErrFormattingUnsupported, segment.Name)
}

func formatVariable(variable ast.Variable) (string, error) {
	if variable.Size < 0 {
		return "", fmt.Errorf("%w: negative variable size %d", ErrFormattingUnsupported, variable.Size)
	}

	size := strconv.Itoa(variable.Size)
	switch {
	case variable.UseOffsetCounter && validIdentifier(variable.Name):
		return variable.Name + " .rs " + size, nil
	case !variable.UseOffsetCounter && variable.Name == "":
		return ".res " + size, nil
	default:
		return "", fmt.Errorf("%w: variable name and offset-counter policy differ", ErrFormattingUnsupported)
	}
}

func configurationSourceValue(item ast.ConfigurationItem, value uint64) (uint64, bool) {
	switch item {
	case ast.ConfigPrg:
		return compressedConfigurationValue(value, 16384)
	case ast.ConfigChr:
		return compressedConfigurationValue(value, 8192)
	default:
		return value, true
	}
}

func compressedConfigurationValue(value, unit uint64) (uint64, bool) {
	// The parser expands small PRG and CHR counts into byte sizes.
	if value%unit == 0 {
		units := value / unit
		if units < 0xeff {
			return units, true
		}
	}
	if value >= 0xeff {
		return value, true
	}
	return 0, false
}

func isAsm6Configuration(item ast.ConfigurationItem) bool {
	return item >= ast.ConfigNes2ChrRAM && item <= ast.ConfigNes2ChrBRam
}

func validIdentifier(value string) bool {
	for index, character := range value {
		if index == 0 && !unicode.IsLetter(character) && character != '_' && character != '@' {
			return false
		}
		if index > 0 &&
			!unicode.IsLetter(character) &&
			!unicode.IsDigit(character) &&
			character != '$' &&
			character != '_' &&
			character != '-' {

			return false
		}
	}
	return value != ""
}

func validQuotedValue(value string) bool {
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return false
	}
	_, err := strconv.Unquote(value)
	return err == nil
}

func streamEntryError(action string, index int, position ast.SourcePosition, err error) error {
	return fmt.Errorf(
		"%s typed stream entry %d at %s:%d:%d: %w",
		action,
		index,
		position.Source,
		position.Line,
		position.Column,
		err,
	)
}
