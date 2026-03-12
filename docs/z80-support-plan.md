# Z80 Architecture Support

## Overview

Z80 assembler support for retroasm. All planned phases (0--19) are complete.
Implementation dates: February 28 -- March 5, 2026.

## Architecture

### Package Structure

| Package | Purpose |
|---------|---------|
| `pkg/arch/z80/` | Architecture adapter, instruction grouping, options |
| `pkg/arch/z80/assembler/` | Address assignment and opcode generation |
| `pkg/arch/z80/parser/` | Operand parsing, instruction resolution, diagnostics |
| `pkg/arch/z80/profile/` | Profile-based instruction filtering |

### Key Design Decisions

**Generic type parameter:** `T = *InstructionGroup` groups all instruction variants by mnemonic name.

```go
type InstructionGroup struct {
    Name     string
    Variants []*z80.Instruction
}
```

**Instruction group sources:** Groups are built from `Opcodes`, `EDOpcodes`, `DDOpcodes`, `FDOpcodes` tables, plus explicit CB-family (`CBRlc`, `CBRrc`, `CBRl`, `CBRr`, `CBSla`, `CBSra`, `CBSll`, `CBSrl`, `CBBit`, `CBRes`, `CBSet`) and indexed-bit (`DdcbShift`, `DdcbBit`, `DdcbRes`, `DdcbSet`, `FdcbShift`, `FdcbBit`, `FdcbRes`, `FdcbSet`) instruction vars.

**Typed operand payloads:** Operands use a typed Z80 argument model (not strings), representing register/condition params, immediate/address/relative/displacement expressions, and bit indices. Payloads flow through `ast.InstructionArgument` and `ast.InstructionArguments`.

**16-bit address width.**

### Resolver Structure

The resolver is split across 9 files (1761 lines total):

| File | Lines | Responsibility |
|------|------:|----------------|
| `resolver.go` | 166 | Types, dispatcher, no-operand, shared helpers |
| `resolver_diagnostics.go` | 248 | Error diagnostics and ambiguity guidance |
| `resolver_extended.go` | 181 | Extended memory operations |
| `resolver_indexed.go` | 106 | Indexed register operations |
| `resolver_indirect.go` | 295 | Indirect load/store and indirect immediate |
| `resolver_port.go` | 139 | Port I/O operations |
| `resolver_single_operand.go` | 243 | Single operand resolution |
| `resolver_two_operand.go` | 224 | Two operand dispatch, register pairs, special pairs |
| `resolver_value.go` | 159 | Value-register and bit operations |

### Profiles

Three profiles are supported via `-z80-profile` CLI flag:

| Profile | CLI Value | Behavior |
|---------|-----------|----------|
| Default | `default` | Accepts all instructions including undocumented |
| Strict Documented | `strict-documented` | Rejects undocumented opcodes and aliases |
| Game Boy Subset | `gameboy-z80-subset` | Rejects DD/ED/FD prefixes and unsupported mnemonics (DJNZ, EX, EXX, IN, OUT) |

### CLI Integration

- `-cpu z80` selects Z80 architecture
- Compatible systems: `generic`, `gameboy`, `zx-spectrum`
- System defaults: `gameboy` and `zx-spectrum` default to Z80; `generic` defaults to Z80
- `-z80-profile` flag for profile selection

## Supported Features

### Operand Forms

- Implied (`NOP`, `RET`)
- Register/register (`LD A,B`)
- Register/immediate (`LD A,n`, `LD HL,nn`)
- Relative jump (`JR e`, `JR NZ,e`)
- Extended jump/call (`JP nn`, `JP NZ,nn`, `CALL nn`)
- Parenthesized indirect (`(HL)`, `(BC)`, `(DE)`)
- Indexed displacement (`(IX+d)`, `(IY+d)`, `(IY-1)`)
- Extended indirect register transfer (`LD A,(nn)`, `LD (nn),A`, `LD BC,(nn)`, `LD (nn),BC`)
- Port I/O immediate (`IN A,(n)`, `OUT (n),A`)
- Port I/O register (`IN B,(C)`, `OUT (C),E`)
- Value-first register (`BIT 3,A`, `RES 7,B`, `SET 0,C`)
- Numeric register (`IM 1`, `RST $38`)
- Indexed bit operations (`BIT 3,(IX+5)`, `RES 7,(IY-128)`, `SET 0,(IX+127)`)
- Tokenized offset expressions (`label+1`, `(label+1)`, `($10+1)`)
- Chained offset expressions (`label+3-1`, `(label+3-1)`)
- Expression-backed values (`target+delta`, `table+index`, `(IX+disp)`)

### Opcode Prefix Chains

- No prefix (base opcodes)
- Single prefix: `CB`, `DD`, `ED`, `FD`
- Double prefix indexed-bit: `DD CB d op`, `FD CB d op`

### Disambiguation

- `C` register vs `C` condition: resolved by mnemonic and operand position context
- `LD` direction for indirect/extended: resolved by opcode-direction checks against operand order
- Indexed vs non-indexed forms: routed by IX/IY base register detection before generic parsing

## Test Coverage

### Unit Tests

- Parser classification and resolver (including `C` ambiguity)
- Parser fuzz/property determinism (`pkg/arch/z80/parser/fuzz_test.go`)
- Address assignment for mixed instruction sizes
- Opcode generation for each addressing family
- Opcode boundary matrices (relative at -128/+127, displacement at 0x00/0xFF, port at 0x00/0xFF, extended at 0x0000/0xFFFF)
- Metadata-driven coverage test (`pkg/arch/z80/assembler/coverage_test.go`) enumerating all instruction variants

### Integration Fixtures

| Fixture | Coverage |
|---------|----------|
| `tests/z80/basic.asm` | Core instruction smoke test |
| `tests/z80/branches.asm` | Relative and absolute control-flow encoding |
| `tests/z80/branches_overflow.asm` | Relative branch out-of-range regression |
| `tests/z80/compatibility.asm` | Mixed control-flow and expression compatibility |
| `tests/z80/expressions.asm` | Expression-backed operands and indexed displacements |
| `tests/z80/indexed.asm` | IX/IY indexed operands and prefixed opcodes (DD, FD, ED) |
| `tests/z80/indexed_boundaries.asm` | Indexed displacement boundaries and indexed CB-family ops |
| `tests/z80/io_extended.asm` | Extended indirect register transfer and port I/O |
| `tests/z80/offsets.asm` | Tokenized offset operands |
| `tests/z80/offsets_chained.asm` | Chained tokenized offset operands |
| `tests/z80/profile_gameboy_subset.asm` | Game Boy subset profile positive assembly |
| `tests/z80/profile_gameboy_subset_rejects.asm` | Game Boy subset profile rejection diagnostics |
| `tests/z80/profile_strict_documented.asm` | Strict documented profile positive assembly |
| `tests/z80/profile_strict_documented_rejects.asm` | Strict documented profile rejection diagnostics |

### Regression Requirements

- Every bug fix adds a focused regression test.
- Uses `github.com/retroenv/retrogolib/assert`.
- Uses `t.Context()` in tests.

## Implementation Notes

### retrogolib Integration

- `LD (HL),r` instructions use `LdReg8.RegisterPairOpcodes` (not `LdIndirect.RegisterOpcodes`). The resolver handles both `RegisterOpcodes` and `RegisterPairOpcodes` for indirect store operations.
- Undocumented alias instructions (`edIm0Alias`, `edRetnAlias`) have `Unofficial=true` with empty opcode maps; the coverage test skips these unassemblable variants.
- `set.Set[T]` is used instead of `map[K]struct{}` per project linter policy.

### Scope Boundaries

**In scope:** Z80 architecture in `pkg/arch/z80`, Zilog-style core syntax, end-to-end assembly pipeline, CPU/system CLI flags.

**Out of scope:** Full assembler dialect compatibility beyond baseline Zilog syntax, systems not modeled in retrogolib (`msx`, `sms`), Game Boy-specific instruction behavior differences beyond the accepted Z80 subset.
