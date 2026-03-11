package retroasm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch/m6502"
	archz80 "github.com/retroenv/retroasm/pkg/arch/z80"
	"github.com/retroenv/retroasm/pkg/assembler"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpum6502 "github.com/retroenv/retrogolib/arch/cpu/m6502"
	cpum68000 "github.com/retroenv/retrogolib/arch/cpu/m68000"
)

// Sentinel errors.
var (
	ErrNilArchitecture               = errors.New("architecture cannot be nil")
	ErrNilConfiguration              = errors.New("configuration cannot be nil")
	ErrNilInput                      = errors.New("input cannot be nil")
	ErrNilSource                     = errors.New("source cannot be nil")
	errAmbiguousArchitecture         = errors.New("multiple architectures registered without explicit selection")
	errArchitectureAdapterMismatch   = errors.New("architecture does not expose a supported adapter config")
	errArchitectureNotRegistered     = errors.New("requested architecture is not registered")
	errUnsupportedArchitectureConfig = errors.New("unsupported architecture config type")
)

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

// New creates a new assembler instance.
func New() Assembler {
	return &defaultAssembler{
		architectures: make(map[string]Architecture, 4),
	}
}

func (a *ArchitectureAdapter[T]) Name() string { return a.name }
func (a *ArchitectureAdapter[T]) AddressWidth() int {
	if aw, ok := a.arch.(interface{ AddressWidth() int }); ok {
		return aw.AddressWidth()
	}
	return 16
}

func (a *ArchitectureAdapter[T]) CreateAssembler(cfg ArchitectureConfig) (ArchitectureAssembler, error) {
	return &architectureAssembler[T]{
		arch:   a.arch,
		config: cfg,
	}, nil
}

func (a *ArchitectureAdapter[T]) configAny() any { return a.config }

type anyReader interface {
	Read(p []byte) (n int, err error)
}

type defaultAssembler struct {
	architectures map[string]Architecture
	config        Configuration
}

type architectureAssembler[T any] struct {
	arch   any
	config ArchitectureConfig
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

	// Prepend segment directive if AST doesn't start with one
	nodes := input.AST
	if len(nodes) > 0 {
		if _, ok := nodes[0].(ast.Segment); !ok {
			nodes = append([]ast.Node{ast.NewSegment("CODE")}, nodes...)
		}
	}

	output, err := a.assembleASTWithArchitecture(ctx, nodes, input.BaseAddr)
	if err != nil {
		return nil, fmt.Errorf("assembling AST: %w", err)
	}

	result := &AssemblyOutput{
		Binary:  output,
		AST:     input.AST,
		Symbols: copyInputSymbols(input.Symbols, input.SourceName),
	}

	return result, nil
}

func (a *defaultAssembler) AssembleText(ctx context.Context, input *TextInput) (*AssemblyOutput, error) {
	if input == nil {
		return nil, ErrNilInput
	}
	if input.Source == nil {
		return nil, ErrNilSource
	}

	output, err := a.assembleTextWithArchitecture(ctx, input.Source, input.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("assembling text: %w", err)
	}

	result := &AssemblyOutput{
		Binary:  output,
		Symbols: copyInputSymbols(input.Symbols, input.SourceName),
	}

	return result, nil
}

func (a *defaultAssembler) assembleASTWithArchitecture(ctx context.Context, nodes []ast.Node, baseAddress uint64) ([]byte, error) {
	cfgAny, err := a.resolveArchitectureConfig()
	if err != nil {
		return nil, err
	}

	switch cfg := cfgAny.(type) {
	case *config.Config[*cpum6502.Instruction]:
		return assembleASTWithConfig(ctx, cfg, nodes, baseAddress)
	case *config.Config[*cpum68000.Instruction]:
		return assembleASTWithConfig(ctx, cfg, nodes, baseAddress)
	case *config.Config[*archz80.InstructionGroup]:
		return assembleASTWithConfig(ctx, cfg, nodes, baseAddress)
	default:
		return nil, fmt.Errorf("%w: %T", errUnsupportedArchitectureConfig, cfgAny)
	}
}

func (a *defaultAssembler) assembleTextWithArchitecture(ctx context.Context, source anyReader, configFile string) ([]byte, error) {
	cfgAny, err := a.resolveArchitectureConfig()
	if err != nil {
		return nil, err
	}

	switch cfg := cfgAny.(type) {
	case *config.Config[*cpum6502.Instruction]:
		return assembleTextWithConfig(ctx, cfg, source, configFile)
	case *config.Config[*cpum68000.Instruction]:
		return assembleTextWithConfig(ctx, cfg, source, configFile)
	case *config.Config[*archz80.InstructionGroup]:
		return assembleTextWithConfig(ctx, cfg, source, configFile)
	default:
		return nil, fmt.Errorf("%w: %T", errUnsupportedArchitectureConfig, cfgAny)
	}
}

func (a *defaultAssembler) resolveArchitectureConfig() (any, error) {
	switch len(a.architectures) {
	case 0:
		return m6502.New(), nil

	case 1:
		for _, architecture := range a.architectures {
			return adapterConfig(architecture)
		}

	default:
		if architecture, ok := a.architectures["6502"]; ok {
			return adapterConfig(architecture)
		}
		return nil, errAmbiguousArchitecture
	}

	return nil, errArchitectureNotRegistered
}

func (a *architectureAssembler[T]) AssembleAST(nodes []ast.Node) (*AssemblyOutput, error) {
	return &AssemblyOutput{
		AST:     nodes,
		Symbols: make(map[string]Symbol),
	}, nil
}

func assembleASTWithConfig[T any](ctx context.Context, cfg *config.Config[T], nodes []ast.Node, baseAddress uint64) ([]byte, error) {
	if err := readAssemblerConfig(cfg, ""); err != nil {
		return nil, err
	}

	applyBaseAddress(cfg, baseAddress)

	var buf bytes.Buffer
	asm := assembler.New(cfg, &buf)

	if err := asm.ProcessAST(ctx, nodes); err != nil {
		return nil, fmt.Errorf("processing AST: %w", err)
	}

	return buf.Bytes(), nil
}

func assembleTextWithConfig[T any](
	ctx context.Context,
	cfg *config.Config[T],
	source anyReader,
	configFile string,
) ([]byte, error) {

	if err := readAssemblerConfig(cfg, configFile); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	asm := assembler.New(cfg, &buf)

	if err := asm.Process(ctx, source); err != nil {
		return nil, fmt.Errorf("processing text: %w", err)
	}

	return buf.Bytes(), nil
}

func readAssemblerConfig[T any](cfg *config.Config[T], configFile string) error {
	if configFile != "" {
		cfgData, err := os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("opening config file '%s': %w", configFile, err)
		}
		if err := cfg.ReadCa65Config(bytes.NewReader(cfgData)); err != nil {
			return fmt.Errorf("reading config file '%s': %w", configFile, err)
		}
		return nil
	}

	if err := cfg.ReadCa65Config(strings.NewReader(defaultConfig)); err != nil {
		return fmt.Errorf("reading default config: %w", err)
	}
	return nil
}

func applyBaseAddress[T any](cfg *config.Config[T], baseAddress uint64) {
	if baseAddress == 0 {
		return
	}

	for _, seg := range cfg.Segments {
		seg.Start = baseAddress
		seg.SegmentStart = baseAddress
	}
}

func adapterConfig(architecture Architecture) (any, error) {
	provider, ok := architecture.(interface{ configAny() any })
	if !ok {
		return nil, errArchitectureAdapterMismatch
	}
	return provider.configAny(), nil
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
