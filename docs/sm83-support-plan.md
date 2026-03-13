# SM83 (Sharp LR35902) Architecture Support Plan

## Context

Add Sharp SM83 (LR35902) assembler support to retroasm. The SM83 is the CPU used in the Nintendo Game Boy and Game Boy Color. The retrogolib package already provides complete instruction definitions (58 instruction variants, 7 addressing modes, 512 opcodes including CB-prefix) at `retrogolib/arch/cpu/sm83/`.

The SM83 is a simplified derivative of the Z80, but with significant differences that warrant a separate architecture implementation rather than a Z80 profile:
- Different register set (no IX/IY, no shadow registers, no I/O ports)
- Only CB prefix (no DD/ED/FD prefixes)
- Only 4 condition codes (NZ, Z, NC, C) vs Z80's 8
- SM83-unique instructions: LDH, STOP, SWAP, LD (HL+), LD (HL-)
- Different instruction grouping and opcode layout

## Key Design Decisions

- **Template**: Z80 architecture (`pkg/arch/z80/`) — same InstructionGroup pattern, same register-based opcode lookup, same resolver approach
- **Address width**: 16-bit
- **Generic type T**: `*InstructionGroup` (groups instruction variants by mnemonic name, like Z80)
- **OpcodeInfo fields**: Uses `Prefix` (0x00 or 0xCB), `Opcode`, `Size` — same structure as Z80
- **Register operand handling**: Uses `RegisterOpcodes` and `RegisterPairOpcodes` maps from retrogolib
- **Instruction building**: Scan all instruction variables from retrogolib SM83 package, group by mnemonic name into InstructionGroups
- **No profile system**: SM83 has no subsets (unlike Z80's gameboy/strict profiles)

## Architecture Details

### Addressing Modes (7)
| Mode | Description | Example |
|------|-------------|---------|
| ImpliedAddressing | No operand | `NOP`, `RET` |
| RegisterAddressing | Register operand | `INC B`, `LD A,B` |
| ImmediateAddressing | 8-bit or 16-bit literal | `LD A,$42`, `LD BC,$1234` |
| ExtendedAddressing | 16-bit absolute address | `JP $8000`, `CALL $1234` |
| RegisterIndirectAddressing | Memory via register | `LD A,(HL)`, `JP (HL)` |
| RelativeAddressing | Signed 8-bit branch offset | `JR label`, `JR NZ,label` |
| BitAddressing | Bit position 0-7 | `BIT 3,B`, `SET 7,A` |

### Register Parameters
- **8-bit**: A, B, C, D, E, H, L
- **16-bit pairs**: AF, BC, DE, HL, SP
- **Indirect**: (HL), (BC), (DE)
- **Conditions**: NZ, Z, NC, C
- **RST vectors**: $00, $08, $10, $18, $20, $28, $30, $38
- **SM83-specific**: (HL+), (HL-), ($FF00+n), ($FF00+C), SP+e

### CB-Prefix Instructions
All rotate/shift/bit operations use 0xCB prefix: RLC, RRC, RL, RR, SLA, SRA, SWAP, SRL, BIT, RES, SET.
Each operates on 7 registers + (HL) indirect.

### SM83-Specific Instructions
- `STOP` — Stop CPU until button press (opcode 0x10, 2 bytes)
- `SWAP r` — Swap upper/lower nibbles (replaces Z80's SLL)
- `LDH (n),A` / `LDH A,(n)` — High memory access ($FF00+n)
- `LD (HL+),A` / `LD A,(HL+)` — Load with post-increment
- `LD (HL-),A` / `LD A,(HL-)` — Load with post-decrement
- `LD (C),A` / `LD A,(C)` — $FF00+C indirect
- `ADD SP,e` / `LD HL,SP+e` — SP offset operations

## Files to Create

### 1. `pkg/arch/sm83/sm83.go` (~80 lines)
Architecture entry point. Similar to Z80 pattern:
- `InstructionGroup` type grouping variants by mnemonic
- `New()` → `*config.Config[*InstructionGroup]`
- `newArchitecture()` → builds instruction groups from retrogolib SM83 definitions
- `AddressWidth()` → 16
- `Instruction(name)` → lookup in instruction groups map
- Delegates ParseIdentifier/AssignInstructionAddress/GenerateInstructionOpcode to sub-packages

### 2. `pkg/arch/sm83/parser/register.go` (~80 lines)
Register name lookup maps:
- `registerParamByName` — maps "a", "b", "bc", "hl", etc. to RegisterParam
- `conditionParamByName` — maps "nz", "z", "nc", "c" to condition RegisterParams
- `indirectRegisterParamsByName` — maps "bc", "de", "hl" to indirect RegisterParams
- `rstVectorByValue` — maps RST addresses ($00, $08, ..., $38) to RegisterParams

### 3. `pkg/arch/sm83/parser/instruction.go` (~350 lines)
Main parser, adapted from Z80 but simpler (no IX/IY, no ports):
- `ParseIdentifier()` — entry point, parses operands and resolves instruction
- `parseOperands()` — collect raw token operands
- `resolveInstruction()` — dispatch to 0/1/2 operand resolvers
- **No-operand**: implied instructions (NOP, RET, etc.)
- **Single operand**: register (INC B), immediate (ADD #n), indirect (JP (HL)), condition (RET NZ), RST vector
- **Two operands**: register-register (LD A,B), register-immediate (LD A,#n), register-indirect (LD A,(HL)), condition-address (JP NZ,addr), bit-register (BIT 3,A), HL+/HL- special forms, LDH, LD (C),A, LD SP,HL, LD (nn),SP, LD (nn),A, LD A,(nn), ADD HL,rr, ADD SP,e, LD HL,SP+e

### 4. `pkg/arch/sm83/assembler/address_assigning_step.go` (~40 lines)
Address assignment — simple lookup of OpcodeInfo.Size from resolved instruction.

### 5. `pkg/arch/sm83/assembler/generate_opcode_step.go` (~180 lines)
Opcode generation:
- Build base opcode bytes (prefix + opcode)
- Append operand bytes based on addressing mode:
  - ImpliedAddressing, RegisterAddressing → base bytes only
  - ImmediateAddressing → append 1 or 2 bytes based on instruction size
  - ExtendedAddressing → append 16-bit address (little-endian)
  - RelativeAddressing → calculate signed offset, append byte
  - BitAddressing → compute bit-encoded opcode
  - RegisterIndirectAddressing → base bytes only (may have immediate operand)

### 6. `pkg/arch/sm83/sm83_test.go` (~200 lines)
Integration tests:
- Implied instructions (NOP, HALT, RET, RETI, etc.)
- Register loads (LD A,B, LD HL,$1234)
- Immediate addressing (LD A,#$42, ADD A,#$10)
- Indirect addressing (LD A,(HL), LD (BC),A)
- Jump/branch (JP $8000, JR label, JP NZ,addr)
- CB-prefix instructions (RLC B, BIT 3,A, SWAP A)
- SM83-specific (LDH, LD (HL+),A, STOP)
- Stack operations (PUSH BC, POP AF)
- RST instructions (RST $38)

### 7. `pkg/arch/sm83/assembler/generate_opcode_step_test.go` (~100 lines)
Unit tests for opcode generation with mock assigner.

## Files to Modify

### 8. `cmd/retroasm/main.go`
- Add `cpuSM83 = string(arch.SM83)` constant
- Add to `supportedSystemsByCPU`: `cpuSM83: set.NewFromSlice([]string{systemGameBoy, systemGeneric})`
- Add to `defaultSystemByCPU`: `cpuSM83: systemGameBoy`
- Add to `defaultCPUBySystem`: update `systemGameBoy` to `cpuSM83` (SM83 is the correct CPU for Game Boy)
- Add case in `registerArchitectureForCPU` for `cpuSM83`
- Update error message CPU lists to include sm83
- Update `-cpu` flag help text
- Import `archsm83 "github.com/retroenv/retroasm/pkg/arch/sm83"`

### 9. `README.md`
- Add SM83 row to Architecture Support table
- Update cpu flag options
- Add usage example for Game Boy assembly with SM83

### 10. `docs/work-branch-changes.md`
- Add SM83 section documenting all new files

## Implementation Order

1. Create `pkg/arch/sm83/sm83.go` (architecture adapter + instruction groups)
2. Create `pkg/arch/sm83/parser/register.go` (register lookup)
3. Create `pkg/arch/sm83/parser/instruction.go` (parser + resolver)
4. Create `pkg/arch/sm83/assembler/address_assigning_step.go`
5. Create `pkg/arch/sm83/assembler/generate_opcode_step.go`
6. Modify `cmd/retroasm/main.go` (CLI registration)
7. Create tests
8. Run linter, fix any issues
9. Update README.md and docs/work-branch-changes.md

## Verification

1. `go build ./...` — compiles without errors
2. `go test ./pkg/arch/sm83/...` — all tests pass
3. `go test ./cmd/retroasm/...` — CLI tests pass
4. `go test ./...` — full suite passes
5. `golangci-lint run ./...` — no linter violations
