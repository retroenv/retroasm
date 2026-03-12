# x816 Assembler Compatibility Plan

## Overview

x816 (65816/6502 assembler by minus/Ballistics, v1.12f) is a legacy assembler originally targeting the 65816 (SNES) but also used for 6502 (NES) development with `.mem 8` / `.index 8` to force 8-bit mode.

See [Compatibility Mode Infrastructure](compatibility-mode-plan.md) for shared features: compatibility mode type, CLI flag, pipeline threading, anonymous labels, colon-optional labels, no-op directive handler, and number formats.

## x816-Specific Label Behavior

### Anonymous Labels

x816 uses `+`/`-` anonymous labels (see shared infrastructure). x816-specific behavior:
- No trailing text suffix (unlike asm6's `+here`)
- Scoped to `.MODULE` blocks — a new module clears anonymous label state

### Colon-less Labels

x816 labels do not require a trailing colon (see shared infrastructure). x816-specific: column-0 detection is the primary label recognition method; colon labels are also accepted.

## x816-Specific Directives

### Directives Already Supported

These x816 directives already work in retroasm (via asm6/ca65 compatibility):

| x816 Directive | retroasm Handler | Notes |
|---|---|---|
| `.org` | `Base` | Origin address |
| `.dcb` / `.db` | `Data` | Byte data |
| `.dcw` / `.dw` | `Data` | Word data |
| `.dsb` | `DataStorage` | Byte storage |
| `.dsw` | `DataStorage` | Word storage |
| `.pad` | `Padding` | Pad to address |
| `.incbin` / `.bin` | `Include` | Binary include |
| `.incsrc` / `.src` | `Include` | Source include |
| `.if` | `If` | Conditional |
| `.else` | `Else` | Conditional else |
| `.endif` | `Endif` | End conditional |
| `.ifdef` | `Ifdef` | If defined |
| `.ifndef` | `Ifndef` | If not defined |
| `.macro` | `Macro` | Macro definition |

### New Directives to Add

**File: `pkg/parser/directives/directives.go` - add to `Handlers` map**

| x816 Directive | Behavior | Priority |
|---|---|---|
| `.mem` | Set accumulator bitwidth (8/16) - no-op for NES (always 8-bit) | High |
| `.index` | Set index register bitwidth (8/16) - no-op for NES (always 8-bit) | High |
| `.opt` / `.optimize` | Toggle address optimization - no-op | High |
| `.list` | Toggle/set listing output - no-op | High |
| `.symbol` | Toggle/set symbol file output - no-op | High |
| `.endm` | End macro (alias) | High |
| `.end` | End block (`.base`, `.table`, `.comment`, etc.) | Medium |
| `.base` | Set relocatable base address (with `.end` pairing) | Medium |
| `.equ` | Assign value to symbol (alias for `=`) | Medium |
| `.detect` | Toggle bitwidth autodetection - no-op | Low |
| `.dasm` | Toggle assembly display - no-op | Low |
| `.echo` | Echo text during assembly - no-op | Low |
| `.comment` | Multi-line comment block (skip to `.end`) | Low |
| `.module` / `.mod` | Define module scope | Low |
| `.localsymbolchar` / `.locchar` | Set local symbol char - no-op | Low |
| `.hrom` / `.lrom` / `.hirom` | ROM mode - no-op for NES | Low |
| `.smc` | SMC output - no-op | Low |
| `.par` / `.parenthesis` | Parenthesis style - no-op | Low |
| `.table` / `.tab` | Virtual data table (like `.base` but no output) | Low |
| `.interrupts` / `.int` | Interrupt vector table | Low |
| `.cartridge` / `.cart` | Cartridge header | Low |
| `.asctable` | ASCII remapping table | Low |
| `.asc` | Data with ASCII remapping | Low |
| `.dcl` / `.dl` | Long (3-byte) data | Low |
| `.dcd` / `.dd` | Double-word (4-byte) data | Low |
| `.dsl` | Long storage | Low |
| `.dsd` | Double-word storage | Low |

### No-Op Directives

Many x816 directives (`.opt`, `.mem`, `.index`, `.list`, `.symbol`, `.detect`, `.dasm`, `.echo`, `.localsymbolchar`, `.locchar`, `.hrom`, `.lrom`, `.hirom`, `.smc`, `.par`, `.parenthesis`) are assembler-UI features that don't affect binary output. These use the shared no-op handler (see infrastructure doc).

## Expression and Number Format Differences

Number format support (trailing `h`/`b`) is covered in the shared infrastructure doc.

### Expression Operators

x816 supports keyword-based logical operators in expressions. These are used in `.dcb` data and address expressions:

| x816 | Meaning | Current |
|---|---|---|
| `SHL` / `<<` | Shift left | `<<` supported |
| `SHR` / `>>` | Shift right | `>>` supported |
| `AND` / `.AND.` / `&&` | Logical AND | `&&` may be supported |
| `OR` / `.OR.` / `\|\|` | Logical OR | `\|\|` may be supported |
| `XOR` / `.XOR.` | Logical XOR | Check |

**File: `pkg/expression/expression.go`**

In x816 mode, recognize `SHL`, `SHR`, `AND`, `OR`, `XOR` as operator keywords during expression evaluation.

### Modifiers

x816 value modifiers (already partially supported):

| Modifier | Meaning | Current |
|---|---|---|
| `<value` | Low byte | Yes (`<`) |
| `>value` | High byte | Yes (`>`) |
| `^value` | Bank byte (bits 16-23) | No |
| `!value` | Force word/absolute addressing | No |

Add `^` (bank byte) support for 65816 mode. The `!` modifier forces absolute addressing even for zero-page addresses.

### Current Address Symbol

x816 uses `*` as the current program counter in expressions:
```
here = *
length = *-start
```

**Current state:** The parser uses `$` as the program counter reference (`expression.ProgramCounterReference`). In x816 mode, also accept `*` in expression context.

## .dcb String Support

x816's `.dcb` accepts both numeric values and quoted strings in the same directive:

```
.dcb 13,10,"=main=",13,10
.dcb "NOTTUB >A< SSERP"
```

**Current state:** Check if the `Data` directive handler supports quoted string literals mixed with numbers. The lexer needs to emit string tokens properly.

### Implementation

In the data directive handler, when a quoted string is encountered:
- Emit each character as a byte value
- Continue parsing after the closing quote for more comma-separated values

## Test Infrastructure

### Binary Comparison Test

**File: `pkg/arch/m6502/x816_test.go` or `tests/nes-bench/bench_test.go` (new)**

```go
func TestX816_NESBench(t *testing.T) {
    // 1. Assemble tests/nes-bench/bench.asm using retroasm in x816 mode
    // 2. Compare output with reference bench.bin (produced by x816.exe via dosbox)
    // 3. Byte-for-byte comparison
}
```

### Generate Reference Binary

Run x816 via dosbox to produce `bench.bin` as the reference output. Store it in `tests/nes-bench/bench.bin.expected` (or similar).

### Unit Tests

Add targeted tests for each x816-specific feature:
- Anonymous label resolution (forward/backward, nested)
- Colon-less label parsing
- `.dcb` with mixed strings and numbers
- No-op directives (`.opt`, `.mem`, `.index`, `.list`, `.symbol`)
- `.pad` behavior
- Expression with `*` as program counter

## Implementation Order

| Step | Description | Effort |
|---|---|---|
| 1 | Compatibility mode infrastructure + CLI flag | Small |
| 2 | No-op directives (`.opt`, `.mem`, `.index`, `.list`, `.symbol`, `.detect`) | Small |
| 3 | `.dcb` string literal support in data directives | Medium |
| 4 | Anonymous `+`/`-` label support | Large |
| 5 | Colon-less label parsing (column-0 detection) | Medium |
| 6 | `*` as program counter in expressions | Small |
| 7 | `.equ` directive alias | Small |
| 8 | `.end` block terminator | Medium |
| 9 | Trailing `b` binary number format | Small |
| 10 | Reference binary test against x816 output | Medium |
| 11 | Bank byte `^` modifier | Small |
| 12 | Keyword expression operators (`SHL`, `AND`, etc.) | Medium |
| 13 | `.module` scoping | Large |
| 14 | Remaining low-priority directives | Medium |

Steps 1-10 should be sufficient to assemble the `tests/nes-bench/bench.asm` source correctly.

## Notes

- x816 was originally a 65816 (SNES) assembler but the bench.asm test uses it for 6502 (NES) code with `.mem 8` / `.index 8` to force 8-bit mode
- The `:` and `\` multi-instruction-per-line separator is a low-priority feature (not used in the test sources)
- The `.base`/`.end` relocatable code feature is distinct from `.org` and may need a stack-based implementation
- x816 is case-insensitive for mnemonics and directives but preserves case in quoted strings
