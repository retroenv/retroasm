# x816 Compatibility Reference

This document tracks compatibility with the x816 assembler syntax used by
legacy 6502 and 65816 sources. Select `config.CompatX816` when constructing the
assembler or parser to enable these rules.

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

## Compatibility Validation

The original [Super Mario Bros. x816 disassembly](https://gist.github.com/1wErt3r/4048722)
passes the complete RetroASM pipeline when assembled in x816 mode inside a
configured `CODE` segment. The 16,351-line source fills the expected `$8000`–`$ffff`
PRG range, places `AreaParserCore` at `$93fc`, and matches the 32 KiB PRG output
from the asm6f port byte-for-byte. The external source is not vendored because it
does not state an open-source license.

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
