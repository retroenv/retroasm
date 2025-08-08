# retroasm - an Assembler for retro computer systems

[![Build status](https://github.com/retroenv/retroasm/actions/workflows/go.yaml/badge.svg?branch=main)](https://github.com/retroenv/retroasm/actions)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/retroenv/retroasm)
[![Go Report Card](https://goreportcard.com/badge/github.com/retroenv/retroasm)](https://goreportcard.com/report/github.com/retroenv/retroasm)
[![codecov](https://codecov.io/gh/retroenv/retroasm/branch/main/graph/badge.svg?token=NS5UY28V3A)](https://codecov.io/gh/retroenv/retroasm)


retroasm allows you to assemble programs retro computer systems.

## Features

The project is at an early stage of development and currently supports:

* Supports 6502 CPU and Nintendo Entertainment System (NES) as target
* Can process [asm6](https://github.com/freem/asm6f)/[ca65](https://github.com/cc65/cc65)/[nesasm](https://github.com/ClusterM/nesasm)
  compatible assembly files
* Supports undocumented 6502 CPU opcodes

## TODO

* Support includes
* More testing of asm6 support
* More testing of nesasm support
* Test with larger projects

## Installation

The tool uses a modern software stack that does not have any system dependencies beside requiring a somewhat modern
operating system to run:

* Linux: 2.6.32+
* Windows: 10+
* macOS: 10.15 Catalina+

There are 2 options to install retroasm:

1. Download and unpack a binary release from [Releases](https://github.com/retroenv/retroasm/releases)

or

2. Compile the latest release from source:

```
go install github.com/retroenv/retroasm/cmd/retroasm@latest
```

## Usage

Assembler a ROM using a ca65 configuration:

```
retroasm -c test.cfg -o test.nes test.asm
```


## Options

```
usage: retroasm [options] <file to assemble>

  -c string
    	assembler config file
  -cpu string
    	target CPU architecture (6502) (default "6502")
  -debug
    	enable debug logging
  -o string
    	name of the output file
  -q	perform operations quietly
  -system string
    	target system (nes) (default "nes")
```
