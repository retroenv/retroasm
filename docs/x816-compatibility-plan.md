# x816 Compatibility Reference

This document tracks compatibility with the x816 assembler syntax used by
legacy 6502 and 65816 sources. Select `config.CompatX816` when constructing the
assembler or parser to enable these rules.

This reference describes `work2`, reviewed on 2026-10-02. Use `-compat x816`
for the CLI. The [merge plan](work-branch-changes.md) separates existing
target behavior from later data-expression and codec changes.

## Implemented Syntax

### Labels and program counter

- Labels in column one may omit the trailing colon.
- `+` and `-` define anonymous labels. Repeated markers produce distinct
  nesting levels.
- `* = expression` assigns the program counter.
- `name .equ expression` defines a symbol.

Colon-optional detection does not reinterpret indented identifiers or known
instruction and directive names as labels.

### Expressions and addressing

- `#Symbol` and unparenthesized immediate expressions such as
  `#FirstMask+SecondMask` are supported.
- `<value`, `>value`, and `^value` select the low, high, and bank bytes.
- Forward labels may be used in aliases and data expressions.
- Numeric indirect operands such as `jmp ($06)` use indirect addressing.

### Data and storage directives

| Directive | Width | Behavior |
|---|---:|---|
| `.db`, `.dcb` | 1 | Byte data |
| `.dcw`, `.dw` | 2 | Word data |
| `.dcl`, `.dl` | 3 | Long data |
| `.dcd`, `.dd` | 4 | Double-word data |
| `.dsb` | 1 | Byte storage |
| `.dsw` | 2 | Word storage |
| `.dsl` | 3 | Long storage |
| `.dsd` | 4 | Double-word storage |

In x816 mode, `.dl` overrides the default low-address-byte directive.

### Includes and comment blocks

- `.src` is an alias for source inclusion.
- `.comment` consumes input through the next `.end` marker.
- An unterminated `.comment` block consumes through end of file.

### Ignored directives

The following directives are accepted and consume the rest of their line
without changing the generated output:

| Category | Directives |
|---|---|
| Bit width | `.mem`, `.index`, `.detect` |
| Listing and symbols | `.list`, `.nolist`, `.sym`, `.symbol` |
| Optimization and display | `.opt`, `.optimize`, `.dasm`, `.echo` |
| Diagnostics | `.cerror`, `.cwarn`, `.message` |
| ROM/output mode | `.hrom`, `.lrom`, `.hirom`, `.smc` |
| Parser settings | `.localsymbolchar`, `.locchar`, `.par`, `.parenthesis` |
| Block terminator | `.end` |

## Shared Behavior

x816 mode also uses the assembler's existing conditionals, macros, binary
includes, padding, keyword expression operators, and hexadecimal/binary number
formats.

## Historical compatibility report

The original [Super Mario Bros. x816 disassembly](https://gist.github.com/1wErt3r/4048722)
was reported to pass the RetroASM pipeline in an earlier review inside a
configured `CODE` segment. The 16,351-line source fills the expected `$8000`–`$ffff`
PRG range, places `AreaParserCore` at `$93fc`, and matches the 32 KiB PRG output
from the asm6f port byte-for-byte. The external source is not vendored because it
does not state an open-source license. This external comparison was not
repeated for the current documentation review. The report does not establish
complete x816 compatibility.

Current repository regressions are in `pkg/parser/parser_x816_test.go` and
`pkg/assembler/assembler_x816_test.go`. They cover label definitions, width
selection for data directives, forward data references, mixed expressions,
and numeric indirect jumps. No code tests were run for this documentation edit.

## Remaining Work

| Feature | Notes |
|---|---|
| Anonymous-label references | Resolve `+` and `-` operands to matching definitions |
| `.base` / `.end` blocks | Relocatable code blocks with nested state |
| `.module` / `.mod` | Module scoping and anonymous-label reset |
| `!` modifier | Force absolute addressing |
| `.table` / `.tab` | Virtual data tables |
| `.asctable` / `.asc` | Character remapping |
| `.interrupts` / `.int` | Interrupt vector table generation |
| `.cartridge` / `.cart` | Cartridge header generation |

The `:` and `\` multi-instruction line separators remain unsupported.
