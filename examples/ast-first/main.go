package main

import (
	"context"
	"fmt"
	"log"

	"github.com/retroenv/retroasm/arch/m6502"
	"github.com/retroenv/retroasm/parser/ast"
	"github.com/retroenv/retroasm/pkg/retroasm"
	cpu "github.com/retroenv/retrogolib/arch/cpu/m6502"
)

func main() {
	// Demonstrate AST-first assembly
	err := astFirstExample()
	if err != nil {
		log.Fatal(err)
	}
}

func astFirstExample() error {
	fmt.Println("Retroasm AST-First Library Example")
	fmt.Println("==================================")

	// Create and configure assembler
	assembler, err := setupAssembler()
	if err != nil {
		return fmt.Errorf("setting up assembler: %w", err)
	}

	// Create AST directly
	program := createSampleProgram()
	fmt.Printf("Generated AST with %d nodes\n", len(program))

	// Assemble and display results
	return assembleAndDisplay(assembler, program)
}

func setupAssembler() (retroasm.Assembler, error) {
	assembler := retroasm.New()

	// Register 6502 architecture
	m6502Arch := m6502.New()
	adapter := retroasm.NewArchitectureAdapter("6502", m6502Arch, m6502Arch)
	err := assembler.RegisterArchitecture("6502", adapter)
	if err != nil {
		return nil, fmt.Errorf("registering architecture: %w", err)
	}

	// Configure assembler
	config := retroasm.NewConfigurationBuilder().
		SetSymbol("RESET_VECTOR", 0xFFFC).
		SetSymbol("START_ADDR", 0x8000).
		AddSegment(retroasm.SegmentConfig{
			Name:      "CODE",
			StartAddr: 0x8000,
			Size:      0x8000,
			Type:      retroasm.SegmentTypeCode,
		}).
		Build()

	err = assembler.SetConfiguration(config)
	if err != nil {
		return nil, fmt.Errorf("setting configuration: %w", err)
	}

	return assembler, nil
}

func assembleAndDisplay(assembler retroasm.Assembler, program []ast.Node) error {
	// Assemble using AST interface
	input := &retroasm.ASTInput{
		AST:        program,
		SourceName: "generated.asm",
		BaseAddr:   0x8000,
	}

	output, err := assembler.AssembleAST(context.Background(), input)
	if err != nil {
		return fmt.Errorf("assembling AST: %w", err)
	}

	// Display results
	fmt.Printf("Assembly successful!\n")
	fmt.Printf("Generated %d bytes of binary code\n", len(output.Binary))
	fmt.Printf("Binary: %X\n", output.Binary)
	fmt.Printf("Defined %d symbols:\n", len(output.Symbols))

	for name, symbol := range output.Symbols {
		fmt.Printf("  %s = 0x%04X (type: %v)\n", name, symbol.Value, getSymbolTypeName(symbol.Type))
	}

	if len(output.Diagnostics) > 0 {
		fmt.Printf("Diagnostics:\n")
		for _, diag := range output.Diagnostics {
			fmt.Printf("  %s: %s\n", getDiagnosticLevelName(diag.Level), diag.Message)
		}
	}

	return nil
}

// createSampleProgram demonstrates how to generate AST nodes programmatically.
func createSampleProgram() []ast.Node {
	var nodes []ast.Node

	// Example: Simple sprite positioning program
	// Equivalent to:
	//   LDA #50      ; X position
	//   STA SPRITE_X
	//   LDA #100     ; Y position
	//   STA SPRITE_Y
	//   RTS

	// Load X position (immediate mode: LDA #50)
	nodes = append(nodes, ast.NewInstruction("LDA", int(cpu.ImmediateAddressing), ast.NewNumber(50), nil))
	// Store to absolute address (STA $0200)
	nodes = append(nodes, ast.NewInstruction("STA", int(cpu.AbsoluteAddressing), ast.NewNumber(0x0200), nil))

	// Load Y position (immediate mode: LDA #100)
	nodes = append(nodes, ast.NewInstruction("LDA", int(cpu.ImmediateAddressing), ast.NewNumber(100), nil))
	// Store to absolute address (STA $0201)
	nodes = append(nodes, ast.NewInstruction("STA", int(cpu.AbsoluteAddressing), ast.NewNumber(0x0201), nil))

	// Return (implied addressing)
	nodes = append(nodes, ast.NewInstruction("RTS", int(cpu.ImpliedAddressing), nil, nil))

	return nodes
}

func getSymbolTypeName(symbolType retroasm.SymbolType) string {
	switch symbolType {
	case retroasm.SymbolTypeLabel:
		return "Label"
	case retroasm.SymbolTypeConstant:
		return "Constant"
	case retroasm.SymbolTypeVariable:
		return "Variable"
	default:
		return "Unknown"
	}
}

func getDiagnosticLevelName(level retroasm.DiagnosticLevel) string {
	switch level {
	case retroasm.DiagnosticError:
		return "Error"
	case retroasm.DiagnosticWarning:
		return "Warning"
	case retroasm.DiagnosticInfo:
		return "Info"
	default:
		return "Unknown"
	}
}
