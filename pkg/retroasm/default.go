package retroasm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch/m6502"
	"github.com/retroenv/retroasm/pkg/assembler"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// Sentinel errors.
var (
	ErrNilArchitecture  = errors.New("architecture cannot be nil")
	ErrNilConfiguration = errors.New("configuration cannot be nil")
	ErrNilInput         = errors.New("input cannot be nil")
	ErrNilSource        = errors.New("source cannot be nil")
)

// New creates a new assembler instance.
func New() Assembler {
	return &defaultAssembler{
		architectures: make(map[string]Architecture, 4),
	}
}

// ArchitectureAdapter adapts existing architectures to the Architecture interface.
type ArchitectureAdapter[T any] struct {
	arch   any
	name   string
	config *config.Config[T]
}

// NewArchitectureAdapter creates a new adapter for existing architectures.
func NewArchitectureAdapter[T any](name string, arch any, cfg *config.Config[T]) Architecture {
	return &ArchitectureAdapter[T]{
		arch:   arch,
		name:   name,
		config: cfg,
	}
}

func (a *ArchitectureAdapter[T]) Name() string      { return a.name }
func (a *ArchitectureAdapter[T]) AddressWidth() int { return 16 }

func (a *ArchitectureAdapter[T]) CreateAssembler(cfg ArchitectureConfig) (ArchitectureAssembler, error) {
	return &architectureAssembler[T]{
		arch:   a.arch,
		config: cfg,
	}, nil
}

type defaultAssembler struct {
	architectures map[string]Architecture
	config        Configuration
}

func (a *defaultAssembler) RegisterArchitecture(name string, arch Architecture) error {
	if arch == nil {
		return ErrNilArchitecture
	}
	a.architectures[name] = arch
	return nil
}

func (a *defaultAssembler) SetConfiguration(cfg Configuration) error {
	if cfg == nil {
		return ErrNilConfiguration
	}
	a.config = cfg
	return nil
}

func (a *defaultAssembler) AssembleAST(ctx context.Context, input *ASTInput) (*AssemblyOutput, error) {
	if input == nil {
		return nil, ErrNilInput
	}

	// Create m6502 config with default segment
	cfg := m6502.New()
	if err := cfg.ReadCa65Config(strings.NewReader(defaultConfig)); err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	// Override start address if provided
	if input.BaseAddr != 0 {
		for _, seg := range cfg.Segments {
			seg.Start = input.BaseAddr
			seg.SegmentStart = input.BaseAddr
		}
	}

	// Prepend segment directive if AST doesn't start with one
	nodes := input.AST
	if len(nodes) > 0 {
		if _, ok := nodes[0].(ast.Segment); !ok {
			nodes = append([]ast.Node{ast.NewSegment("CODE")}, nodes...)
		}
	}

	// Assemble using existing assembler
	var buf bytes.Buffer
	asm := assembler.New(cfg, &buf)

	if err := asm.ProcessAST(ctx, nodes); err != nil {
		return nil, fmt.Errorf("processing AST: %w", err)
	}

	output := &AssemblyOutput{
		Binary:  buf.Bytes(),
		AST:     input.AST,
		Symbols: copyInputSymbols(input.Symbols, input.SourceName),
	}

	return output, nil
}

func (a *defaultAssembler) AssembleText(ctx context.Context, input *TextInput) (*AssemblyOutput, error) {
	if input == nil {
		return nil, ErrNilInput
	}
	if input.Source == nil {
		return nil, ErrNilSource
	}

	// Create m6502 config
	cfg := m6502.New()

	// Load config file if specified, otherwise use default
	if input.ConfigFile != "" {
		cfgData, err := os.ReadFile(input.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf("opening config file '%s': %w", input.ConfigFile, err)
		}
		if err := cfg.ReadCa65Config(bytes.NewReader(cfgData)); err != nil {
			return nil, fmt.Errorf("reading config file '%s': %w", input.ConfigFile, err)
		}
	} else {
		if err := cfg.ReadCa65Config(strings.NewReader(defaultConfig)); err != nil {
			return nil, fmt.Errorf("reading default config: %w", err)
		}
	}

	// Assemble using existing assembler
	var buf bytes.Buffer
	asm := assembler.New(cfg, &buf)

	if err := asm.Process(ctx, input.Source); err != nil {
		return nil, fmt.Errorf("processing text: %w", err)
	}

	output := &AssemblyOutput{
		Binary:  buf.Bytes(),
		Symbols: copyInputSymbols(input.Symbols, input.SourceName),
	}

	return output, nil
}

type architectureAssembler[T any] struct {
	arch   any
	config ArchitectureConfig
}

func (a *architectureAssembler[T]) AssembleAST(nodes []ast.Node) (*AssemblyOutput, error) {
	return &AssemblyOutput{
		AST:     nodes,
		Symbols: make(map[string]Symbol),
	}, nil
}

// copyInputSymbols converts a map of symbol names to values into the output Symbol map.
func copyInputSymbols(symbols map[string]uint64, sourceName string) map[string]Symbol {
	result := make(map[string]Symbol, len(symbols))
	for name, value := range symbols {
		result[name] = Symbol{
			Name:  name,
			Value: value,
			Type:  SymbolTypeConstant,
			Location: SourceLocation{
				Filename: sourceName,
			},
		}
	}
	return result
}

const defaultConfig = `
MEMORY {
    CODE: start = $8000, size = $8000, fill = yes;
}
SEGMENTS {
    CODE: load = CODE, type = rw;
}
`
