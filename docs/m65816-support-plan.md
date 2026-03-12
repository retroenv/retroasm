# M65816 (WDC 65C816) Architecture Support

## Overview

retroasm supports the WDC 65C816 (65816) processor, a 16/24-bit extension of the 6502 used in the Super Nintendo Entertainment System (SNES/Super Famicom) and Apple IIGS.

## Architecture Details

- **Address width:** 24-bit (16 MB address space)
- **Byte order:** Little-endian
- **Instruction set:** 83 mnemonics, 256 opcodes
- **Addressing modes:** 21 modes (extending 6502 with indirect long, stack relative, block move, relative long)
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

## Current Limitations

- **Emulation mode only:** Initial implementation assumes 8-bit accumulator and index registers (M=1, X=1). The `BaseSize` field is used directly for instruction sizing.
- **16-bit mode deferred:** `.a8`/`.a16`/`.i8`/`.i16` directives for switching between 8-bit and 16-bit register widths are not yet implemented. When enabled, immediate addressing for accumulator instructions (LDA, ADC, etc.) would use 2 bytes instead of 1.

## Implementation

The implementation follows the established M6502 pattern:

- `pkg/arch/m65816/m65816.go` — Architecture entry point
- `pkg/arch/m65816/parser/addressing.go` — Addressing mode constants and disambiguation
- `pkg/arch/m65816/parser/instruction.go` — Instruction parser
- `pkg/arch/m65816/assembler/address_assigning_step.go` — Address assignment
- `pkg/arch/m65816/assembler/generate_opcode_step.go` — Opcode generation

## CLI Usage

```bash
# Assemble a 65816 program for SNES
retroasm -cpu 65816 -system snes -o game.sfc program.asm

# With generic system
retroasm -cpu 65816 -system generic -o program.bin program.asm
```
