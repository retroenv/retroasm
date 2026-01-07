package retroasm

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/retroenv/retroasm/pkg/arch/m6502"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch"
	cpu "github.com/retroenv/retrogolib/arch/cpu/m6502"
	"github.com/retroenv/retrogolib/assert"
)

// Test constants.
const (
	testFilename = "test.asm"
)

func TestAssemblerCreation(t *testing.T) {
	assembler := New()
	assert.NotNil(t, assembler)
}

func TestArchitectureRegistration(t *testing.T) {
	tests := []struct {
		name        string
		archName    string
		arch        Architecture
		expectedErr error
	}{
		{
			name:        "successful registration",
			archName:    string(arch.M6502),
			arch:        NewArchitectureAdapter(string(arch.M6502), m6502.New(), m6502.New()),
			expectedErr: nil,
		},
		{
			name:        "nil architecture",
			archName:    "null",
			arch:        nil,
			expectedErr: ErrNilArchitecture,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assembler := New()
			err := assembler.RegisterArchitecture(tt.archName, tt.arch)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigurationSetting(t *testing.T) {
	tests := []struct {
		name        string
		config      Configuration
		expectedErr error
	}{
		{
			name:        "successful configuration",
			config:      NewDefaultConfiguration(),
			expectedErr: nil,
		},
		{
			name:        "nil configuration",
			config:      nil,
			expectedErr: ErrNilConfiguration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assembler := New()
			err := assembler.SetConfiguration(tt.config)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestASTAssembly(t *testing.T) {
	tests := []struct {
		name           string
		input          *ASTInput
		expectedErr    error
		expectedBinary []byte
	}{
		{
			name:           "empty AST",
			input:          &ASTInput{AST: []ast.Node{}, SourceName: testFilename},
			expectedBinary: nil,
		},
		{
			name:        "nil input",
			input:       nil,
			expectedErr: ErrNilInput,
		},
		{
			name: "simple instruction lowercase",
			input: &ASTInput{
				AST:        []ast.Node{ast.NewInstruction("lda", int(cpu.ImmediateAddressing), ast.NewNumber(1), nil)},
				SourceName: testFilename,
			},
			expectedBinary: []byte{0xA9, 0x01}, // LDA #$01
		},
		{
			name: "simple instruction uppercase",
			input: &ASTInput{
				AST:        []ast.Node{ast.NewInstruction("LDA", int(cpu.ImmediateAddressing), ast.NewNumber(1), nil)},
				SourceName: testFilename,
			},
			expectedBinary: []byte{0xA9, 0x01}, // LDA #$01
		},
		{
			name: "multiple instructions",
			input: &ASTInput{
				AST: []ast.Node{
					ast.NewInstruction("LDA", int(cpu.ImmediateAddressing), ast.NewNumber(1), nil),
					ast.NewInstruction("STA", int(cpu.AbsoluteAddressing), ast.NewNumber(0x0200), nil),
				},
				SourceName: testFilename,
			},
			expectedBinary: []byte{0xA9, 0x01, 0x8D, 0x00, 0x02}, // LDA #$01; STA $0200
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := runASTAssembly(tt.input)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, output)
			assert.Equal(t, tt.expectedBinary, output.Binary)
		})
	}
}

func runASTAssembly(input *ASTInput) (*AssemblyOutput, error) {
	assembler := New()
	m6502Arch := m6502.New()
	adapter := NewArchitectureAdapter(string(arch.M6502), m6502Arch, m6502Arch)
	if err := assembler.RegisterArchitecture(string(arch.M6502), adapter); err != nil {
		return nil, fmt.Errorf("registering architecture: %w", err)
	}
	output, err := assembler.AssembleAST(context.Background(), input)
	if err != nil {
		return nil, fmt.Errorf("assembling AST: %w", err)
	}
	return output, nil
}

func TestTextAssembly(t *testing.T) {
	tests := []struct {
		name           string
		input          *TextInput
		expectedErr    error
		expectedBinary []byte
	}{
		{
			name: "simple instruction",
			input: &TextInput{
				Source:     strings.NewReader(".segment \"CODE\"\nLDA #$01"),
				SourceName: testFilename,
				Format:     FormatCa65,
			},
			expectedBinary: []byte{0xA9, 0x01}, // LDA #$01
		},
		{
			name:        "nil input",
			input:       nil,
			expectedErr: ErrNilInput,
		},
		{
			name: "nil source",
			input: &TextInput{
				Source:     nil,
				SourceName: testFilename,
				Format:     FormatAsm6,
			},
			expectedErr: ErrNilSource,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assembler := New()

			// Register architecture
			m6502Arch := m6502.New()
			adapter := NewArchitectureAdapter(string(arch.M6502), m6502Arch, m6502Arch)
			err := assembler.RegisterArchitecture(string(arch.M6502), adapter)
			assert.NoError(t, err)

			output, err := assembler.AssembleText(context.Background(), tt.input)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, output)
			assert.Equal(t, tt.expectedBinary, output.Binary)
		})
	}
}

func TestConfigurationBuilder(t *testing.T) {
	config := NewConfigurationBuilder().
		SetSymbol("test", 0x1000).
		AddSegment(SegmentConfig{
			Name:      "CODE",
			StartAddr: 0x8000,
			Size:      0x8000,
			Type:      SegmentTypeCode,
		}).
		Build()

	symbols := config.Symbols()
	assert.Equal(t, uint64(0x1000), symbols["test"])

	segments := config.Segments()
	assert.Equal(t, 1, len(segments))
	assert.Equal(t, "CODE", segments[0].Name)
	assert.Equal(t, uint64(0x8000), segments[0].StartAddr)
	assert.Equal(t, uint64(0x8000), segments[0].Size)
	assert.Equal(t, SegmentTypeCode, segments[0].Type)
}

func TestSymbolHandling(t *testing.T) {
	assembler := New()

	// Register architecture
	m6502Arch := m6502.New()
	adapter := NewArchitectureAdapter(string(arch.M6502), m6502Arch, m6502Arch)
	err := assembler.RegisterArchitecture(string(arch.M6502), adapter)
	assert.NoError(t, err)

	// Test symbol passing
	symbols := map[string]uint64{
		"start": 0x8000,
		"end":   0xFFFF,
	}

	input := &ASTInput{
		AST:        []ast.Node{},
		SourceName: testFilename,
		Symbols:    symbols,
	}

	output, err := assembler.AssembleAST(context.Background(), input)
	assert.NoError(t, err)

	// Check that symbols are preserved in output
	assert.Equal(t, len(symbols), len(output.Symbols))

	for name, expectedValue := range symbols {
		symbol, exists := output.Symbols[name]
		assert.True(t, exists, "Expected symbol '%s' in output", name)
		assert.Equal(t, expectedValue, symbol.Value, "Symbol '%s' value mismatch", name)
		assert.Equal(t, SymbolTypeConstant, symbol.Type)
		assert.Equal(t, name, symbol.Name)
		assert.Equal(t, testFilename, symbol.Location.Filename)
	}
}

// TestDefaultConfiguration tests the default configuration implementation.
func TestDefaultConfiguration(t *testing.T) {
	config := NewDefaultConfiguration()
	assert.NotNil(t, config)

	// Test default memory layout
	layout := config.MemoryLayout()
	assert.Equal(t, 16, layout.AddressSize)
	assert.Equal(t, LittleEndian, layout.Endianness)

	// Test empty segments and symbols
	assert.Equal(t, 0, len(config.Segments()))
	assert.Equal(t, 0, len(config.Symbols()))
}

// TestDefaultConfigurationMutation tests configuration modification methods.
func TestDefaultConfigurationMutation(t *testing.T) {
	config := NewDefaultConfiguration().(*DefaultConfiguration)

	// Test memory layout modification
	newLayout := MemoryLayout{AddressSize: 8, Endianness: BigEndian}
	config.SetMemoryLayout(newLayout)
	layout := config.MemoryLayout()
	assert.Equal(t, 8, layout.AddressSize)
	assert.Equal(t, BigEndian, layout.Endianness)

	// Test segment addition
	segment := SegmentConfig{
		Name:      "TEST",
		StartAddr: 0x1000,
		Size:      0x1000,
		Type:      SegmentTypeData,
	}
	config.AddSegment(segment)
	segments := config.Segments()
	assert.Equal(t, 1, len(segments))
	assert.Equal(t, "TEST", segments[0].Name)

	// Test symbol setting
	config.SetSymbol("test_symbol", 0x2000)
	symbols := config.Symbols()
	assert.Equal(t, uint64(0x2000), symbols["test_symbol"])
}

// TestArchitectureAdapter tests the architecture adapter functionality.
func TestArchitectureAdapter(t *testing.T) {
	m6502Arch := m6502.New()
	adapter := NewArchitectureAdapter(string(arch.M6502), m6502Arch, m6502Arch)

	// Test adapter properties
	assert.Equal(t, string(arch.M6502), adapter.Name())
	assert.Equal(t, 16, adapter.AddressWidth()) // 6502 default

	// Test assembler creation
	config := ArchitectureConfig{
		BaseAddr: 0x8000,
		Symbols:  map[string]uint64{"test": 0x1000},
	}
	archAssembler, err := adapter.CreateAssembler(config)
	assert.NoError(t, err)
	assert.NotNil(t, archAssembler)
}

// TestArchitectureAssemblerImpl tests the architecture assembler implementation.
func TestArchitectureAssemblerImpl(t *testing.T) {
	m6502Arch := m6502.New()
	adapter := NewArchitectureAdapter(string(arch.M6502), m6502Arch, m6502Arch)
	config := ArchitectureConfig{
		BaseAddr: 0x8000,
		Symbols:  map[string]uint64{"test": 0x1000},
	}

	archAssembler, err := adapter.CreateAssembler(config)
	assert.NoError(t, err)

	// Test AST assembly (minimal bridge implementation)
	nodes := []ast.Node{
		ast.NewInstruction("NOP", 0, nil, nil),
	}
	output, err := archAssembler.AssembleAST(nodes)
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, nodes, output.AST)
}

// TestErrorValues tests that error constants are properly defined.
func TestErrorValues(t *testing.T) {
	// Test that sentinel errors are not nil and have proper messages
	assert.NotNil(t, ErrNilArchitecture)
	assert.NotNil(t, ErrNilConfiguration)
	assert.NotNil(t, ErrNilInput)
	assert.NotNil(t, ErrNilSource)

	// Test error messages
	assert.Equal(t, "architecture cannot be nil", ErrNilArchitecture.Error())
	assert.Equal(t, "configuration cannot be nil", ErrNilConfiguration.Error())
	assert.Equal(t, "input cannot be nil", ErrNilInput.Error())
	assert.Equal(t, "source cannot be nil", ErrNilSource.Error())
}

// TestAssemblyOutputStructure tests the structure and types of AssemblyOutput.
func TestAssemblyOutputStructure(t *testing.T) {
	output := &AssemblyOutput{
		Binary:      make([]byte, 0, 1024),
		AST:         []ast.Node{},
		Symbols:     make(map[string]Symbol, 16),
		Segments:    make([]Segment, 0, 8),
		Diagnostics: make([]Diagnostic, 0, 4),
	}

	// Test field types and initial values
	assert.NotNil(t, output.Binary)
	assert.NotNil(t, output.AST)
	assert.NotNil(t, output.Symbols)
	assert.NotNil(t, output.Segments)
	assert.NotNil(t, output.Diagnostics)

	assert.Equal(t, 0, len(output.Binary))
	assert.Equal(t, 0, len(output.AST))
	assert.Equal(t, 0, len(output.Symbols))
	assert.Equal(t, 0, len(output.Segments))
	assert.Equal(t, 0, len(output.Diagnostics))
}

// TestSymbolTypes tests the different symbol types.
func TestSymbolTypes(t *testing.T) {
	// Test symbol type constants
	assert.Equal(t, SymbolType(0), SymbolTypeLabel)
	assert.Equal(t, SymbolType(1), SymbolTypeConstant)
	assert.Equal(t, SymbolType(2), SymbolTypeVariable)
}

// TestDiagnosticLevels tests the diagnostic level constants.
func TestDiagnosticLevels(t *testing.T) {
	// Test diagnostic level constants
	assert.Equal(t, DiagnosticLevel(0), DiagnosticError)
	assert.Equal(t, DiagnosticLevel(1), DiagnosticWarning)
	assert.Equal(t, DiagnosticLevel(2), DiagnosticInfo)
}

// TestEndianness tests the endianness constants.
func TestEndianness(t *testing.T) {
	// Test endianness constants
	assert.Equal(t, Endianness(0), LittleEndian)
	assert.Equal(t, Endianness(1), BigEndian)
}

// TestSegmentTypes tests the segment type constants.
func TestSegmentTypes(t *testing.T) {
	// Test segment type constants
	assert.Equal(t, SegmentType(0), SegmentTypeCode)
	assert.Equal(t, SegmentType(1), SegmentTypeData)
	assert.Equal(t, SegmentType(2), SegmentTypeBSS)
}
