package retroasm

import (
	"context"
	"fmt"

	asmcpu6502 "github.com/retroenv/retroasm/pkg/arch/cpu6502"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/arch/cpu/cpu6502"
)

// ExampleNew demonstrates basic usage of the AST-first assembler library.
func ExampleNew() {
	// Create assembler instance
	assembler := New()

	// Register 6502 architecture using adapter
	asmcpu6502 := asmcpu6502.New()
	adapter := NewArchitectureAdapter(string(arch.CPU6502), asmcpu6502, asmcpu6502)
	err := assembler.RegisterArchitecture(string(arch.CPU6502), adapter)
	if err != nil {
		fmt.Printf("registering architecture: %v\n", err)
		return
	}

	// AST-first assembly
	program := []ast.Node{
		ast.NewInstruction("LDA", int(cpu6502.ImmediateAddressing), ast.NewNumber(1), nil),
		ast.NewInstruction("STA", int(cpu6502.AbsoluteAddressing), ast.NewNumber(0x0200), nil),
	}

	input := &ASTInput{
		AST:        program,
		SourceName: "generated.asm",
	}

	output, err := assembler.AssembleAST(context.Background(), input)
	if err != nil {
		fmt.Printf("assembling AST: %v\n", err)
		return
	}

	fmt.Printf("Binary length: %d\n", len(output.Binary))
	fmt.Printf("Binary: %X\n", output.Binary)
	// Output:
	// Binary length: 5
	// Binary: A9018D0002
}

// ExampleNewConfigurationBuilder demonstrates how to build assembler configurations.
func ExampleNewConfigurationBuilder() {
	config := NewConfigurationBuilder().
		SetMemoryLayout(MemoryLayout{
			AddressSize: 16,
			Endianness:  LittleEndian,
		}).
		SetSymbol("RESET_VECTOR", 0xFFFC).
		SetSymbol("IRQ_VECTOR", 0xFFFE).
		AddSegment(SegmentConfig{
			Name:      "CODE",
			StartAddr: 0x8000,
			Size:      0x8000,
			Type:      SegmentTypeCode,
		}).
		AddSegment(SegmentConfig{
			Name:      "RAM",
			StartAddr: 0x0000,
			Size:      0x0800,
			Type:      SegmentTypeData,
		}).
		Build()

	fmt.Printf("Address size: %d bits\n", config.MemoryLayout().AddressSize)
	fmt.Printf("Segments: %d\n", len(config.Segments()))
	fmt.Printf("Symbols: %d\n", len(config.Symbols()))
	// Output:
	// Address size: 16 bits
	// Segments: 2
	// Symbols: 2
}
