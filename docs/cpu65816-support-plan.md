# CPU65816 (WDC 65C816) Architecture Support

## Overview

The `work2` branch contains a WDC 65C816 adapter. This reference was reviewed
on 2026-10-02. The adapter is absent from local `main` at `0e4317a` and is
excluded from the [current merge plan](work-branch-changes.md).

## Architecture Details

- **Address width:** 24-bit (16 MB address space)
- **Byte order:** Little-endian
- **Instruction definitions:** Supplied by the pinned retrogolib dependency
- **Addressing:** Includes indirect long, stack relative, block move, and relative long forms
- **Opcode size:** 1-4 bytes depending on addressing mode

## Addressing Modes

### Inherited from 6502 (renamed)
| Mode | Syntax | Description |
|------|--------|-------------|
| Implied | `NOP` | No operand |
| Accumulator | `ASL A` | Operates on accumulator |
| Immediate | `LDA #$42` | 8-bit literal (emulation mode) |
| Direct Page | `LDA $10` | 8-bit offset from DP register |
| Direct Page,X | `LDA $10,X` | DP + X index |
| Direct Page,Y | `LDX $10,Y` | DP + Y index |
| (Direct Page,X) | `LDA ($10,X)` | Pre-indexed indirect |
| (Direct Page),Y | `LDA ($10),Y` | Post-indexed indirect |
| Absolute | `LDA $1234` | 16-bit address |
| Absolute,X | `LDA $1234,X` | Absolute + X |
| Absolute,Y | `LDA $1234,Y` | Absolute + Y |
| Relative | `BNE label` | 8-bit signed branch offset |
| (Absolute) | `JMP ($1234)` | Indirect jump |

### New in 65816
| Mode | Syntax | Description |
|------|--------|-------------|
| (Direct Page) | `LDA ($10)` | DP indirect (no index) |
| [Direct Page] | `LDA [$10]` | Indirect long (24-bit pointer) |
| [Direct Page],Y | `LDA [$10],Y` | Indirect long + Y |
| Absolute Long | `JML $012345` | 24-bit address |
| Absolute Long,X | `LDA $012345,X` | 24-bit + X |
| (Absolute,X) | `JMP ($1234,X)` | Indexed indirect jump |
| [Absolute] | `JML [$1234]` | Indirect long jump |
| Stack Relative | `LDA $05,S` | Stack pointer + offset |
| (Stack,S),Y | `LDA ($05,S),Y` | Stack indirect + Y |
| Relative Long | `BRL label` | 16-bit signed branch offset |
| Block Move | `MVN $01,$02` | Bank-to-bank block copy |

## Address Size Prefixes

When an instruction supports both direct page and absolute addressing, the assembler disambiguates by value size. Explicit prefixes can force a specific mode:

- `z:` — Force direct page addressing: `LDA z:$10`
- `a:` — Force absolute addressing: `LDA a:$0010`
- `f:` — Force long addressing: `LDA f:$7E0010`

## Register-width state

`parser.DefaultState()` starts in native mode with 8-bit accumulator and index
widths. Parser streams own their state. `REP`, `SEP`, `CLC`, `SEC`, `XCE`,
`PLP`, and `RTI` update tracked width, carry, and emulation information.
Immediate encoding uses the selected width. Runtime-dependent state can make
a width unknown; the codec rejects operands that require an unknown width.

Use the stateful codec API for an explicit entry state. This is sequential
stream tracking, not control-flow analysis. `.a8`, `.a16`, `.i8`, and `.i16`
are not registered directives. x816 `.mem` and `.index` remain no-ops.

## Implementation

The implementation follows the established CPU6502 pattern:

- `pkg/arch/cpu65816/cpu65816.go` — Architecture entry point
- `pkg/arch/cpu65816/parser/addressing.go` — Addressing mode constants and disambiguation
- `pkg/arch/cpu65816/parser/instruction.go` — Instruction parser
- `pkg/arch/cpu65816/parser/state.go` — Entry state and instruction transitions
- `pkg/arch/cpu65816/parser/operand.go` — Owned typed operands
- `pkg/arch/cpu65816/parser/resolved.go` — Resolved operands and width selection
- `pkg/arch/cpu65816/parser/codec.go` — Typed build, validation, and formatting
- `pkg/arch/cpu65816/parser/symbol_rewrite.go` — Typed symbol rewriting
- `pkg/arch/cpu65816/assembler/address_assigning_step.go` — Address assignment
- `pkg/arch/cpu65816/assembler/generate_opcode_step.go` — Opcode generation

## CLI Usage

```bash
# Assemble a 65816 program for SNES
retroasm -cpu 65816 -system snes -c memory.cfg -o game.sfc program.asm

# With generic system
retroasm -cpu 65816 -system generic -c memory.cfg -o program.bin program.asm
```

Supply the memory layout and cartridge data. `-system snes` does not generate
a complete SNES cartridge header or memory map.

## Validation and extraction

`pkg/codec/cpu65816_test.go` contains stateful width, transition, independent
stream, relocation, and invalid-state tests. Architecture assembly tests are
in `pkg/arch/cpu65816/cpu65816_test.go` and its assembler subpackage.
No code tests were run for this documentation review.

A future extraction must include the state API, typed operands, encoder width
selection, and their tests together. CLI registration and user documentation
must follow the adapter. Run focused CPU/codec/CLI checks and the common code
gates on that candidate with the pinned dependency and no local replacement.
