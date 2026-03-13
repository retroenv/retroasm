# retroasm - an assembler for retro computer systems

[![Build status](https://github.com/retroenv/retroasm/actions/workflows/go.yaml/badge.svg?branch=main)](https://github.com/retroenv/retroasm/actions)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/retroenv/retroasm)
[![Go Report Card](https://goreportcard.com/badge/github.com/retroenv/retroasm)](https://goreportcard.com/report/github.com/retroenv/retroasm)
[![codecov](https://codecov.io/gh/retroenv/retroasm/branch/main/graph/badge.svg?token=NS5UY28V3A)](https://codecov.io/gh/retroenv/retroasm)

retroasm is a modern, multi-architecture assembler for retro computer systems that compiles assembly language into machine code for classic hardware platforms.

## Features

### Architecture Support

| CPU | Systems | Status |
|-----|---------|--------|
| **6502** | NES | Stable |
| **65816** | SNES | Stable |
| **Chip-8** | Chip-8 VM | Stable |
| **Intel 8086** | IBM PC | In development |
| **Motorola 68000** | Amiga, Mega Drive | Stable |
| **SM83** | Game Boy | Stable |
| **Z80** | ZX Spectrum | Stable |

### Assembler Compatibility Modes

retroasm supports multiple legacy assembler syntaxes via the `-compat` flag, allowing direct assembly of source files written for popular retro assemblers:

| Mode | Assembler | Key Features |
|------|-----------|--------------|
| **asm6** | asm6 / asm6f | Colon-optional labels, `+`/`-` anonymous labels, `@` local label scoping, NES 2.0 header directives, source file inclusion |
| **ca65** | cc65 toolchain | `@` local label scoping, `:` unnamed labels with `:+`/`:-` references, `.scope`/`.endscope` blocks, `.asciiz`, `.faraddr`, `.bankbytes`, `.warning`, `.endmacro` |
| **nesasm** | NESASM (MagicKit) | Dot-prefixed local labels (`.label`), `name .macro` syntax with `\1`-`\9` positional parameters, `*` as PC, `.fail`, `.ds` storage |
| **x816** | x816 v1.12f | Colon-optional labels, `+`/`-` anonymous labels, `*` as PC, `SHL`/`SHR`/`AND`/`OR`/`XOR` expression operators, `.comment`/`.end` blocks, `.equ` aliases, 3-byte/4-byte data directives |

### Development Features
* **AST-based Assembly** - Direct AST input for programmatic assembly
* **Conditionals** - `.if`/`.else`/`.endif`, `.ifdef`/`.ifndef` conditional assembly
* **Configuration Files** - ca65-style configuration for custom memory layouts
* **Expressions** - Full expression evaluator with arithmetic, bitwise, shift, and comparison operators
* **Library API** - Use as a Go library for compiler integration and code generation
* **Macros** - Standard `.macro`/`.endm` definitions with named or positional parameters
* **Modern Implementation** - Fast, reliable Go codebase with comprehensive tests
* **Z80 Profiles** - Instruction set filtering (full, strict-documented, Game Boy subset)

## Quick Start

### Installation

**Option 1:** Download a binary from [Releases](https://github.com/retroenv/retroasm/releases)

**Option 2:** Install from source:
```bash
go install github.com/retroenv/retroasm/cmd/retroasm@latest
```

### Basic Usage

```bash
# Assemble a 6502 program for NES
retroasm -cpu 6502 -system nes -o game.nes program.asm

# Assemble using a specific compatibility mode
retroasm -compat ca65 -o game.nes program.asm

# Assemble a Chip-8 program
retroasm -cpu chip8 -o game.ch8 program.asm

# Assemble a 65816 program for SNES
retroasm -cpu 65816 -system snes -o game.sfc program.asm

# Assemble a Motorola 68000 program
retroasm -cpu m68000 -system generic -o program.bin program.asm

# Assemble an SM83 program for Game Boy
retroasm -cpu sm83 -system gameboy -o game.gb program.asm

# Assemble a Z80 program for ZX Spectrum
retroasm -cpu z80 -system zx-spectrum -o program.bin program.asm

# Assemble Z80 with Game Boy Z80 subset (alternative to SM83)
retroasm -cpu z80 -system gameboy -z80-profile gameboy-z80-subset -o game.gb program.asm
```

With ca65-style configuration:
```bash
retroasm -c memory.cfg -o game.nes main.asm
```

### Command-Line Options

```
usage: retroasm [options] <file to assemble>

  -c string
        assembler config file (ca65 compatible)
  -compat string
        assembler compatibility mode (asm6, ca65, default, nesasm, x816)
  -cpu string
        target CPU architecture (6502, 65816, chip8, m68000, sm83, z80)
  -debug
        enable debug logging with detailed output
  -m string
        assembler compatibility mode (shorthand for -compat)
  -o string
        output ROM file name (required)
  -q    perform operations quietly (minimal output)
  -system string
        target system (chip8, gameboy, generic, nes, snes, zx-spectrum)
  -z80-profile string
        Z80 instruction profile (default, strict-documented, gameboy-z80-subset)
```

## Library Usage

retroasm can be used as a Go library for integrating assembly into compilers and code generators:

```go
import "github.com/retroenv/retroasm/pkg/retroasm"

assembler := retroasm.New()

// Assemble from text
output, err := assembler.AssembleText(ctx, input)

// Or assemble from AST for code generation
output, err := assembler.AssembleAST(ctx, astInput)
```

See [examples/](examples/) for complete examples.

## System Requirements

* **Linux:** 2.6.32+
* **Windows:** 10+
* **macOS:** 10.15 Catalina+

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
