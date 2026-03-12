# NESASM Assembler Compatibility

## Overview

NESASM (based on MagicKit/PCEas by David Michel) is a 6502 assembler designed for NES
development. It uses a bank-based memory model and has unique syntax for macros, local labels,
and character data.

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

The assembler tracks the current bank number and bank size. Each `.bank` directive advances the
output position to `bank_number * bank_size`. `.org` sets the PC within the current bank.

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
| `.org` | Base | Set PC (within bank) |

#### Symbol and Variable Definition

| Directive | Handler | Notes |
|---|---|---|
| `.equ` / `=` | Alias | Symbol assignment (`name .equ value`) |
| `.rs` | Variable | Reserve symbol using offset counter |
| `.rsset` | OffsetCounter | Set RS counter base address |

#### iNES Header

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
| `.bank` | Bank | Select ROM bank |
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

- NESASM's bank model is tightly coupled to NES mapper hardware. The assembler needs to know the
  bank size (typically 8KB or 16KB for PRG, 8KB for CHR).
- The `name .macro` syntax (name before keyword) is unique among 6502 assemblers and requires
  special parsing in the identifier handler.
- Many NESASM projects use a simple flat structure with sequential `.bank`/`.org` pairs, making
  compatibility straightforward for common cases.
