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
	// ErrUnknownInstruction indicates that the target does not register a mnemonic.
	ErrUnknownInstruction = errors.New("unknown target instruction")
	// ErrExpectedInstruction indicates that a single-instruction parse produced another node shape.
	ErrExpectedInstruction = errors.New("expected exactly one instruction")
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
	if source == nil {
		return nil, ErrNilSource
	}

	p := parser.New(c.configuration.Arch, source, c.configuration.CompatibilityMode)
	if err := p.Read(ctx); err != nil {
		return nil, fmt.Errorf("reading assembly stream: %w", err)
	}
	nodes, err := p.TokensToAstNodes()
	if err != nil {
		return nil, fmt.Errorf("resolving assembly stream: %w", err)
	}
	return nodes, nil
}

// ParseInstruction resolves a source stream containing exactly one instruction.
func (c *Codec[T]) ParseInstruction(ctx context.Context, source io.Reader) (ast.Instruction, error) {
	nodes, err := c.Parse(ctx, source)
	if err != nil {
		return ast.Instruction{}, err
	}
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

// Assemble assembles a copy of nodes directly, without formatting or reparsing text.
func (c *Codec[T]) Assemble(ctx context.Context, nodes []ast.Node) (*Assembly, error) {
	var output bytes.Buffer
	asm := assembler.New(c.configuration, &output)
	if err := asm.ProcessAST(ctx, ast.CopyNodes(nodes)); err != nil {
		return nil, fmt.Errorf("assembling typed stream: %w", err)
	}
	return &Assembly{
		Binary:  output.Bytes(),
		Symbols: maps.Clone(asm.Symbols()),
	}, nil
}
