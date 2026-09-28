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

* **6502 assembly** - Assembles source for NES output
* **Library API** - Assembles source text or an AST from Go programs
* **Configuration files** - Uses ca65-style files to define memory and segments
* **Macros and conditionals** - Processes macros and conditional assembly directives
* **Expressions** - Evaluates arithmetic and bitwise expressions in assembly source

## Supported Systems

| System | CPU | Status |
|--------|-----|--------|
| NES | 6502 | Supported |

## Quick Start

### Installation

Download a binary from [Releases](https://github.com/retroenv/retroasm/releases),
or install from source with Go 1.25 or newer:

```bash
go install github.com/retroenv/retroasm/cmd/retroasm@latest
```

### Basic Usage

Assemble a 6502 source file for NES. Use `-o` to set the output file:

```bash
retroasm -o game.nes program.asm
```

Use a ca65-style configuration file to set the memory layout:

```bash
retroasm -c memory.cfg -o game.nes program.asm
```

See the [library guide](docs/library-usage.md) for Go integration. Run
`retroasm -h` to see all command options.
