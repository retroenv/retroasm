# asm6 / asm6f Assembler Compatibility

## Overview

asm6 (v1.6) by loopy is a popular 6502 assembler for NES development. asm6f is a community fork adding undocumented opcode support, iNES headers, and symbol file export. Both share identical syntax.

See [Compatibility Mode Infrastructure](compatibility-mode-plan.md) for shared features (anonymous labels, colon-optional labels, number formats).

## Directive Support

All asm6 directives listed below are implemented. Directives marked "base" are available in all compatibility modes; directives marked "asm6" are added by the asm6 handler overlay.

| Directive | Handler | Source |
|---|---|---|
| `ALIGN` | Align | base |
| `BASE` | Base | base |
| `BIN` | Include | base |
| `DB` / `BYTE` / `DCB` / `DCW` | Data | base |
| `DH` | AddrHigh | base |
| `DL` | AddrLow | base |
| `DSB` / `DSW` | DataStorage | base |
| `DW` / `WORD` | Data | base |
| `ELSE` / `ELSEIF` / `ENDIF` | Else / Elseif / Endif | base |
| `ENDE` | Ende | base |
| `ENDR` | Endr | base |
| `ENUM` | Enum | base |
| `ERROR` | Error | base |
| `FILLVALUE` | FillValue | base |
| `HEX` | Hex | base |
| `IF` / `IFDEF` / `IFNDEF` | If / Ifdef / Ifndef | base |
| `INCBIN` / `INCLUDE` / `INCSRC` | Include | base |
| `INESPRG` / `INESCHR` / `INESMAP` / `INESMIR` | NesasmConfig | base |
| `MACRO` | Macro | base |
| `ORG` | Base | base |
| `PAD` | Padding | base |
| `REPT` | Rept | base |
| `=` (assignment) | parseAlias | parser |

### asm6-Only Overlay Directives

These are registered by `asm6Handlers()` and available only in asm6 compatibility mode:

| Directive | Handler | Purpose |
|---|---|---|
| `ENDINL` | NoOp | Symbol file control (no-op) |
| `HUNSTABLE` | NoOp | Enable highly unstable opcodes (no-op) |
| `IGNORENL` | NoOp | Symbol file control (no-op) |
| `NES2BRAM` | Nes2Config | Battery PRG RAM size (NES 2.0) |
| `NES2CHRBRAM` | Nes2Config | Battery CHR RAM size (NES 2.0) |
| `NES2CHRRAM` | Nes2Config | CHR RAM size (NES 2.0) |
| `NES2PRGRAM` | Nes2Config | PRG RAM size (NES 2.0) |
| `NES2SUB` | Nes2Config | Submapper (NES 2.0) |
| `NES2TV` | Nes2Config | TV mode (NES 2.0) |
| `NES2VS` | Nes2Config | Vs. Unisystem (NES 2.0) |
| `UNSTABLE` | NoOp | Enable unstable opcodes (no-op) |

## Parser Features

### Labels

Labels are case-sensitive. The colon after a label is optional (`ColonOptionalLabels` returns true for asm6 mode).

**Local labels** begin with `@` and are scoped between non-local labels (`LocalLabelScoping` returns true for asm6 mode). At each non-local label definition, the scope resets. Local label names are prefixed with the parent label to produce unique scoped names (e.g., `label1.@tmp`).

```asm
label1:
  @tmp:     ; scoped as label1.@tmp
label2:
  @tmp:     ; scoped as label2.@tmp
```

**Anonymous labels** (`+`/`-`) are supported (`AnonymousLabels` returns true for asm6 mode). Consecutive `+` or `-` tokens increase the nesting level.

### `$` as Program Counter

`$` is recognized as the current program counter in expressions (`expression.ProgramCounterReference`). Direct PC assignment (`$=value`) is handled by the parser's `parseNumber` path, which delegates to `Base`.

### EQU vs `=`

- `EQU` is handled via `parseDotIdentifier` as an alias (numeric assignment).
- `=` evaluates to a number and the symbol can be reassigned.

Both are currently treated as numeric assignment. True text-substitution semantics for `EQU` (like C `#define`) are not implemented, but this rarely matters in practice since most real-world `EQU` usage is for numeric constants.

### Absolute Addressing Prefix (`a:`)

The `a:` prefix forces absolute (16-bit) addressing for addresses that would otherwise use zero-page (8-bit) mode. This is implemented and tested in both the parser and assembler (see `parser_test.go` and `assembler_ca65_test.go`).

```asm
lda a:$00    ; force absolute addressing
```

### Expression Operators

asm6 uses C-style operators with standard precedence:

| Precedence | Operators |
|---|---|
| Highest | `( )` |
| Unary | `+ - ~ ! < >` |
| Multiplicative | `* / %` |
| Additive | `+ -` |
| Shift | `<< >>` |
| Relational | `< > <= >=` |
| Equality | `= == != <>` |
| Bitwise AND | `&` |
| Bitwise XOR | `^` |
| Bitwise OR | `\|` |
| Logical AND | `&&` |
| Logical OR | `\|\|` |

Unary `<` and `>` give low/high byte of a 16-bit word.

## Not Implemented

The following asm6 features are not yet supported:

| Feature | Description |
|---|---|
| String arithmetic | `DB "ABCDE"+1` shifts all character values by an offset |
| Undocumented opcodes | `UNSTABLE`/`HUNSTABLE` are accepted as no-ops but do not gate opcode availability; the undocumented opcodes themselves need retrogolib registration |
