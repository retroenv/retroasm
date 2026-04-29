# Library Usage

This document covers the embeddable `pkg/retroasm` API.
For most users, the `retroasm` CLI is the primary interface. The library API is mainly useful when you are:

- generating assembly code programmatically
- integrating assembly into a compiler or build pipeline
- assembling source text inside another Go tool

## Current Target Coverage

The public library API is designed around a reusable assembler interface with pluggable architectures.
At the moment, the implemented production path matches the current CLI target:

- system: NES
- CPU: 6502
- text formats: `asm6`, `ca65`, `nesasm`

The core entry points are:

- `AssembleText` for source text input
- `AssembleAST` for AST-first workflows

## Installation

```bash
go get github.com/retroenv/retroasm
```

Requirements:

- Go 1.24 or later

## Basic Setup

Create an assembler instance and register the architecture adapter you want to assemble for.
The example below uses the current 6502 implementation:

```go
package main

import (
	"github.com/retroenv/retroasm/pkg/arch/m6502"
	"github.com/retroenv/retroasm/pkg/retroasm"
	"github.com/retroenv/retrogolib/arch"
)

func newAssembler() (retroasm.Assembler, error) {
	assembler := retroasm.New()

	m6502Arch := m6502.New()
	adapter := retroasm.NewArchitectureAdapter(string(arch.M6502), m6502Arch, m6502Arch)
	if err := assembler.RegisterArchitecture(string(arch.M6502), adapter); err != nil {
		return nil, err
	}

	return assembler, nil
}
```

## Assemble Source Text

Use `AssembleText` when you already have assembly source as text.
The example below shows the current 6502/NES-oriented path:

```go
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch/m6502"
	"github.com/retroenv/retroasm/pkg/retroasm"
	"github.com/retroenv/retrogolib/arch"
)

func main() {
	assembler := retroasm.New()

	m6502Arch := m6502.New()
	adapter := retroasm.NewArchitectureAdapter(string(arch.M6502), m6502Arch, m6502Arch)
	if err := assembler.RegisterArchitecture(string(arch.M6502), adapter); err != nil {
		panic(err)
	}

	output, err := assembler.AssembleText(context.Background(), &retroasm.TextInput{
		Source:     strings.NewReader(".segment \"CODE\"\nLDA #$01\nSTA $0200\n"),
		SourceName: "example.asm",
		Format:     retroasm.FormatCa65,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("% X\n", output.Binary)
}
```

Notes:

- `Source` is required.
- `SourceName` is used in diagnostics and symbol metadata.
- `Format` should be one of `retroasm.FormatAsm6`, `retroasm.FormatCa65`, or `retroasm.FormatNesasm`.
- If `ConfigFile` is empty, retroasm uses its built-in default ca65-style memory configuration for the current implementation.

### Using a ca65 Config File

If your source depends on a custom memory map, pass a config file path:

```go
output, err := assembler.AssembleText(context.Background(), &retroasm.TextInput{
	Source:     strings.NewReader(source),
	SourceName: "game.asm",
	Format:     retroasm.FormatCa65,
	ConfigFile: "memory.cfg",
})
```

## Assemble from AST

Use `AssembleAST` when another part of your program already produces assembly nodes directly.
The example below again uses the current 6502 path:

```go
package main

import (
	"context"
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch/m6502"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retroasm/pkg/retroasm"
	"github.com/retroenv/retrogolib/arch"
	cpu "github.com/retroenv/retrogolib/arch/cpu/m6502"
)

func main() {
	assembler := retroasm.New()

	m6502Arch := m6502.New()
	adapter := retroasm.NewArchitectureAdapter(string(arch.M6502), m6502Arch, m6502Arch)
	if err := assembler.RegisterArchitecture(string(arch.M6502), adapter); err != nil {
		panic(err)
	}

	program := []ast.Node{
		ast.NewInstruction("LDA", int(cpu.ImmediateAddressing), ast.NewNumber(1), nil),
		ast.NewInstruction("STA", int(cpu.AbsoluteAddressing), ast.NewNumber(0x0200), nil),
	}

	output, err := assembler.AssembleAST(context.Background(), &retroasm.ASTInput{
		AST:        program,
		SourceName: "generated.asm",
		BaseAddr:   0x8000,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("% X\n", output.Binary)
}
```

Notes:

- If the AST does not begin with a segment node, retroasm prepends `CODE` automatically.
- `BaseAddr` overrides the built-in default start address for AST assembly.
- See [examples/ast-first](../examples/ast-first) for a larger AST-first example.

## Output Structure

Both `AssembleText` and `AssembleAST` return `*retroasm.AssemblyOutput`.

Relevant fields:

- `Binary`: assembled machine code
- `AST`: AST nodes returned by the assembly flow
- `Symbols`: symbol metadata keyed by symbol name
- `Segments`: segment output metadata
- `Diagnostics`: warnings or informational diagnostics

Example:

```go
output, err := assembler.AssembleText(ctx, input)
if err != nil {
	return err
}

fmt.Printf("bytes: %d\n", len(output.Binary))
for name, symbol := range output.Symbols {
	fmt.Printf("%s = $%X\n", name, symbol.Value)
}
```

## Configuration API

The package also exposes a configuration builder:

```go
config := retroasm.NewConfigurationBuilder().
	SetMemoryLayout(retroasm.MemoryLayout{
		AddressSize: 16,
		Endianness:  retroasm.LittleEndian,
	}).
	SetSymbol("RESET_VECTOR", 0xFFFC).
	AddSegment(retroasm.SegmentConfig{
		Name:      "CODE",
		StartAddr: 0x8000,
		Size:      0x8000,
		Type:      retroasm.SegmentTypeCode,
	}).
	Build()
```

At the moment, the active implementation primarily consumes:

- `TextInput.ConfigFile` for text-based custom memory layout
- `ASTInput.BaseAddr` for AST-based base address control
- `ASTInput.Symbols` and `TextInput.Symbols` for symbol metadata passed into the output

So while `SetConfiguration` exists on the public interface, callers should currently treat it as a broader API surface than the main configuration mechanism used by the implemented target path today.

## Limitations

The current library surface is intentionally narrow:

- only the current NES/6502 path is implemented as a production target
- text assembly still depends on the existing assembler pipeline and ca65-style configuration model
- library consumers should expect the AST-first path to be the more specialized integration mode

## Related References

- [README](../README.md)
- [AST example](../examples/ast-first/main.go)
- [pkg.go.dev API reference](https://pkg.go.dev/github.com/retroenv/retroasm)
