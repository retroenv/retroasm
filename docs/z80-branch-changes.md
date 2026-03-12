# Z80 Support -- Development History

This document is a historical reference for the Z80 architecture implementation in retroasm.
The Z80 branch has been merged to `main`. For current file tracking, see
[work-branch-changes.md](work-branch-changes.md).

**Scope:** ~50 files changed, ~8700 insertions, ~150 deletions across three categories:
1. Shared infrastructure -- architecture-agnostic pipeline extensions
2. Z80 implementation -- new packages under `pkg/arch/z80/`
3. Tests and fixtures -- unit tests, integration tests, and assembly fixtures

---

## 1. Shared Infrastructure Changes

These changes made the assembler pipeline generic enough to support any multi-operand
architecture, independent of Z80-specific code.

### AST Extensions (`pkg/parser/ast/`)

| File | Description |
|------|-------------|
| `expression.go` | `Expression` node wrapping `*expression.Expression` for symbolic operands |
| `instruction_argument.go` | `InstructionArgument` (single typed payload) and `InstructionArguments` (multi-operand list) |
| `instruction_argument_test.go` | Copy semantics tests for both node types |
| `node_test.go` | Added `TestExpression_Copy` |

### Assembler Pipeline (`pkg/assembler/`)

| File | Change |
|------|--------|
| `parse_ast_nodes.go` | Extended `convertInstructionArgument` to handle `InstructionArgument` and `InstructionArguments` |
| `address_assigning_step.go` | Added `ast.Number`, `ast.Label`, `ast.Identifier`, `ast.Expression` cases; added `argumentExpressionValue` for PC-relative expressions; added overflow-safe `applyInt64Offset`/`applyUint64Offset` |
| `generate_opcode_step.go` | Propagated `arch` and `programCounter` into address assigner for expression evaluation during opcode generation |

### High-Level API (`pkg/retroasm/default.go`)

Refactored from hard-coded M6502 dispatch to architecture-agnostic dispatch:
- `resolveArchitectureConfig()` selects the registered architecture (backward-compatible M6502 default)
- Generic helpers: `assembleASTWithConfig[T]`, `assembleTextWithConfig[T]`, `readAssemblerConfig[T]`, `applyBaseAddress[T]`
- Sentinel errors: `errAmbiguousArchitecture`, `errArchitectureAdapterMismatch`, `errArchitectureNotRegistered`, `errUnsupportedArchitectureConfig`

### CLI (`cmd/retroasm/main.go`)

- Replaced single-CPU constants with lookup tables (`supportedSystemsByCPU`, `defaultSystemByCPU`, `defaultCPUBySystem`)
- Added `-z80-profile` flag (values: `default`, `strict-documented`, `gameboy-z80-subset`)
- Refactored architecture validation into a pipeline of small functions
- Added `registerArchitectureForCPU` dispatch

---

## 2. Z80 Implementation

All files under `pkg/arch/z80/`. Uses `*InstructionGroup` as the generic type parameter,
grouping instruction variants by mnemonic name.

### Architecture Entry Point

| File | Description |
|------|-------------|
| `z80.go` | Builds instruction groups from retrogolib opcode tables (base, CB, DD, ED, FD families). 16-bit address width. Case-insensitive lookup. |
| `options.go` | `WithProfile(kind)` functional option for instruction set filtering |

### Parser (`pkg/arch/z80/parser/`)

| File | Description |
|------|-------------|
| `instruction.go` | Entry points `ParseIdentifier` and `ParseIdentifierWithProfile`. Handles all operand forms: registers, conditions, indexed `(IX+d)`, indirect `(HL)`, parenthesized value `(nn)`, expressions, port `(C)`. |
| `register.go` | Table-driven operand classification: 16 registers, 8 conditions, 7 indirect forms, 10 numeric operands (RST vectors, IM modes) |
| `resolver.go` | Top-level dispatcher and shared helpers for no-operand, single-operand, two-operand resolution |
| `resolver_single_operand.go` | Register, immediate, indirect, indexed, numeric single-operand paths |
| `resolver_two_operand.go` | Register-pair dispatch and special-pair instructions |
| `resolver_indirect.go` | Indirect load/store and indirect-immediate patterns |
| `resolver_extended.go` | 16-bit memory operations (`LD (nn),rr`, `LD rr,(nn)`) with HL encoding preference |
| `resolver_port.go` | Port I/O: `IN r,(C)`, `OUT (C),r`, `IN A,(n)`, `OUT (n),A` |
| `resolver_indexed.go` | `LD r,(IX+d)`, `LD (IX+d),r` indexed register operations |
| `resolver_value.go` | `LD r,n`, `ADD A,n` value-register and `BIT`/`RES`/`SET` bit operations |
| `resolver_diagnostics.go` | Diagnostic errors for C condition/register ambiguity, form mismatches, direction mismatches |
| `doc.go` | Package documentation |

### Assembler (`pkg/arch/z80/assembler/`)

| File | Description |
|------|-------------|
| `address_assigning_step.go` | Extracts `ResolvedInstruction`, looks up `OpcodeInfo`, sets address/size. Opcode lookup priority: single register, register pair, addressing mode, single-entry fallback. |
| `generate_opcode_step.go` | Byte emission by addressing family (Implied, Immediate, Extended, Relative, RegisterIndirect, Port, Bit). Handles CB bit encoding and indexed bit encoding. |
| `doc.go` | Package documentation |

### Profile (`pkg/arch/z80/profile/`)

| Kind | String | Behavior |
|------|--------|----------|
| `Default` | `default` | All opcodes including undocumented |
| `StrictDocumented` | `strict-documented` | Rejects undocumented opcodes (SLL, IXH/IXL variants, specific ED/CB ranges) |
| `GameBoySubset` | `gameboy-z80-subset` | Rejects DD/ED/FD prefixes and DJNZ/EX/EXX/IN/OUT mnemonics |

Validation occurs at parse time for immediate error messages. Detection uses instruction
pointer identity, mnemonic matching, opcode byte ranges, and prefix key set lookups.

---

## 3. Tests

### Unit Tests

| File | Coverage |
|------|----------|
| `z80_test.go` | Instruction lookup, case-insensitive keys, CB/indexed-bit variant presence |
| `assembler/address_assigning_step_test.go` | 1-4 byte instruction size assignment, error paths |
| `assembler/generate_opcode_step_test.go` | Opcode emission matrix, boundary values, error paths |
| `assembler/coverage_test.go` | Exhaustive: synthesizes valid `ResolvedInstruction` per opcode variant from all tables |
| `parser/instruction_test.go` | All operand forms, all resolver paths, error cases, diagnostic quality assertions |
| `parser/register_test.go` | Register/condition/indirect/indexed classification |
| `parser/fuzz_test.go` | Property-based determinism: same token stream produces same outcome |
| `parser/profile_test.go` | Profile-gated instruction acceptance/rejection |
| `profile/profile_test.go` | Parse/validation for all three profiles |

### Integration Tests

| File | Coverage |
|------|----------|
| `cmd/retroasm/z80_fixture_test.go` | End-to-end fixture assembly with byte-accurate assertions; 54 inline tests covering all resolver paths |
| `cmd/retroasm/main_test.go` | Extended CPU/system validation matrix, architecture registration |

### Assembly Fixtures (`tests/z80/`)

| Fixture | Purpose |
|---------|---------|
| `basic.asm` | Core instructions: NOP, LD, HALT, arithmetic |
| `branches.asm` | JR, JP, CALL, RET with conditions (forward/backward) |
| `branches_overflow.asm` | JR out-of-range displacement error regression |
| `compatibility.asm` | Mixed control flow edge cases |
| `expressions.asm` | Symbolic expressions: `JP target+delta`, `LD A,(IX+disp)` |
| `indexed.asm` | IX/IY prefix path: indexed loads, BIT, JP, IM, RST |
| `indexed_boundaries.asm` | Displacement edge values: -128, +127 |
| `io_extended.asm` | Extended memory and port I/O instructions |
| `offsets.asm` | Tokenized offsets: `JP target+1`, `IN A,($10+1)` |
| `offsets_chained.asm` | Chained offsets: `JP target+2-1` |
| `profile_gameboy_subset.asm` | Positive: Game Boy profile acceptance |
| `profile_gameboy_subset_rejects.asm` | Negative: Game Boy profile rejection (IN instruction) |
| `profile_strict_documented.asm` | Positive: strict profile acceptance |
| `profile_strict_documented_rejects.asm` | Negative: strict profile rejection (SLL instruction) |

---

## 4. retrogolib Changes (External Dependency)

The following changes were made to `retrogolib/arch/cpu/z80/` to support the assembler:

| File | Change |
|------|--------|
| `instruction.go` | Moved `LD (HL),r` from `LdIndirect.RegisterOpcodes` to `LdReg8.RegisterPairOpcodes` |
| `instruction_dd.go` | Added `DdLdSpIX` variant (`LD SP,IX`, DD F9) |
| `instruction_ed.go` | Added undocumented alias instructions (`edIm0Alias`, `edRetnAlias`) for emulator decoding |
| `instruction_fd.go` | Added `FdLdSpIY` variant (`LD SP,IY`, FD F9) |
| `opcode.go` | Added DD 0xF9 and FD 0xF9 opcode table entries |
| `emulation_dd.go` | Added `ddLdSpIX` emulation function |
| `emulation_fd.go` | Added `fdLdSpIY` emulation function |

---

## 5. Dependency Layers

Changes were organized into four independent layers, each depending only on layers below it:

```
Layer 4: CLI (cmd/retroasm/main.go)
Layer 3: Z80 packages (pkg/arch/z80/)
Layer 2: High-level API dispatch (pkg/retroasm/default.go)
Layer 1: Shared assembler + AST extensions (pkg/assembler/, pkg/parser/ast/)
```

Layer 1 was fully extractable to `main` without pulling in any Z80-specific code, making the
pipeline generic for any multi-operand architecture.
