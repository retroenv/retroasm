# retroasm - an assembler for retro computer systems

[![Build status](https://github.com/retroenv/retroasm/actions/workflows/go.yaml/badge.svg?branch=main)](https://github.com/retroenv/retroasm/actions)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/retroenv/retroasm)
[![Go Report Card](https://goreportcard.com/badge/github.com/retroenv/retroasm)](https://goreportcard.com/report/github.com/retroenv/retroasm)
[![codecov](https://codecov.io/gh/retroenv/retroasm/branch/main/graph/badge.svg?token=NS5UY28V3A)](https://codecov.io/gh/retroenv/retroasm)

retroasm is a modern assembler for retro computer systems that compiles assembly language into machine code for classic hardware platforms.

## Features

* **Multi-Format Support** - Compatible with asm6, ca65, and nesasm assembly syntax
* **Library API** - Use as a Go library for compiler integration and code generation
* **AST-based Assembly** - Direct AST input for programmatic assembly
* **Configuration Files** - ca65-style configuration for custom memory layouts
* **Modern Implementation** - Fast, reliable Go codebase with comprehensive tests

## Supported Systems

| System | Architecture | Assemblers | Status |
|--------|-------------|------------|--------|
| **NES** | 6502 | asm6, ca65, nesasm | Stable |

## Quick Start

### Installation

**Option 1:** Download a binary from [Releases](https://github.com/retroenv/retroasm/releases)

**Option 2:** Install from source:
```bash
go install github.com/retroenv/retroasm/cmd/retroasm@latest
```

### Basic Usage

Assemble a program:
```bash
retroasm -o game.nes program.asm
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
  -cpu string
        target CPU architecture: 6502 (default "6502")
  -debug
        enable debug logging with detailed output
  -o string
        output ROM file name (required)
  -q    perform operations quietly (minimal output)
  -system string
        target system: nes (default "nes")
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
