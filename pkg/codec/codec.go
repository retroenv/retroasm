package codec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"strings"

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
	// ErrFormattingUnsupported indicates that an architecture has no deterministic instruction formatter.
	ErrFormattingUnsupported = errors.New("typed instruction formatting is not supported by architecture")
	// ErrStateType indicates that an architecture returned a different stream-state type.
	ErrStateType = errors.New("unexpected architecture stream state type")
)

// Assembly is the result of assembling a typed node stream.
type Assembly struct {
	Binary  []byte
	Symbols map[string]uint64
}

// Codec binds typed stream operations to one architecture configuration.
type Codec[T any] struct {
	configuration *config.Config[T]
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
	return stream, nil
}

// ParseInstruction resolves a source stream containing exactly one instruction.
func (c *Codec[T]) ParseInstruction(ctx context.Context, source io.Reader) (ast.Instruction, error) {
	nodes, err := c.Parse(ctx, source)
	if err != nil {
		return ast.Instruction{}, err
	}
	return singleInstruction(nodes)
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

// ValidateStream checks stream metadata and every architecture-specific instruction.
func (c *Codec[T]) ValidateStream(stream *ast.Stream) error {
	if stream == nil {
		return ErrNilStream
	}
	if err := stream.Validate(); err != nil {
		return fmt.Errorf("validating typed stream: %w", err)
	}

	for index, entry := range stream.Entries() {
		instruction, ok := ast.InstructionFromNode(entry.Node)
		if !ok {
			continue
		}
		if err := c.ValidateInstruction(instruction); err != nil {
			return fmt.Errorf(
				"validating typed stream entry %d at %s:%d:%d: %w",
				index,
				entry.Position.Source,
				entry.Position.Line,
				entry.Position.Column,
				err,
			)
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

// Assemble assembles a copy of nodes directly, without formatting or reparsing text.
func (c *Codec[T]) Assemble(ctx context.Context, nodes []ast.Node) (*Assembly, error) {
	return c.AssembleStream(ctx, ast.NewStreamFromNodes(nodes...))
}

// AssembleStream assembles a typed stream directly, without formatting or reparsing text.
func (c *Codec[T]) AssembleStream(ctx context.Context, stream *ast.Stream) (*Assembly, error) {
	if stream == nil {
		return nil, ErrNilStream
	}

	var output bytes.Buffer
	asm := assembler.New(c.configuration, &output)
	if err := asm.ProcessAST(ctx, stream.Nodes()); err != nil {
		return nil, fmt.Errorf("assembling typed stream: %w", err)
	}
	return &Assembly{
		Binary:  output.Bytes(),
		Symbols: maps.Clone(asm.Symbols()),
	}, nil
}
