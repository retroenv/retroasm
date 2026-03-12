# Assembler Compatibility Mode Infrastructure

## Overview

retroasm needs to support multiple legacy 6502 assembler syntaxes so that existing source files can be assembled without modification. This document describes the shared infrastructure for compatibility mode switching. Individual assembler details are in separate documents:

- [x816 Compatibility](x816-compatibility-plan.md)
- [asm6 Compatibility](asm6-compatibility.md)
- [ca65 Compatibility](ca65-compatibility.md)
- [NESASM Compatibility](nesasm-compatibility.md)

## Current State

retroasm already defines format constants in `pkg/retroasm/assembler.go`:

```go
const (
    FormatAsm6   = "asm6"
    FormatCa65   = "ca65"
    FormatNesasm = "nesasm"
)
```

These are used for output format selection. The compatibility mode system extends this to also affect **input parsing** — directive names, label syntax, number formats, and expression operators.

## Phase 1: Compatibility Mode Type

### 1.1 Define Compatibility Mode Enum

**File: `pkg/assembler/config/compatibility.go` (new)**

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

### 1.2 CLI Flag

**File: `cmd/retroasm/main.go`**

Add `--compat` / `-m` flag:
```
-m, --compat string    Assembler compatibility mode (default, x816, asm6, ca65, nesasm)
```

### 1.3 Thread Mode Through Pipeline

The compatibility mode must reach:
- **Lexer** — number format differences (trailing `h`/`b`, `@` prefix for octal in NESASM)
- **Parser** — label syntax, directive dispatch, expression operators
- **Directives** — mode-specific behavior and no-op handlers

Add `CompatibilityMode` field to `config.Config[T]` and pass to `Parser` constructor.

## Phase 2: Shared Features

### 2.1 Anonymous Labels (`+`/`-`)

Used by both **x816** and **asm6/asm6f**. Labels consisting of one or more `+` or `-` characters define anonymous forward/backward branch targets.

| Feature | x816 | asm6 |
|---|---|---|
| Basic `+`/`-` | Yes | Yes |
| Nested `--`/`++` etc. | Yes (nesting level) | Yes (more unique names) |
| Suffix text (`+here`) | No | Yes (`+here` is valid) |
| Scope reset on `.module` | Yes | No (reset on non-local label) |

**Implementation:**

In `TokensToAstNodes()`, when `token.Minus` or `token.Plus` appears at line start:
1. Collect consecutive `+`/`-` tokens and any trailing identifier text
2. Generate synthetic label name encoding direction, nesting level, and occurrence index
3. Emit as `ast.Label` node

For references in operand position:
1. First pass: collect all anonymous label positions with synthetic names
2. Resolution pass: match references to nearest appropriate definition

### 2.2 Colon-Optional Labels

Both **x816** and **asm6** treat the trailing colon on labels as optional. A label is recognized by position (column 0 / start of line) when the identifier is not a known instruction or directive.

**Current state:** The parser requires `identifier:` for labels.

**Implementation in `parseIdentifier()`:**

In compat modes (x816, asm6), when an identifier is at column 0:
1. Check if it's an instruction mnemonic → parse as instruction
2. Check if followed by `=` or `EQU` → parse as assignment
3. Check if it's a directive → parse as directive
4. If next token is instruction/directive/EOL → treat as label definition
5. Otherwise → try instruction parsing (current behavior)

Both `label:` and `label` (at column 0) are accepted for maximum compatibility.

### 2.3 No-Op Directive Handler

Many assemblers have directives for listing, display, and symbol file output that don't affect binary output. A shared no-op handler skips to EOL:

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

### 2.4 Program Counter Symbol

| Assembler | PC Symbol |
|---|---|
| asm6 (current) | `$` |
| x816 | `*` |
| ca65 | `*` (in expressions), `.loword(*)` etc. |
| NESASM | `*` (limited) |

In x816/ca65 modes, also accept `*` as `expression.ProgramCounterReference` in expression context.

### 2.5 Number Formats

| Format | asm6 | x816 | ca65 | NESASM |
|---|---|---|---|---|
| `$xx` hex | Yes | Yes | Yes | Yes |
| `%xxxx` binary | Yes | Yes | Yes | Yes |
| `0xxh` trailing hex | Yes | Yes | No | No |
| `xxxb` trailing binary | Yes | Yes | No | No |
| `@xxx` octal | No | No | No | Yes |
| `0x` C-style hex | No | No | Yes | No |

**File: `pkg/number/number.go`** — Add trailing `b` binary parsing and `@` octal parsing based on compat mode.

### 2.6 Low/High Byte Operators

| Operator | Meaning | asm6 | x816 | ca65 | NESASM |
|---|---|---|---|---|---|
| `<value` | Low byte | Yes | Yes | Yes | `LOW()` |
| `>value` | High byte | Yes | Yes | Yes | `HIGH()` |
| `^value` | Bank byte (bits 16-23) | No | Yes | Yes (`.bankbyte`) | No |

### 2.7 String Data in `.db`/`.dcb`

All assemblers support quoted strings in data directives, emitting each character as a byte:

```
.db "HELLO",0        ; asm6, ca65
.dcb "HELLO",13,10   ; x816
.db "HELLO",0        ; NESASM
```

Ensure the `Data` directive handler processes quoted string tokens by emitting each character as a byte value, then continuing with comma-separated values.

## Phase 3: Directive Registration

### 3.1 Mode-Specific Directive Maps

Rather than one global `Handlers` map, build directive maps per compatibility mode. The base map contains universally supported directives, and each mode overlays its specific additions:

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
        mergeHandlers(handlers, nesamHandlers())
    }
    return handlers
}
```

### 3.2 Case Sensitivity

| Assembler | Directives | Mnemonics | Labels |
|---|---|---|---|
| x816 | Case-insensitive | Case-insensitive | Case-sensitive |
| asm6 | Case-insensitive | Case-insensitive | Case-sensitive |
| ca65 | Case-insensitive (with `.`) | Case-insensitive | Case-sensitive |
| NESASM | Case-insensitive | Case-insensitive | Case-sensitive |

All assemblers are case-insensitive for directives and mnemonics but case-sensitive for labels. This matches retroasm's current behavior.

## Phase 4: Test Infrastructure

### 4.1 Per-Assembler Test Suites

Each compatibility mode should have:
1. **Unit tests** for mode-specific features (label parsing, directives, number formats)
2. **Integration tests** assembling real-world source files
3. **Binary comparison tests** against reference output from the original assembler (where possible)

### 4.2 Reference Binary Generation

For x816: run via dosbox to produce reference `.bin` files.
For asm6: run native binary (Linux/Windows).
For ca65: run ca65 + ld65 toolchain.
For NESASM: run native binary.

Store reference outputs in `tests/<assembler>/` directories.

## Implementation Priority

| Priority | Feature | Assemblers |
|---|---|---|
| 1 | Compatibility mode infrastructure + CLI flag | All |
| 2 | No-op directive handler | x816, ca65, NESASM |
| 3 | Anonymous `+`/`-` labels | x816, asm6 |
| 4 | Colon-optional labels | x816, asm6 |
| 5 | `*` as program counter | x816, ca65 |
| 6 | String data in `.db`/`.dcb` | All |
| 7 | Mode-specific directive maps | All |
| 8 | Trailing `b` binary numbers | x816, asm6 |
| 9 | Individual assembler features | Per-assembler docs |
