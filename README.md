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

**Requirements:**
- Go 1.24 or later

## Overview

Retroasm is a Go-based assembler for retro computer systems and CPU architectures.
It can be used both as a command-line assembler and as a library for tools that need to generate machine code directly
from source text or from an already-built AST.

### Key Design Principles
- **AST-first architecture**: Assemble either parsed source text or generated AST nodes
- **Multi-format parsing**: Supports asm6, ca65, and nesasm-style source input
- **Embeddable API**: Designed to plug into compilers, code generators, and build tooling
- **Configurable output**: Supports ca65-style configuration for memory layout and segments
- **Modern Go implementation**: Clear package boundaries and comprehensive test coverage

## Supported Targets

### Current Support
- **NES / 6502**: End-to-end support for ROM-oriented assembly output in the current CLI and library workflow

### Source Formats
- **asm6**: asm6 and asm6f-style syntax
- **ca65**: cc65 toolchain syntax with optional config file support
- **nesasm**: NESasm3-style syntax

## Features

### Command-Line Assembly
- Assemble source files for the currently supported target
- Select target system and CPU through CLI flags
- Enable quiet or debug logging for build integration and troubleshooting

### Library API
- Embed the assembler in Go tooling when CLI usage is not enough
- Assemble directly from text input or generated AST nodes
- See [docs/library-usage.md](docs/library-usage.md) for the detailed integration guide

## Package Overview

    ├─ cmd/retroasm      command-line assembler entry point
    ├─ docs              reference material and deeper guides
    ├─ examples          runnable and integration examples
    ├─ pkg/arch          architecture implementations
    ├─ pkg/assembler     core assembly pipeline
    ├─ pkg/expression    expression model and helpers
    ├─ pkg/lexer         tokenization for supported source formats
    ├─ pkg/number        numeric parsing helpers
    ├─ pkg/parser        source parsing and AST generation
    ├─ pkg/retroasm      public library API
    ├─ pkg/scope         symbol scope management

## Quick Start

### CLI Usage

Assemble a source file for the current NES/6502 target:

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

Use a ca65-compatible config file:

```bash
retroasm -c memory.cfg -o game.nes main.asm
```

Show command usage:

```text
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
        name of the output file
  -q    perform operations quietly
  -system string
        target system (chip8, gameboy, generic, nes, snes, zx-spectrum)
  -z80-profile string
        Z80 instruction profile (default, strict-documented, gameboy-z80-subset)
```

## License

This project is licensed under the Apache License Version 2.0 - see the [LICENSE](LICENSE) file for details.
