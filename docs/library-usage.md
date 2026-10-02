# Library Usage

This document covers `pkg/retroasm` and the owned stream API in `pkg/codec`.
It describes the `work2` source branch reviewed on 2026-10-02. The
[merge plan](work-branch-changes.md) defines the smaller scope for `main`.
For most users, the `retroasm` CLI is the primary interface. The library API is mainly useful when you are:

- generating assembly code programmatically
- integrating assembly into a compiler or build pipeline
- assembling source text inside another Go tool

## Current Target Coverage

Without registration, `retroasm.New()` selects 6502. A registered architecture
adapter supplies another CPU configuration. The CLI registers 6502, 65816,
68000, SM83, or Z80 through this API. The CHIP-8 CLI path uses `pkg/assembler`
directly. A system name does not supply a cartridge header or memory layout.

Select syntax through `config.Config.CompatibilityMode`. Available modes are
default, asm6, ca65, NESASM, and x816. Each mode supports a subset of the
original syntax. See the [compatibility guide](compatibility-mode-plan.md).

The core entry points are:

- `AssembleText` for source text input
- `AssembleAST` for AST-first workflows

## Installation

```bash
go get github.com/retroenv/retroasm
```

Requirements:

- Go 1.25.0 or later

## Basic Setup

Create an assembler instance and register the architecture adapter you want to assemble for.
The example below uses the current 6502 implementation:

```go
package main

import (
	"github.com/retroenv/retroasm/pkg/arch/cpu6502"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/retroasm"
	"github.com/retroenv/retrogolib/arch"
)

func newAssembler() (retroasm.Assembler, error) {
	assembler := retroasm.New()

	cpu6502Arch := cpu6502.New()
	cpu6502Arch.CompatibilityMode = config.CompatCa65
	adapter := retroasm.NewArchitectureAdapter(string(arch.CPU6502), cpu6502Arch, cpu6502Arch)
	if err := assembler.RegisterArchitecture(string(arch.CPU6502), adapter); err != nil {
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

	"github.com/retroenv/retroasm/pkg/arch/cpu6502"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/retroasm"
	"github.com/retroenv/retrogolib/arch"
)

func main() {
	assembler := retroasm.New()

	cpu6502Arch := cpu6502.New()
	cpu6502Arch.CompatibilityMode = config.CompatCa65
	adapter := retroasm.NewArchitectureAdapter(string(arch.CPU6502), cpu6502Arch, cpu6502Arch)
	if err := assembler.RegisterArchitecture(string(arch.CPU6502), adapter); err != nil {
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
- `SourceName` labels copied input symbol metadata. Use `codec.ParseStream`
  to attach a source name to stream positions and codec diagnostics.
- `Format` does not select compatibility behavior in the current dispatcher.
  Set `cpu6502Arch.CompatibilityMode = config.CompatCa65` before registration
  for ca65 syntax. Import `pkg/assembler/config` for the mode constants.
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

	"github.com/retroenv/retroasm/pkg/arch/cpu6502"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retroasm/pkg/retroasm"
	"github.com/retroenv/retrogolib/arch"
	cpu "github.com/retroenv/retrogolib/arch/cpu/cpu6502"
)

func main() {
	assembler := retroasm.New()

	cpu6502Arch := cpu6502.New()
	adapter := retroasm.NewArchitectureAdapter(string(arch.CPU6502), cpu6502Arch, cpu6502Arch)
	if err := assembler.RegisterArchitecture(string(arch.CPU6502), adapter); err != nil {
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

The current high-level dispatcher fills these fields:

| Field | Current behavior |
|---|---|
| `Binary` | Assembled bytes. The default config starts at `$8000` and emits used bytes without full-bank padding. |
| `AST` | Original input nodes for `AssembleAST`; not populated by `AssembleText`. |
| `Symbols` | Copies of input symbol metadata. These values are not injected into expression resolution. Newly resolved labels are not returned here. |
| `Segments`, `Diagnostics` | Not populated by this dispatcher. Assembly errors are returned as errors. |

For resolved labels and relocation records, use `pkg/codec`. Do not use the
high-level `Symbols` map as a linker result.

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

`SetConfiguration` stores the builder result, but the assembly dispatcher does
not apply it. `ArchitectureAdapter.CreateAssembler().AssembleAST` also does
not generate bytes. Use the top-level methods or `pkg/codec` for assembly.

### Fixed-size output

Pass a config file through `TextInput.ConfigFile` when callers need bank fill:

```text
MEMORY {
    CODE: start = $8000, size = $8000, fill = yes, fillval = $ff;
}
SEGMENTS {
    CODE: load = CODE, type = rw;
}
```

The source must select `CODE`. This requests a filled 32 KiB memory area.
For custom AST memory layouts, use the configured lower-level assembler or
codec. The high-level AST path reloads its built-in config.

## Owned streams and resolved output

Use `pkg/codec` for source positions, resolved labels, and relocation metadata.
Load a memory config before assembly; `codec.New` does not load one.

```go
cfg := cpu6502.New()
cfg.CompatibilityMode = config.CompatCa65
if err := cfg.ReadCa65Config(strings.NewReader(memoryConfig)); err != nil {
	return err
}
c, err := codec.New(cfg)
if err != nil {
	return err
}
stream, err := c.ParseStream(ctx, "input.asm", strings.NewReader(source))
if err != nil {
	return err
}
result, err := c.AssembleStream(ctx, stream)
if err != nil {
	return err
}
fmt.Printf("% X\n", result.Binary)
```

This fragment also requires `pkg/codec`. `result.Symbols` contains resolved
label addresses. `result.Stream.Relocations()` returns owned relocation
records. Assembly operates on a copy of the input stream.

Stream getters return copies. Use `RenameSymbols`, `Rewrite`, or `EditNodes`
to publish changes. Native edit `At` and `Nodes` reads are independent copies.
An accepted mutation makes earlier edit views stale. `FormatStream` returns
normalized source and rejects unsupported nodes; it does not preserve original
spacing. Relocations do not provide a complete external linker interface.

### AST migration details

- Import `pkg/arch/cpu6502` instead of `pkg/arch/m6502` on this branch.
- `ast.Data.Values` is `[]*expression.Expression`. Put each data item in a
  separate expression. Do not put a complete comma-separated list in one.
- Opcode IDs contain an architecture and a numeric value. Use codec parsing
  or `codec.BuildInstruction` to obtain typed instructions with valid IDs.
- Mutable opaque instruction arguments must implement
  `ast.InstructionArgumentCopier`. Unsupported mutable values cause a panic
  at construction or copying.
- Custom assembly adapters must implement `arch.ByteOrderer` on this branch.

## Limitations

Memory layout still uses ca65-style configuration. Supply a layout suitable
for the selected CPU. Register one architecture per assembler when possible:
with multiple registrations, the dispatcher prefers `6502` or returns an
ambiguity error. The high-level API does not select a CPU from `TextInput.Format`.

## Related References

- [README](../README.md)
- [AST example](../examples/ast-first/main.go)
- [pkg.go.dev API reference](https://pkg.go.dev/github.com/retroenv/retroasm)
