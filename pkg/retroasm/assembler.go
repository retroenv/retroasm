package retroasm

import (
	"context"
	"io"

	"github.com/retroenv/retroasm/parser/ast"
)

// Common assembly format constants.
const (
	FormatAsm6   = "asm6"
	FormatCa65   = "ca65"
	FormatNesasm = "nesasm"
)

// Assembler is the main interface for assembly operations.
type Assembler interface {
	AssembleAST(ctx context.Context, input *ASTInput) (*AssemblyOutput, error)
	AssembleText(ctx context.Context, input *TextInput) (*AssemblyOutput, error)
	RegisterArchitecture(name string, arch Architecture) error
	SetConfiguration(config Configuration) error
}

// ASTInput represents direct AST input.
type ASTInput struct {
	AST        []ast.Node
	Symbols    map[string]uint64
	SourceName string
	BaseAddr   uint64
}

// TextInput represents text-based assembly input.
type TextInput struct {
	Source     io.Reader
	SourceName string
	Format     string // "asm6", "ca65", "nesasm"
	ConfigFile string // optional ca65 config file path
	Symbols    map[string]uint64
}

// AssemblyOutput contains the results of assembly.
type AssemblyOutput struct {
	Binary      []byte
	AST         []ast.Node
	Symbols     map[string]Symbol
	Segments    []Segment
	Diagnostics []Diagnostic
}

// Symbol represents a symbol definition.
type Symbol struct {
	Name     string
	Value    uint64
	Type     SymbolType
	Segment  string
	Location SourceLocation
}

// SymbolType represents the type of a symbol.
type SymbolType int

const (
	SymbolTypeLabel SymbolType = iota
	SymbolTypeConstant
	SymbolTypeVariable
)

// Segment represents a memory segment.
type Segment struct {
	Name      string
	StartAddr uint64
	Size      uint64
	Data      []byte
}

// Diagnostic represents a warning or error from assembly.
type Diagnostic struct {
	Level    DiagnosticLevel
	Message  string
	Location SourceLocation
	Code     string
	Hints    []string
}

// DiagnosticLevel represents the severity of a diagnostic.
type DiagnosticLevel int

const (
	DiagnosticError DiagnosticLevel = iota
	DiagnosticWarning
	DiagnosticInfo
)

// SourceLocation represents a location in source code.
type SourceLocation struct {
	Filename string
	Line     int
	Column   int
	Length   int
}

// Architecture defines the interface for CPU architectures.
type Architecture interface {
	Name() string
	AddressWidth() int
	CreateAssembler(config ArchitectureConfig) (ArchitectureAssembler, error)
}

// ArchitectureAssembler handles architecture-specific assembly.
type ArchitectureAssembler interface {
	AssembleAST(nodes []ast.Node) (*AssemblyOutput, error)
}

// ArchitectureConfig contains configuration for architecture-specific operations.
type ArchitectureConfig struct {
	BaseAddr uint64
	Symbols  map[string]uint64
}

// Configuration defines assembler configuration.
type Configuration interface {
	MemoryLayout() MemoryLayout
	Segments() []SegmentConfig
	Symbols() map[string]uint64
}

// MemoryLayout defines memory organization.
type MemoryLayout struct {
	AddressSize int
	Endianness  Endianness
}

// Endianness represents byte order.
type Endianness int

const (
	LittleEndian Endianness = iota
	BigEndian
)

// SegmentConfig defines a memory segment configuration.
type SegmentConfig struct {
	Name      string
	StartAddr uint64
	Size      uint64
	Type      SegmentType
}

// SegmentType represents the type of a memory segment.
type SegmentType int

const (
	SegmentTypeCode SegmentType = iota
	SegmentTypeData
	SegmentTypeBSS
)
