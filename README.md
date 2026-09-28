# retroasm

[![CI](https://github.com/retroenv/retroasm/actions/workflows/go.yaml/badge.svg?branch=main)](https://github.com/retroenv/retroasm/actions/workflows/go.yaml)
[![Codecov](https://codecov.io/gh/retroenv/retroasm/branch/main/graph/badge.svg?token=NS5UY28V3A)](https://codecov.io/gh/retroenv/retroasm)
[![Release](https://img.shields.io/github/v/release/retroenv/retroasm)](https://github.com/retroenv/retroasm/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/retroenv/retroasm.svg)](https://pkg.go.dev/github.com/retroenv/retroasm)
[![License](https://img.shields.io/github/license/retroenv/retroasm)](LICENSE)
![LLM assisted: human reviewed](https://img.shields.io/badge/LLM%20assisted-human%20reviewed-6f42c1)

An assembler for retro systems. It converts assembly source or an abstract
syntax tree (AST) to machine code and ROM files.

## Features

* **Multiple CPUs** - Assembles source for 6502, 65816, CHIP-8, 68000, SM83, and Z80
* **Assembler modes** - Accepts supported asm6, ca65, NESASM, and x816 syntax
* **Library API** - Assembles source text or an AST from Go programs
* **Configuration files** - Uses ca65-style files to define memory and segments
* **Macros and conditionals** - Processes macros and conditional assembly directives
* **Expressions** - Evaluates arithmetic and bitwise expressions in assembly source
* **Z80 profiles** - Selects the full, strict documented, or Game Boy instruction set

## Supported Systems

| System | CPU | Status |
|--------|-----|--------|
| NES | 6502 | Supported |
| SNES | 65816 | Supported |
| CHIP-8 | CHIP-8 VM | Supported |
| Game Boy | SM83 or Z80 Game Boy subset | Supported |
| ZX Spectrum | Z80 | Supported |
| Generic binary output | 6502, 65816, 68000, SM83, or Z80 | Supported |

Intel 8086 support is in development.

## Quick Start

### Installation

Download a binary from [Releases](https://github.com/retroenv/retroasm/releases),
or install from source with Go 1.25 or newer:

```bash
go install github.com/retroenv/retroasm/cmd/retroasm@latest
```

### Basic Usage

Assemble a source file. Use `-o` to set the output file:

```bash
retroasm -cpu 6502 -system nes -o game.nes program.asm
retroasm -cpu chip8 -o game.ch8 program.asm
retroasm -cpu 65816 -system snes -o game.sfc program.asm
retroasm -cpu cpu68000 -system generic -o program.bin program.asm
retroasm -cpu sm83 -system gameboy -o game.gb program.asm
retroasm -cpu z80 -system zx-spectrum -o program.bin program.asm
```

Select an assembler mode for existing 6502 source:

```bash
retroasm -compat ca65 -o game.nes program.asm
```

Use a ca65-style configuration file to set the memory layout:

```bash
retroasm -c memory.cfg -o game.nes program.asm
```

Select the Game Boy Z80 subset:

```bash
retroasm -cpu z80 -system gameboy -z80-profile gameboy-z80-subset -o game.gb program.asm
```

See the [library guide](docs/library-usage.md) for Go integration and the
[compatibility guides](docs/compatibility-mode-plan.md) for supported assembler
syntax. Run `retroasm -h` to see all command options.
