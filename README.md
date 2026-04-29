# retroasm - an assembler for retro computer systems

[![Build status](https://github.com/retroenv/retroasm/actions/workflows/go.yaml/badge.svg?branch=main)](https://github.com/retroenv/retroasm/actions)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/retroenv/retroasm)
[![Go Report Card](https://goreportcard.com/badge/github.com/retroenv/retroasm)](https://goreportcard.com/report/github.com/retroenv/retroasm)
[![codecov](https://codecov.io/gh/retroenv/retroasm/branch/main/graph/badge.svg?token=NS5UY28V3A)](https://codecov.io/gh/retroenv/retroasm)

## Installation

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
retroasm -o game.nes program.asm
```

Use a ca65-compatible config file:

```bash
retroasm -c memory.cfg -o game.nes main.asm
```

Show command usage:

```text
usage: retroasm [options] <file to assemble>

  -c string
        assembler config file
  -cpu string
        target CPU architecture (6502) (default "6502")
  -debug
        enable debug logging
  -o string
        name of the output file
  -q    perform operations quietly
  -system string
        target system (nes) (default "nes")
```

## License

This project is licensed under the Apache License Version 2.0 - see the [LICENSE](LICENSE) file for details.
