# retroasm - Modern Assembler for Retro Computer Systems

[![Build status](https://github.com/retroenv/retroasm/actions/workflows/go.yaml/badge.svg?branch=main)](https://github.com/retroenv/retroasm/actions)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/retroenv/retroasm)
[![Go Report Card](https://goreportcard.com/badge/github.com/retroenv/retroasm)](https://goreportcard.com/report/github.com/retroenv/retroasm)
[![codecov](https://codecov.io/gh/retroenv/retroasm/branch/main/graph/badge.svg?token=NS5UY28V3A)](https://codecov.io/gh/retroenv/retroasm)

retroasm is a modern, high-performance assembler for retro computer systems, designed to compile assembly language into machine code for classic hardware platforms. It provides both a command-line interface and a Go library API, making it ideal for both standalone development and integration into larger toolchains.

## Key Features

### Architecture Support
* **6502 CPU** - Full support including undocumented opcodes
* **Nintendo Entertainment System (NES)** - Native target platform with proper memory mapping
* **Extensible Architecture** - Framework designed for adding Z80 and other retro CPUs

### Assembler Compatibility
* **[asm6](https://github.com/freem/asm6f)** - Compatible with asm6/asm6f assembly syntax
* **[ca65](https://github.com/cc65/cc65)** - Supports ca65 configuration files and directives
* **[nesasm](https://github.com/ClusterM/nesasm)** - Process nesasm-compatible assembly files
* **Multi-format** - Single tool for multiple assembly dialects

### Advanced Capabilities
* **Library API** - Use as a Go library for compiler integration
* **AST-based Assembly** - Direct AST input for programmatic code generation
* **Configuration Files** - ca65-style configuration for memory layouts
* **Modern Go Implementation** - Fast, reliable, and maintainable codebase

## Quick Start

### Installation

**Option 1: Download Pre-built Binary**
Download from [Releases](https://github.com/retroenv/retroasm/releases) and extract to your PATH.

**Option 2: Install from Source**
```bash
go install github.com/retroenv/retroasm/cmd/retroasm@latest
```

**System Requirements:**
- Linux: 2.6.32+
- Windows: 10+
- macOS: 10.15 Catalina+

### Basic Usage

**Simple Assembly:**
```bash
# Assemble a basic 6502 program
retroasm -o game.nes program.asm
```

**With Configuration:**
```bash
# Use ca65-style configuration file
retroasm -c memory.cfg -o game.nes main.asm
```

### Command Line Options

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

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
