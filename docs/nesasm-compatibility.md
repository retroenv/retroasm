# NESASM Assembler Compatibility

## Overview

This reference describes NESASM syntax accepted on `work2`, reviewed on
2026-10-02. Select `-compat nesasm` or `config.CompatNesasm`.

See [Compatibility Mode Infrastructure](compatibility-mode-plan.md) for shared features.

## Key Differences from retroasm

- **Bank-based** memory model (`.bank`/`.org` pairs) instead of flat addressing
- **Macro syntax** uses `name .macro` (name before directive) instead of `.macro name`
- **Macro parameters** use `\1`-`\9` instead of named parameters
- **Local labels** prefixed with `.` (dot) instead of `@`
- **`*` as program counter** reference (shared with x816 and ca65 modes)

## Implemented Features

### Label Syntax

Standard labels require a trailing colon. Local labels use a `.` prefix and are scoped between
non-local labels. The parser disambiguates `.label` from `.directive` by checking whether the
token after `.` is a known directive name.

```asm
some_routine:
.loop:              ; local label — internally scoped as some_routine.loop
    dex
    bne .loop
another_routine:
.loop:              ; different scope — internally scoped as another_routine.loop
    dey
    bne .loop
```

### Bank-Based Memory Model

```asm
.bank 0
.org $8000          ; PRG bank 0

.bank 1
.org $A000          ; PRG bank 1

.bank 2
.org $0000          ; CHR bank 0
```

These directives are parsed, but `.bank` does not select an output bank in
the current assembler pipeline. Address assignment rejects the resulting
`ast.Bank` node. Do not use this example as a complete memory
layout. Define memory areas and segments in a ca65-style config, then select
them with `.segment`. `.org` changes the address within the configured layout.
The output writer uses configured memory areas and their fill settings.

### NESASM Macro Syntax

The macro name comes before the `.macro` keyword. Parameters are referenced positionally with
`\1` through `\9`. During expansion, backslash-number sequences are substituted with the
corresponding call arguments.

```asm
add_val .macro
    clc
    adc \1
.endm

    add_val #$10    ; expands to: clc / adc #$10
```

### Directives

#### Data and Storage

| Directive | Handler | Notes |
|---|---|---|
| `.byte` / `.db` | Data | Byte data |
| `.ds` | DataStorage | Define storage (fill bytes) |
| `.dw` / `.word` | Data | Word data |
| `.incbin` | Include | Binary include |
| `.include` | Include | Source include |
| `.org` | Base | Set PC within the configured memory layout |

#### Symbol and Variable Definition

| Directive | Handler | Notes |
|---|---|---|
| `.equ` / `=` | Alias | Symbol assignment (`name .equ value`) |
| `.rs` | Variable | Reserve symbol using offset counter |
| `.rsset` | OffsetCounter | Set RS counter base address |

#### iNES Header

These handlers create configuration nodes. The output writer does not generate
an iNES header from them. Supply header bytes in a configured header segment.

| Directive | Handler | Notes |
|---|---|---|
| `.inesbat` | NesasmConfig | Battery flag |
| `.ineschr` | NesasmConfig | CHR ROM banks |
| `.inesmap` | NesasmConfig | Mapper number |
| `.inesmir` | NesasmConfig | Mirroring mode |
| `.inesprg` | NesasmConfig | PRG ROM banks |
| `.inessubmap` | NesasmConfig | Submapper number |

#### Structure

| Directive | Handler | Notes |
|---|---|---|
| `.bank` | Bank | Parsed only; assembly does not implement bank selection |
| `.endp` | EndProc | End procedure |
| `.macro` / `.endm` | Macro | Macro definition (NESASM syntax) |
| `.proc` | Proc | Procedure definition |

#### Conditionals

| Directive | Handler | Notes |
|---|---|---|
| `.else` | Else | Conditional else |
| `.endif` | Endif | End conditional |
| `.if` | If | Conditional assembly |
| `.ifdef` | Ifdef | Symbol defined check |
| `.ifndef` | Ifndef | Symbol not defined check |

#### Error Handling

| Directive | Handler | Notes |
|---|---|---|
| `.fail` | Error | Trigger assembly error |

#### No-Op Directives

These directives are accepted and silently ignored to avoid parse errors in NESASM sources.

| Directive | Notes |
|---|---|
| `.bss` | Section switching stub |
| `.code` | Section switching stub |
| `.data` | Section switching stub |
| `.list` | Listing control |
| `.mlist` | Macro listing control |
| `.nolist` | Listing control |
| `.nomlist` | Macro listing control |
| `.opt` | Assembler options |
| `.zp` | Section switching stub |

## Not Implemented

The following NESASM features are not currently supported.

| Feature | Notes |
|---|---|
| `@` octal number format | Conflicts with `@` local labels in other modes |
| `BANK()` function operator | Returns bank number of a label |
| `HIGH()` function operator | Use `>expr` instead |
| `LOW()` function operator | Use `<expr` instead |
| `PAGE()` function operator | Page number (addr >> 8) |
| `SIZEOF()` function operator | Size of structure |
| `.defchr` | Define 8x8 character tile inline (16 bytes CHR data) |
| `.func` | Function-style macro |
| `.incchr` | Include and convert image to CHR format |
| `.pcm` | Include PCM audio data |
| `.procgroup` / `.endprocgroup` | Procedure group |

## Notes

- Configure each output bank explicitly. Memory areas can share CPU addresses
  while retaining separate output bytes.
- The `name .macro` syntax (name before keyword) is unique among 6502 assemblers and requires
  special parsing in the identifier handler.
- `pkg/parser/parser_nesasm_test.go` checks local labels and macro definitions.
  It does not verify emitted bytes from positional macro expansion. The
  [merge plan](work-branch-changes.md) requires that coverage in P09.
- No code tests were run for this documentation review.
