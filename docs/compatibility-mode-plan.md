# Assembler Compatibility Mode Infrastructure

## Overview

retroasm supports multiple legacy 6502 assembler syntaxes so that existing source files can be assembled without modification. This document describes the shared infrastructure for compatibility mode switching. Individual assembler details are in separate documents:

- [asm6 Compatibility](asm6-compatibility.md)
- [ca65 Compatibility](ca65-compatibility.md)
- [NESASM Compatibility](nesasm-compatibility.md)
- [x816 Compatibility](x816-compatibility-plan.md)

## Compatibility Mode Type

### Mode Enum

**File:** `pkg/assembler/config/compatibility.go`

```go
type CompatibilityMode int

const (
    CompatDefault CompatibilityMode = iota // current behavior (asm6/ca65 hybrid)
    CompatX816                             // x816 assembler (65816/6502)
    CompatAsm6                             // asm6 / asm6f
    CompatCa65                             // cc65 toolchain assembler
    CompatNesasm                           // NESASM (MagicKit)
)
```

### CLI Flags

**File:** `cmd/retroasm/main.go`

```
--compat string    Assembler compatibility mode (default, x816, asm6, ca65, nesasm)
-m string          Assembler compatibility mode (shorthand for -compat)
```

### Pipeline Threading

The `CompatibilityMode` field on `config.Config[T]` is passed through to the `Parser` constructor, which uses it to:

- Select the correct directive handler map via `directives.BuildHandlers(mode)`
- Gate syntax features through feature-query methods on `CompatibilityMode`

## Feature Methods

Each feature is exposed as a method on `CompatibilityMode` so the parser can query behavior without switching on mode constants directly.

| Method | asm6 | ca65 | NESASM | x816 |
|---|---|---|---|---|
| `AnonymousLabels()` | Yes | - | - | Yes |
| `AsteriskProgramCounter()` | - | Yes | Yes | Yes |
| `BankByteOperator()` | - | Yes | - | Yes |
| `ColonOptionalLabels()` | Yes | - | - | Yes |
| `DotLocalLabels()` | - | - | Yes | - |
| `LocalLabelScoping()` | Yes | Yes | - | - |
| `NesasmMacroSyntax()` | - | - | Yes | - |
| `UnnamedLabels()` | - | Yes | - | - |

## Shared Features

### Anonymous Labels (`+`/`-`)

Supported by **asm6** and **x816**. Labels consisting of one or more `+` or `-` characters define anonymous forward/backward branch targets.

| Feature | asm6 | x816 |
|---|---|---|
| Basic `+`/`-` | Yes | Yes |
| Nested `--`/`++` etc. | Yes (more unique names) | Yes (nesting level) |

The parser collects consecutive `+`/`-` tokens at line start, tracks nesting level, and generates synthetic label names encoding direction, level, and occurrence index (e.g., `__anon_fwd_1_3`).

### Colon-Optional Labels

Supported by **asm6** and **x816**. A trailing colon on labels is optional; labels are recognized by position (column 1) when the identifier is not a known instruction or directive.

In `parseIdentifier()`, when an identifier at column 1 is not an instruction:
1. Check if followed by `=` or `.equ` -- parse as assignment
2. Check if it is a directive -- parse as directive
3. If next token is instruction, directive, dot, EOL, or EOF -- treat as label definition
4. Otherwise -- fall through to current behavior

Both `label:` and `label` (at column 1) are accepted.

### Unnamed Labels (`:`)

Supported by **ca65**. A bare colon at line start defines an unnamed label. References use `:+` (forward) and `:-` (backward) with optional nesting level. Each unnamed label gets a synthetic name based on a counter (e.g., `__unnamed_3`).

### Local Label Scoping

Two variants are implemented:

- **`@local` scoping** (asm6, ca65): Labels starting with `@` are scoped under the last non-local label, producing names like `main.@loop`.
- **Dot-local scoping** (NESASM): Labels starting with `.` are scoped under the last non-local label, producing names like `main.loop`.

### NESASM Macro Syntax

Supported by **NESASM**. Macros use `name .macro` syntax (name before directive) instead of `.macro name`. The parser detects this pattern in `parseDotIdentifier()`.

### No-Op Directive Handler

A shared handler that consumes all tokens until end of line, used for directives that do not affect binary output (listing, display, symbol files, optimization hints):

```go
func NoOp(p arch.Parser) (ast.Node, error) {
    for {
        p.AdvanceReadPosition(1)
        if p.NextToken(0).Type.IsTerminator() {
            return nil, nil
        }
    }
}
```

### Program Counter Symbol

| Assembler | PC Symbol |
|---|---|
| asm6 (default) | `$` |
| ca65 | `*` (in `* = value` assignments) |
| NESASM | `*` (in `* = value` assignments) |
| x816 | `*` (in `* = value` assignments) |

The parser handles `*` as a program counter assignment (`* = $8000`) by delegating to the `Base` directive handler.

### Low/High Byte and Bank Byte Operators

| Operator | Meaning | asm6 | ca65 | NESASM | x816 |
|---|---|---|---|---|---|
| `<value` | Low byte | Yes | Yes | `LOW()` | Yes |
| `>value` | High byte | Yes | Yes | `HIGH()` | Yes |
| `^value` | Bank byte (bits 16-23) | - | Yes (`.bankbyte`) | - | Yes |

The `<` and `>` prefix operators are handled as `AddrLow` and `AddrHigh` directives. The `^` bank byte operator is gated by `BankByteOperator()`.

### Number Formats

| Format | asm6 | ca65 | NESASM | x816 |
|---|---|---|---|---|
| `$xx` hex | Yes | Yes | Yes | Yes |
| `%xxxx` binary | Yes | Yes | Yes | Yes |
| `xxxb` trailing binary | Yes | - | - | Yes |
| `0x` C-style hex | - | Yes | - | - |

**File:** `pkg/number/number.go`

The number parser supports `$` hex prefix, `%` binary prefix, trailing `b` binary suffix, and `0x` C-style hex prefix.

Note: Trailing `h` hex suffix and `@` octal prefix (NESASM) are not yet implemented in the shared number parser.

## Directive Registration

### Mode-Specific Directive Maps

**File:** `pkg/parser/directives/directives.go`

`BuildHandlers()` constructs directive maps per compatibility mode. The base map contains universally supported directives, and each mode overlays its specific additions:

```go
func BuildHandlers(mode config.CompatibilityMode) map[string]Handler {
    handlers := baseHandlers()
    switch mode {
    case config.CompatX816:
        mergeHandlers(handlers, x816Handlers())
    case config.CompatAsm6:
        mergeHandlers(handlers, asm6Handlers())
    case config.CompatCa65:
        mergeHandlers(handlers, ca65Handlers())
    case config.CompatNesasm:
        mergeHandlers(handlers, nesasmHandlers())
    }
    return handlers
}
```

Mode-specific directive counts (beyond base):

| Mode | Directives added |
|---|---|
| asm6 | 9 (NES 2.0 header, symbol file control, undocumented opcode tiers) |
| ca65 | 25 (scoping, data, diagnostics, linker, charmap) |
| NESASM | 10 (storage, procedure, listing, section switching) |
| x816 | 26 (data widths, source include, comment blocks, mode/optimization) |

### Case Sensitivity

All modes are case-insensitive for directives and mnemonics but case-sensitive for labels. This matches retroasm's default behavior; directive names are lowercased before handler lookup.

## Implementation Status

All shared infrastructure features listed in this document are implemented:

| Feature | Status |
|---|---|
| Anonymous `+`/`-` labels | Done |
| Asterisk program counter (`*`) | Done |
| Bank byte operator (`^`) | Done |
| CLI `--compat` / `-m` flag | Done |
| Colon-optional labels | Done |
| Compatibility mode enum and parsing | Done |
| Dot-local label scoping (NESASM) | Done |
| Feature query methods | Done |
| Local `@` label scoping | Done |
| Low/high byte operators | Done |
| Mode-specific directive maps | Done |
| NESASM macro syntax | Done |
| No-op directive handler | Done |
| Number format parsing | Done (trailing `h` hex and `@` octal not yet added) |
| Pipeline threading (config to parser) | Done |
| Unnamed labels (ca65 `:`) | Done |
