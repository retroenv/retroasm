# Z80 Architecture Support Plan

## Overview

This document outlines the plan for adding Z80 assembler support to retroasm. The Z80 is a 16-bit address bus / 8-bit data bus processor used in the ZX Spectrum, Amstrad CPC, MSX computers, Game Boy (modified Z80), and Sega Master System/Game Gear.

retrogolib already provides comprehensive Z80 CPU definitions including instruction tables, addressing modes, register parameters, opcode maps (base + CB/DD/ED/FD prefixed), and bidirectional opcode/instruction lookup. This plan focuses exclusively on the retroasm assembler-side implementation.

## Key Differences from 6502

The Z80 presents several challenges not present in the 6502 implementation:

| Aspect | 6502 | Z80 |
|--------|------|-----|
| **Address width** | 16-bit | 16-bit |
| **Instruction lookup** | `map[string]*Instruction` by name | Multiple `*Instruction` vars per mnemonic (e.g., `LdImm8`, `LdReg8`, `LdReg16`, `LdIndirect`, `LdExtended`) |
| **Opcode prefixes** | None | CB, DD, ED, FD prefix bytes |
| **Max opcode size** | 3 bytes | 4 bytes (prefix + opcode + 2 operand bytes) |
| **Register operands** | Accumulator only | 8-bit regs (A,B,C,D,E,H,L), 16-bit pairs (BC,DE,HL,SP,AF,IX,IY) |
| **Addressing modes** | 13 modes | 8 modes with register variants via `RegisterOpcodes` and `RegisterPairOpcodes` |
| **Instruction variants** | 1 Instruction per mnemonic | Many: same mnemonic maps to different `*Instruction` based on operands |
| **Conditions** | Implicit in instruction name (BEQ, BNE) | Explicit condition codes as operands (JP NZ,nn / JR C,e) |
| **Bit operations** | None | BIT/SET/RES with bit number + register |
| **I/O ports** | Memory-mapped only | Dedicated IN/OUT with port addressing |
| **Index registers** | None | IX, IY with displacement (IX+d, IY+d) |

### Instruction Disambiguation Challenge

The most significant difference is instruction disambiguation. The 6502 has a simple `Instructions map[string]*Instruction` where each mnemonic maps to exactly one Instruction struct. The Z80 has multiple Instruction structs per mnemonic:

- `LD` alone maps to: `LdImm8`, `LdReg8`, `LdReg16`, `LdIndirect`, `LdExtended`, `LdIndirectImm`, `LdSp`, plus DD/ED/FD-prefixed variants
- `INC` maps to: `IncReg8`, `IncReg16`, `IncIndirect`, plus DD-prefixed variants
- `ADD` maps to: `AddA`, `AddHl`, plus DD-prefixed variants
- `JP` maps to: `JpAbs`, `JpCond`, `JpIndirect`

The parser must determine which Instruction struct to use based on the operand types parsed from the token stream.

## Architecture

### Package Structure

```
pkg/arch/z80/
├── z80.go                    # Architecture adapter (implements arch.Architecture[T])
├── parser/
│   ├── instruction.go        # Z80 instruction parsing and operand analysis
│   ├── register.go           # Register name parsing and classification
│   ├── addressing.go         # Addressing mode determination
│   └── resolver.go           # Instruction disambiguation (mnemonic + operands → *Instruction)
└── assembler/
    ├── address_assigning_step.go   # Z80-specific address assignment
    └── generate_opcode_step.go     # Z80-specific opcode generation (including prefixes)
```

### Type Parameter

The `arch.Architecture[T]` generic interface requires a type parameter. For the 6502, `T` is `*m6502.Instruction`. For the Z80, `T` cannot simply be `*z80.Instruction` because a single mnemonic maps to multiple Instruction vars.

**Approach**: Define a Z80-specific wrapper type that represents a resolved instruction lookup result:

```go
// InstructionGroup holds all z80.Instruction variants that share a mnemonic.
type InstructionGroup struct {
    Name     string
    Variants []*z80.Instruction
}
```

The `Architecture.Instruction(name string)` method returns an `InstructionGroup` containing all variants for that mnemonic. The parser's `ParseIdentifier` then disambiguates by examining operands.

### Instruction Lookup Table

Build a `map[string]*InstructionGroup` at init time by iterating the `z80.Opcodes` array (and CB/DD/ED/FD arrays) and grouping by `Instruction.Name`. This avoids hardcoding the mapping and stays synchronized with retrogolib.

## Implementation Phases

### Phase 1: Core Z80 Architecture Adapter

**Files**: `pkg/arch/z80/z80.go`

Implement the `arch.Architecture[*InstructionGroup]` interface:

- `AddressWidth() int` → returns 16
- `Instruction(name string) (*InstructionGroup, bool)` → lookup in instruction group map
- `ParseIdentifier(p arch.Parser, ins *InstructionGroup) (ast.Node, error)` → delegate to parser package
- `AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error)` → delegate to assembler package
- `GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error` → delegate to assembler package

Build the instruction group map from `z80.Opcodes`, `z80.OpcodesCB`, `z80.OpcodesDD`, `z80.OpcodesED`, `z80.OpcodesFD`.

### Phase 2: Register and Operand Parsing

**Files**: `pkg/arch/z80/parser/register.go`

Parse register names from tokens and classify them:

- **8-bit registers**: A, B, C, D, E, H, L, I, R
- **16-bit register pairs**: AF, BC, DE, HL, SP, IX, IY
- **Indirect registers**: (HL), (BC), (DE), (SP), (IX+d), (IY+d)
- **Condition codes**: NZ, Z, NC, C, PO, PE, P, M
- **Shadow registers**: AF' (EX AF,AF')

Map parsed register names to `z80.RegisterParam` constants.

Handle the `C` ambiguity: `C` is both a register and a condition code. Context determines meaning:
- `JP C,nn` → condition code
- `LD A,C` → register
- `IN A,(C)` → port register

### Phase 3: Instruction Parser

**Files**: `pkg/arch/z80/parser/instruction.go`, `pkg/arch/z80/parser/addressing.go`, `pkg/arch/z80/parser/resolver.go`

Parse Z80 instruction syntax and resolve to specific `*z80.Instruction` + operand data.

#### Operand Patterns to Recognize

1. **No operands**: `NOP`, `HALT`, `RET`, `EI`, `DI`, `EXX`, `CCF`, `SCF`, `CPL`, `DAA`, `RLA`, `RRA`, `RLCA`, `RRCA`
2. **Single register**: `INC B`, `DEC HL`, `PUSH BC`, `POP DE`
3. **Single immediate**: `RST 08H`, `IM 1`
4. **Single address/label**: `JP nn`, `CALL nn`, `JR e`
5. **Condition + address**: `JP NZ,nn`, `CALL Z,nn`, `JR NC,e`, `RET Z`
6. **Register, register**: `LD A,B`, `ADD A,C`, `EX DE,HL`
7. **Register, immediate**: `LD A,42`, `LD BC,1234h`, `ADD A,5`
8. **Register, indirect**: `LD A,(HL)`, `LD B,(IX+5)`
9. **Register, address**: `LD A,(nn)`, `LD HL,(nn)`
10. **Indirect, register**: `LD (HL),A`, `LD (IX+5),B`
11. **Indirect, immediate**: `LD (HL),n`
12. **Address, register**: `LD (nn),A`, `LD (nn),HL`
13. **Bit, register**: `BIT 3,A`, `SET 5,(HL)`, `RES 0,B`
14. **Port, register / register, port**: `OUT (n),A`, `IN A,(n)`, `OUT (C),A`, `IN A,(C)`

#### Disambiguation Strategy

After parsing operands, match against instruction variants:

```
resolve(mnemonic, dst_operand, src_operand) → *z80.Instruction
```

1. Parse the mnemonic to get the `InstructionGroup`
2. Parse first operand (if any) → classify as register/immediate/indirect/condition/address/bit
3. Parse second operand (if any) → classify similarly
4. Match the operand pattern against each variant's `Addressing`, `RegisterOpcodes`, and `RegisterPairOpcodes` maps
5. Return the matching variant or error

#### AST Node Representation

Reuse existing `ast.Instruction` node. The `Addressing` field stores the Z80 addressing mode. The `Argument` field stores the operand(s). For two-operand instructions, a new AST node or a pair-wrapper may be needed.

**Option A**: Store a `z80.RegisterParam` (or pair) in the Argument, encoding the full operand selection. The opcode generator can then look up `RegisterOpcodes[param]` or `RegisterPairOpcodes[[2]RegisterParam{dst, src}]` directly.

**Option B**: Extend `ast.Instruction` to support two arguments. This is cleaner but requires modifying shared AST code.

**Recommended**: Option A for the initial implementation — it avoids modifying shared code and maps directly to the retrogolib lookup structures. Store a struct with the resolved `RegisterParam` value(s) plus any immediate/address value as the Argument.

### Phase 4: Address Assignment

**Files**: `pkg/arch/z80/assembler/address_assigning_step.go`

Assign addresses to Z80 instructions:

1. Look up instruction size from the resolved `OpcodeInfo.Size`
2. Handle prefix bytes: CB/DD/ED/FD prefixed instructions are larger
3. Handle IX+d/IY+d displacement byte (adds 1 byte to size)
4. For relative jumps (JR, DJNZ): size is always 2

No addressing mode disambiguation is needed (unlike 6502's zero-page vs absolute) because Z80 instruction sizes are determined by the opcode, not the operand value.

### Phase 5: Opcode Generation

**Files**: `pkg/arch/z80/assembler/generate_opcode_step.go`

Generate machine code bytes for each instruction:

1. **Prefix emission**: If `OpcodeInfo.Prefix != 0`, emit the prefix byte first
2. **Opcode emission**: Emit `OpcodeInfo.Opcode`
3. **Operand emission** (varies by addressing mode):
   - `ImpliedAddressing`: no operand bytes
   - `RegisterAddressing`: no operand bytes (register encoded in opcode)
   - `ImmediateAddressing` (8-bit): 1 operand byte
   - `ImmediateAddressing` (16-bit): 2 operand bytes (little-endian)
   - `ExtendedAddressing`: 2 address bytes (little-endian)
   - `RegisterIndirectAddressing`: no operand bytes (except IX+d/IY+d: 1 displacement byte)
   - `RelativeAddressing`: 1 signed offset byte (calculated from PC after instruction)
   - `BitAddressing`: no extra bytes (bit encoded in opcode)
   - `PortAddressing`: 1 port byte (for immediate port addressing)

#### Relative Jump Offset Calculation

Same principle as 6502 but the Z80 offset is relative to the address after the instruction (PC + 2 for all relative jumps):

```
offset = target_address - (instruction_address + 2)
```

Range: -128 to +127 (signed byte).

### Phase 6: CLI and Configuration Integration

**Files**: `cmd/retroasm/main.go`, `pkg/assembler/config/`

1. Register Z80 architecture in CLI alongside 6502
2. Add system mappings:
   - `zx-spectrum` → Z80
   - `msx` → Z80
   - `gameboy` → Z80 (note: Game Boy has a modified Z80 — may need a separate adapter later)
   - `sms` → Z80 (Sega Master System)
3. Add default memory configurations for each target system:
   - ZX Spectrum: 48K/128K memory layout
   - MSX: slot-based memory mapping
   - Game Boy: ROM banks + RAM
4. Update CLI flags: `-cpu z80`, `-system zx-spectrum`

### Phase 7: Assembler Format Support

Determine which assembly syntax to support initially. Common Z80 assemblers:

| Assembler | Syntax Style | Notes |
|-----------|-------------|-------|
| **Zilog standard** | `LD A,42` | Official Z80 notation |
| **PASMO** | Zilog-compatible | Popular cross-assembler |
| **z80asm** | Zilog-compatible | Common in ZX Spectrum community |
| **RGBASM** | Game Boy specific | Modified syntax for GB |
| **WLA-DX** | Multi-architecture | Supports Z80 among others |
| **SjASMPlus** | Extended Zilog | Structs, LUA scripting |

**Initial target**: Zilog standard syntax (compatible with PASMO/z80asm). This covers the majority of Z80 assembly code.

Key syntax elements:
- `;` for comments (already supported)
- `$` or `0x` or `nnnnH` for hex numbers
- `%` or `0b` or `nnnnB` for binary numbers
- Register names are case-insensitive
- Parentheses for indirect addressing: `(HL)`, `(nn)`
- `+`/`-` displacement: `(IX+5)`, `(IY-3)`

### Phase 8: Testing

#### Unit Tests

- `pkg/arch/z80/parser/`: Test parsing of all operand patterns
- `pkg/arch/z80/assembler/`: Test opcode generation for all addressing modes
- Register disambiguation tests (C register vs C condition)
- IX/IY displacement parsing
- Prefix byte emission

#### Integration Tests

Create test assembly files in `tests/z80/`:

```asm
; tests/z80/basic.asm - Basic Z80 instruction test
    ORG $0000

    LD A, 42        ; immediate load
    LD B, A         ; register-to-register
    LD HL, $1234    ; 16-bit immediate
    LD (HL), A      ; indirect store
    LD A, (HL)      ; indirect load
    ADD A, B        ; register arithmetic
    SUB 10          ; immediate arithmetic
    JP $0100        ; absolute jump
    JR loop         ; relative jump
    CALL $0200      ; subroutine call
    RET             ; return
    PUSH BC         ; stack push
    POP DE          ; stack pop
    BIT 3, A        ; bit test
    SET 5, (HL)     ; bit set
    RES 0, B        ; bit reset
    RL C            ; rotate (CB-prefixed)
    IN A, ($FE)     ; port input
    OUT ($FE), A    ; port output
```

#### Opcode Verification Tests

For each instruction variant, verify the exact byte sequence against known-good assembler output. Use the `z80.Opcodes` table as the reference:

```go
func TestOpcodeGeneration(t *testing.T) {
    tests := []struct {
        source   string
        expected []byte
    }{
        {"NOP", []byte{0x00}},
        {"LD BC,$1234", []byte{0x01, 0x34, 0x12}},
        {"INC B", []byte{0x04}},
        {"LD A,42", []byte{0x3E, 0x2A}},
        {"BIT 3,A", []byte{0xCB, 0x5F}},
        // ... comprehensive table
    }
}
```

## Dependencies

### retrogolib Requirements

The following retrogolib Z80 exports are used:

- `z80.Instruction` struct and all instruction variables
- `z80.AddressingMode` constants
- `z80.RegisterParam` constants
- `z80.OpcodeInfo` struct
- `z80.Opcodes` array (base opcodes)
- `z80.OpcodesCB`, `z80.OpcodesDD`, `z80.OpcodesED`, `z80.OpcodesFD` arrays
- `z80.BranchingInstructions` set

### retroasm Core Changes

Minimal changes expected to core assembler code:

1. **No changes to `arch.Architecture[T]` interface** — the generic design handles Z80 naturally
2. **No changes to the 6-step pipeline** — all Z80-specific logic lives in the arch adapter
3. **Possible AST extension**: If two-operand support requires a new node type (Phase 3, Option B), add it to `parser/ast/`
4. **Lexer consideration**: Verify parentheses `()` and `+`/`-` in `(IX+d)` are tokenized correctly. The existing lexer handles parentheses and operators, so this should work without changes.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Instruction disambiguation complexity | High | Build comprehensive test table against z80.Opcodes; start with common instructions, add edge cases incrementally |
| `C` register/condition ambiguity | Medium | Use context-aware parsing: condition codes only valid after JP/JR/CALL/RET |
| IX/IY displacement parsing | Medium | Careful lexer integration for `(IX+expr)` syntax |
| Prefixed opcode emission | Medium | Validate prefix byte handling against z80.OpcodeInfo.Prefix field |
| Game Boy Z80 variant differences | Low | Defer Game Boy-specific handling to a later phase; focus on standard Z80 first |
| retrogolib Z80 instruction completeness | Low | retrogolib already has comprehensive Z80 support including CB/DD/ED/FD prefixed instructions |

## Suggested Implementation Order

1. **Phase 1**: Z80 architecture adapter skeleton — get it registered and compiling
2. **Phase 2**: Register parsing — foundational for everything else
3. **Phase 3**: Instruction parser for implied/register-only instructions (NOP, HALT, INC B, LD A,B)
4. **Phase 5**: Opcode generation for the same simple instructions — enables end-to-end testing early
5. **Phase 4**: Address assignment
6. **Phase 3 continued**: Extend parser to immediate, extended, relative, indirect addressing
7. **Phase 5 continued**: Extend opcode generation for remaining addressing modes
8. **Phase 3 continued**: CB/DD/ED/FD prefixed instruction parsing
9. **Phase 5 continued**: Prefixed opcode generation
10. **Phase 6**: CLI integration
11. **Phase 7**: Assembler format tuning
12. **Phase 8**: Comprehensive testing

This order prioritizes getting a minimal end-to-end flow working early, then expanding instruction coverage incrementally.
