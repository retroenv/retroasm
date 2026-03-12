# NESASM Assembler Compatibility

## Overview

NESASM (based on MagicKit/PCEas by David Michel) is a 6502 assembler specifically designed for NES development. It uses a bank-based memory model and has unique syntax for macros, local labels, and character data.

See [Compatibility Mode Infrastructure](compatibility-mode-plan.md) for shared features.

## Key Differences from retroasm

- **Bank-based** memory model (`.bank`/`.org` pairs) instead of flat addressing
- **Macro syntax** uses `name .macro` (name before directive) instead of `.macro name`
- **Macro parameters** use `\1`-`\9` instead of named parameters
- **Local labels** prefixed with `.` (dot) instead of `@`
- **Function-style** operators: `HIGH()`, `LOW()`, `BANK()`
- **iNES header** integration via dedicated directives

## Label Syntax

### Standard Labels

```asm
label:          ; standard label with colon (required)
```

NESASM requires the trailing colon on labels.

### Local Labels

```asm
some_routine:
.loop:              ; local label (dot prefix)
    dex
    bne .loop
another_routine:
.loop:              ; different .loop, scoped to another_routine
    dey
    bne .loop
```

- Prefixed with `.` (dot)
- Scoped between non-local labels
- Names can be reused across different scopes

**Implementation:** Similar to `@` local labels but using `.` prefix. Need to distinguish `.label` from `.directive` — a local label reference in operand position vs. a directive at line start.

## Bank-Based Memory Model

```asm
.bank 0
.org $8000
    ; PRG bank 0 code

.bank 1
.org $A000
    ; PRG bank 1 code

.bank 2
.org $0000
    ; CHR bank 0 data
```

- `.bank N` selects a ROM bank
- `.org` sets the address within that bank
- Bank size is determined by mapper configuration
- Banks are laid out sequentially in the output

**Implementation approach:** Track current bank number and bank size. Each `.bank` directive advances the output position to `bank_number * bank_size`. `.org` sets the PC within the current bank.

**Priority:** High — this is fundamental to NESASM source compatibility.

## Directives

### Already Supported

| NESASM Directive | retroasm Handler | Notes |
|---|---|---|
| `.db` / `.byte` | `Data` | Byte data |
| `.dw` / `.word` | `Data` | Word data |
| `.org` | `Base` | Set PC (within bank) |
| `.incbin` | `Include` | Binary include |
| `.include` | `Include` | Source include |
| `.if` / `.else` / `.endif` | Conditional | Conditionals |
| `.ifdef` / `.ifndef` | Conditional | Symbol conditionals |
| `.rsset` | `RSSet` | Set RS counter |
| `.rs` | `RS` | Reserve symbol space |
| `.inesprg` | `INESHeader` | PRG ROM banks |
| `.ineschr` | `INESHeader` | CHR ROM banks |
| `.inesmap` | `INESHeader` | Mapper number |
| `.inesmir` | `INESHeader` | Mirroring mode |

### New Directives to Add

| Directive | Behavior | Priority |
|---|---|---|
| `.bank` | Select ROM bank | High |
| `.macro` / `.endm` | Macro (NESASM syntax: `name .macro`) | High |
| `.ds` | Define storage (fill bytes) | Small |
| `.equ` / `=` | Symbol assignment | Supported |
| `.defchr` | Define 8x8 character tile inline | Medium |
| `.incchr` | Include and convert image to CHR | Low |
| `.pcm` | Include PCM audio data | Low |
| `.proc` / `.endp` | Procedure definition | Medium |
| `.procgroup` / `.endprocgroup` | Procedure group | Low |
| `.func` | Function-style macro | Low |
| `.zp` / `.bss` / `.code` / `.data` | Section switching | Medium |
| `.opt` | Assembler options | Small (no-op) |
| `.list` / `.nolist` / `.mlist` / `.nomlist` | Listing control | Small (no-op) |
| `.fail` | Assembly error | Small |

### `.macro` / `.endm` (NESASM Syntax)

```asm
add_val .macro
    clc
    adc \1          ; \1 = first parameter
.endm

    add_val #$10    ; expands to: clc / adc #$10
```

Key differences from standard macro syntax:
- Name comes **before** `.macro` keyword
- Parameters are referenced by number: `\1` through `\9`
- No named parameters in the definition
- Local labels inside macros use `.` prefix

**Implementation:** In NESASM mode, when parsing an identifier followed by `.macro`, treat the identifier as the macro name. During macro expansion, substitute `\1`-`\9` tokens with the corresponding arguments.

### `.defchr` Character Definition

```asm
.defchr $00111100,\
        $01000010,\
        $10000001,\
        $10000001,\
        $10000001,\
        $10000001,\
        $01000010,\
        $00111100
```

Defines an 8x8 tile using 8 rows of pixel data. Each digit (0-3) represents a 2-bit color value. The directive generates 16 bytes of CHR data (2 bitplanes).

**Priority:** Medium. Useful but not essential for most projects.

### `.incchr` Character Include

```asm
.incchr "sprite.pcx"
```

Includes and converts a PCX image file to NES CHR format.

**Priority:** Low. Most projects pre-convert graphics.

## Expression Operators

### Function-Style Operators

| Function | Meaning | Standard Equivalent |
|---|---|---|
| `LOW(expr)` | Low byte | `<expr` |
| `HIGH(expr)` | High byte | `>expr` |
| `BANK(label)` | Bank number of label | (no equivalent) |
| `SIZEOF(struct)` | Size of structure | (no equivalent) |
| `PAGE(label)` | Page number (addr >> 8) | (no equivalent) |

**Implementation:** Parse `LOW(`, `HIGH(`, `BANK(` as unary function-style operators in expression evaluation. `LOW`/`HIGH` map to existing `<`/`>` behavior.

### Standard Operators

NESASM supports standard arithmetic and bitwise operators similar to other assemblers. Operator precedence follows C conventions.

## Number Formats

| Format | Example | Notes |
|---|---|---|
| `$xx` | `$FF` | Hexadecimal |
| `%xxxx` | `%10101010` | Binary |
| Decimal | `255` | Standard decimal |
| `'c'` | `'A'` | Character constant |
| `@xxx` | `@377` | **Octal** (NESASM-specific) |

The `@` octal prefix is unique to NESASM and conflicts with `@` local labels in asm6/ca65. This is mode-specific.

## Sections

NESASM supports section-based organization:

```asm
.zp                ; zero-page section (for variables)
.bss               ; uninitialized data section
.code              ; code section (default)
.data              ; initialized data section
```

These are simpler than ca65's `.segment` system — they switch between predefined sections rather than arbitrary named segments.

## Group 0 / Group 1 Instructions

NESASM distinguishes between "group 0" (standard) and "group 1" (extended) 6502 instructions in some configurations. This affects which instruction set is available.

**Priority:** Low. Standard 6502 instruction set is sufficient for most NES projects.

## Implementation Order

| Step | Feature | Effort |
|---|---|---|
| 1 | `.bank` directive with bank tracking | Medium |
| 2 | Dot-prefixed local labels (`.label`) | Medium |
| 3 | NESASM macro syntax (`name .macro`, `\1`-`\9` params) | Large |
| 4 | `LOW()`/`HIGH()` function-style operators | Small |
| 5 | `@` octal number format | Small |
| 6 | `.ds` storage directive | Small |
| 7 | `.zp`/`.bss`/`.code`/`.data` sections | Medium |
| 8 | No-op directives (`.opt`, `.list`, `.nolist`) | Small |
| 9 | `.defchr` character tile definition | Medium |
| 10 | `BANK()` function | Medium |
| 11 | `.proc` / `.endp` | Medium |
| 12 | `.fail` directive | Small |

## Notes

- NESASM's bank model is tightly coupled to NES mapper hardware. The assembler needs to know the bank size (typically 8KB or 16KB for PRG, 8KB for CHR).
- The `name .macro` syntax (name before keyword) is unique among 6502 assemblers and requires special parsing logic.
- NESASM's `.` local labels can conflict with directive parsing since both start with `.`. The disambiguation rule: if the token after `.` is a known directive name, treat as directive; otherwise treat as local label.
- Many NESASM projects use a simple flat structure with sequential `.bank`/`.org` pairs, making compatibility relatively straightforward for common cases.
