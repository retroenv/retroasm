// Package retroasm provides a library for assembling retro computer assembly code.
//
// The assembler converts assembly source directly to machine code (no separate
// linking phase). It supports the 6502 processor used in systems like the NES,
// Commodore 64, and Apple II.
//
// # Usage
//
// The library provides two input methods:
//
//  1. AST input - pass AST nodes directly, useful for code generators
//  2. Text input - parse assembly source text
//
// Basic example:
//
//	assembler := retroasm.New()
//
//	// Register 6502 architecture
//	m6502Arch := m6502.New()
//	adapter := retroasm.NewArchitectureAdapter("6502", m6502Arch, m6502Arch)
//	assembler.RegisterArchitecture("6502", adapter)
//
//	// Assemble from AST
//	program := []ast.Node{
//		ast.NewInstruction("LDA", int(cpu.ImmediateAddressing), ast.NewNumber(1), nil),
//		ast.NewInstruction("STA", int(cpu.AbsoluteAddressing), ast.NewNumber(0x0200), nil),
//	}
//
//	input := &retroasm.ASTInput{
//		AST:        program,
//		SourceName: "example.asm",
//	}
//
//	output, err := assembler.AssembleAST(context.Background(), input)
//
// # Assembly Formats
//
// Text input supports multiple assembly formats:
//   - asm6: asm6/asm6f syntax
//   - ca65: cc65 toolchain syntax
//   - nesasm: NESasm3 syntax
//
// # Examples
//
// See the Example functions and the examples/ directory.
package retroasm
