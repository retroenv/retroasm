# x816 Assembler Compatibility Reference

## Overview

x816 (65816/6502 assembler by minus/Ballistics, v1.12f) is a legacy assembler originally targeting the 65816 (SNES) but also used for 6502 (NES) development with `.mem 8` / `.index 8` to force 8-bit mode.

See [Compatibility Mode Infrastructure](compatibility-mode-plan.md) for shared features: compatibility mode type, CLI flag, pipeline threading, anonymous labels, colon-optional labels, no-op directive handler, and number formats.

## Label Behavior

### Anonymous Labels

x816 uses `+`/`-` anonymous labels. Implemented via `AnonymousLabels()` feature flag in `compatibility.go`, with parsing in `parser.go` (`parseAnonymousLabel`). Nesting level is tracked by counting consecutive `+` or `-` tokens.

### Colon-Optional Labels

x816 labels do not require a trailing colon. Implemented via `ColonOptionalLabels()` feature flag, with column-0 detection in `parser.go` (`isColonOptionalLabel`). Colon labels are also accepted.

## Directives

### Status Key

- [x] = Implemented
- [ ] = Not yet implemented

### Directives Supported via Base Handlers

These x816 directives work through the base handler map shared across all modes.

| Directive | Handler | Notes | Status |
|---|---|---|---|
| `.bin` | `Include` | Binary include | [x] |
| `.db` | `Data` | Byte data (1-byte) | [x] |
| `.dcb` | `Data` | Byte data (1-byte) | [x] |
| `.dcw` | `Data` | Data (mapped to 1-byte width -- see note) | [x] |
| `.dsb` | `DataStorage` | Byte storage | [x] |
| `.dsw` | `DataStorage` | Word storage | [x] |
| `.dw` | `Data` | Word data (2-byte) | [x] |
| `.else` | `Else` | Conditional else | [x] |
| `.endif` | `Endif` | End conditional | [x] |
| `.if` | `If` | Conditional | [x] |
| `.ifdef` | `Ifdef` | If defined | [x] |
| `.ifndef` | `Ifndef` | If not defined | [x] |
| `.incbin` | `Include` | Binary include | [x] |
| `.incsrc` | `Include` | Source include | [x] |
| `.macro` | `Macro` | Macro definition | [x] |
| `.org` | `Base` | Origin address | [x] |
| `.pad` | `Padding` | Pad to address | [x] |

**Note:** `.dcw` is mapped to 1-byte width in `dataByteWidth`. This may be a bug if x816 `.dcw` is intended to produce 2-byte word data.

### x816-Specific Directives (Implemented)

These are registered in `x816Handlers()` in `directives.go`.

| Directive | Handler | Notes | Status |
|---|---|---|---|
| `.comment` | `CommentBlock` | Multi-line comment block (skip to `.end`) | [x] |
| `.dcd` / `.dd` | `Data` | Double-word data (4-byte) | [x] |
| `.dcl` / `.dl` | `Data` | Long data (3-byte); `.dl` overrides base `AddrLow` | [x] |
| `.dsd` | `DataStorage` | Double-word storage (4-byte) | [x] |
| `.dsl` | `DataStorage` | Long storage (3-byte) | [x] |
| `.end` | `NoOp` | Block terminator | [x] |
| `.src` | `Include` | Source include alias | [x] |

### x816-Specific No-Op Directives (Implemented)

All registered in `x816Handlers()`. These consume tokens to end-of-line without affecting output.

| Directive | Category |
|---|---|
| `.cerror` | Diagnostic messages |
| `.cwarn` | Diagnostic messages |
| `.dasm` | Display |
| `.detect` | Bitwidth autodetection |
| `.echo` | Display |
| `.hirom` | ROM mode (NES-irrelevant) |
| `.hrom` | ROM mode (NES-irrelevant) |
| `.index` | Index register bitwidth (always 8-bit for NES) |
| `.list` | Listing output |
| `.localsymbolchar` / `.locchar` | Local symbol character setting |
| `.lrom` | ROM mode (NES-irrelevant) |
| `.mem` | Accumulator bitwidth (always 8-bit for NES) |
| `.message` | Diagnostic messages |
| `.nolist` | Listing output |
| `.opt` / `.optimize` | Address optimization |
| `.par` / `.parenthesis` | Parenthesis style |
| `.smc` | SMC output (NES-irrelevant) |
| `.sym` / `.symbol` | Symbol file output |

### Macro Termination

`.endm` and `ENDM` are handled inside the `Macro` reader loop (in `macro.go`), not as standalone directive entries. No separate handler registration is needed.

### .equ Alias

`name .equ value` is handled via the `parseDotIdentifier` path in `parser.go`, which detects the `.equ` keyword after an identifier and delegates to `parseAlias`. This is implemented.

### Not Yet Implemented

| Directive | Behavior | Priority |
|---|---|---|
| `.asc` | Data with ASCII remapping | Low |
| `.asctable` | ASCII remapping table | Low |
| `.base` | Relocatable base address (with `.end` pairing) | Medium |
| `.cartridge` / `.cart` | Cartridge header | Low |
| `.interrupts` / `.int` | Interrupt vector table | Low |
| `.module` / `.mod` | Module scope (clears anonymous label state) | Low |
| `.table` / `.tab` | Virtual data table (like `.base` but no output) | Low |

## Expressions

### Keyword Operators (Implemented)

Keyword operators are defined in `expression.go` (`keywordOperators` map) and resolved via `resolveKeywordOperator` during expression parsing. Bitwise evaluation is in `operator.go` (`evaluateBitwiseIntInt`).

| Keyword | Symbol | Status |
|---|---|---|
| `AND` | `&` (bitwise AND) | [x] |
| `OR` | `\|` (bitwise OR) | [x] |
| `SHL` | `<<` (shift left) | [x] |
| `SHR` | `>>` (shift right) | [x] |
| `XOR` | `^` (bitwise XOR) | [x] |

**Note:** These are bitwise operators (not logical). `AND` maps to `token.Ampersand`, `OR` maps to `token.Pipe`.

### Value Modifiers

| Modifier | Meaning | Status |
|---|---|---|
| `<value` | Low byte | [x] |
| `>value` | High byte | [x] |
| `^value` | Bank byte (bits 16-23) | [x] |
| `!value` | Force absolute addressing | [ ] |

The `^` bank byte operator is enabled via `BankByteOperator()` feature flag in `compatibility.go` (returns `true` for x816 and ca65 modes).

### Current Address Symbol (Implemented)

x816 uses `*` as the current program counter. Implemented via `AsteriskProgramCounter()` feature flag and `parseAsteriskPC()` in `parser.go`. Supports `* = $8000` assignment syntax.

### Number Formats (Implemented)

Trailing `h` suffix (hexadecimal) and `b` suffix (binary) are supported in the number parser (`pkg/number`).

## .dcb String Support

x816's `.dcb` accepts quoted strings mixed with numeric values:

```
.dcb 13,10,"=main=",13,10
.dcb "NOTTUB >A< SSERP"
```

String tokens in data directives are processed by the expression evaluator's `processData` function, which emits each character as a byte value.

## Remaining Work

| Item | Description | Priority |
|---|---|---|
| `.base`/`.end` pairing | Relocatable code blocks (stack-based) | Medium |
| `.module` scoping | Module scope with anonymous label clearing | Low |
| `!` force absolute | Force absolute addressing modifier | Low |
| `.table`/`.tab` | Virtual data table | Low |
| `.asctable`/`.asc` | ASCII remapping | Low |
| `.interrupts`/`.int` | Interrupt vector table | Low |
| `.cartridge`/`.cart` | Cartridge header | Low |

## Notes

- x816 is case-insensitive for mnemonics and directives but preserves case in quoted strings
- The `:` and `\` multi-instruction-per-line separator is low-priority (not used in test sources)
- The `.base`/`.end` relocatable code feature is distinct from `.org` and may need a stack-based implementation
