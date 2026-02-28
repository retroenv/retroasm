# Z80 Architecture Support Plan (Revised)

## Overview

This plan adds Z80 assembler support to retroasm with an implementation order that matches the current codebase constraints. It focuses on a working, testable assembler path first, then CLI/system expansion.

## Progress

- Completed on February 28, 2026: Phase 0 (AST/assembler plumbing for typed and multi-operand instruction arguments).
- Completed on February 28, 2026: Phase 1 (Z80 architecture adapter and instruction grouping).
- Next implementation target: Phase 2 (operand classifier and resolver).

## Scope

### In Scope (first implementation)

- Z80 architecture support in `pkg/arch/z80`
- Parsing Zilog-style core syntax (`LD A,42`, `JP NZ,label`, `BIT 3,A`, `(IX+5)`)
- End-to-end assembly pipeline support (parse -> address assign -> opcode generation)
- CPU flag `-cpu z80` in CLI
- System flag compatibility for systems currently known by retrogolib (`generic`, `zx-spectrum`, `gameboy`)

### Out of Scope (initially)

- Full assembler dialect compatibility beyond baseline Zilog syntax
- New systems not currently modeled in retrogolib system enum (`msx`, `sms`)
- Game Boy specific instruction behavior differences beyond accepted Z80 subset

## Codebase Reality Checks (must be reflected in plan)

1. `ast.Instruction` currently has one `Argument` field. Z80 needs multi-operand semantics.
2. `pkg/assembler/parse_ast_nodes.go` now accepts scalar arguments (`ast.Number`, `ast.Label`, `ast.Identifier`) and typed/multi-operand wrappers (`ast.InstructionArgument`, `ast.InstructionArguments`).
3. `addressAssign.ArgumentValue` resolves only scalar/reference values. Z80 needs structured operand metadata.
4. `cmd/retroasm/main.go` and `pkg/retroasm/default.go` are effectively hard-wired to 6502/NES behavior today.
5. retrogolib opcode sources are:
   - `z80.Opcodes`
   - `z80.EDOpcodes`
   - `z80.DDOpcodes`
   - `z80.FDOpcodes`
   - CB, DDCB, and FDCB families are exposed via instruction vars (not a standalone `OpcodesCB` array).
6. `pkg/arch/z80/z80.go` now provides an architecture adapter with mnemonic grouping lookup and placeholder parse/address/opcode hooks for later phases.

## Architecture Decisions

### 1) Keep `arch.Architecture[T]` unchanged

No interface change required. Z80 uses `T = *InstructionGroup`:

```go
type InstructionGroup struct {
    Name     string
    Variants []*z80.Instruction
}
```

### 2) Add type-safe Z80 operand payload

Do not encode operands as strings. Use a typed Z80 argument model that can represent:

- register and condition params (`z80.RegisterParam`)
- optional immediate/address/relative/displacement expression nodes
- bit index for bit operations

This payload is stored in `ast.Instruction.Argument` and converted in `parse_ast_nodes` to assembler-internal data.

### 3) Build instruction groups from opcode tables plus manual CB-family additions

Group instruction pointers by mnemonic, deduplicated by pointer identity:

- from `Opcodes`, `EDOpcodes`, `DDOpcodes`, `FDOpcodes`
- plus `CBRlc`, `CBRrc`, `CBRl`, `CBRr`, `CBSla`, `CBSra`, `CBSll`, `CBSrl`, `CBBit`, `CBRes`, `CBSet`
- plus `DdcbShift`, `DdcbBit`, `DdcbRes`, `DdcbSet`, `FdcbShift`, `FdcbBit`, `FdcbRes`, `FdcbSet`

## Phased Implementation

## Phase 0: Foundation and Plumbing (Completed)

Files:

- `pkg/parser/ast/` (new operand node type(s))
- `pkg/assembler/parse_ast_nodes.go`

Tasks:

- Add AST node(s) for multi-operand instruction arguments.
- Extend AST-to-assembler conversion to accept the new Z80 operand node(s).
- Keep 6502 behavior unchanged.

Completed result:

- Added AST node types for typed arguments and multi-operand argument lists.
- Extended AST-to-assembler conversion to accept and preserve typed/multi-operand argument payloads.
- Added unit tests for copy behavior and parse conversion paths.

## Phase 1: Z80 Architecture Adapter (Completed)

Files:

- `pkg/arch/z80/z80.go`

Tasks:

- Implement `arch.Architecture[*InstructionGroup]`:
  - `AddressWidth() int` -> `16`
  - `Instruction(name string) (*InstructionGroup, bool)`
  - `ParseIdentifier(...)`
  - `AssignInstructionAddress(...)`
  - `GenerateInstructionOpcode(...)`
- Build and cache instruction groups at init.

Definition of done:

- Package compiles.
- Instruction lookup resolves known mnemonics with non-empty variants.

Completed result:

- Added `pkg/arch/z80/z80.go` implementing `arch.Architecture[*InstructionGroup]`.
- Added instruction group indexing from `Opcodes`, `EDOpcodes`, `DDOpcodes`, `FDOpcodes`.
- Added explicit CB and indexed-bit instruction family inclusion for complete mnemonic grouping.
- Added `pkg/arch/z80/z80_test.go` coverage for lookup behavior, case-insensitive keys, and presence of CB/indexed-bit variants.

## Phase 2: Operand Classifier + Resolver

Files:

- `pkg/arch/z80/parser/register.go`
- `pkg/arch/z80/parser/instruction.go`
- `pkg/arch/z80/parser/resolver.go`

Tasks:

- Parse and classify registers/conditions/indirect/indexed forms.
- Resolve `C` ambiguity by mnemonic+position context (condition list vs register).
- Resolve mnemonic+operands -> exact `*z80.Instruction` variant + selected param(s).

Minimum instruction slice for first green path:

- implied (`NOP`, `RET`)
- reg/reg (`LD A,B`)
- reg/immediate (`LD A,n`, `LD HL,nn`)
- relative jump (`JR e`, `JR NZ,e`)
- extended jump/call (`JP nn`, `JP NZ,nn`, `CALL nn`)

Definition of done:

- Parser unit tests pass for the minimum slice.
- Resolver deterministically returns one variant or a clear error.

## Phase 3: Address Assignment

Files:

- `pkg/arch/z80/assembler/address_assigning_step.go`

Tasks:

- Set instruction address from program counter.
- Determine size from resolved opcode info (not from operand value heuristics).
- Store finalized size/addressing for opcode generation.

Definition of done:

- Address assignment tests pass for 1/2/3/4-byte instructions.

## Phase 4: Opcode Generation (Core)

Files:

- `pkg/arch/z80/assembler/generate_opcode_step.go`

Tasks:

- Emit opcode bytes using resolved instruction metadata.
- Support prefix chains:
  - none
  - single-prefix (`CB`, `DD`, `ED`, `FD`)
  - two-prefix indexed bit forms (`DD CB d op`, `FD CB d op`)
- Emit operand bytes for immediate/extended/relative/displacement forms.
- Relative offsets use:

```text
offset = target - (ins.Address() + ins.Size())
```

Definition of done:

- Opcode tests pass for baseline matrix below.

Baseline verification matrix:

- `NOP` -> `00`
- `LD BC,$1234` -> `01 34 12`
- `LD A,42` -> `3E 2A`
- `JR label` forward/backward in range
- `BIT 3,A` -> `CB 5F`
- `LD A,(IX+5)` and `LD (IY-3),A`

## Phase 5: Extended Instruction Coverage

Tasks:

- Expand resolver/opcode coverage to all instruction families in grouped variants.
- Add CB/ED/DD/FD family coverage and edge conditions.
- Decide and implement behavior for undocumented ops (include or reject explicitly).

Definition of done:

- Coverage tests generated from instruction groups ensure no silent gaps in supported subset.

## Phase 6: CLI and Runtime Integration

Files:

- `cmd/retroasm/main.go`
- `pkg/retroasm/default.go` (or equivalent architecture selection path)

Tasks:

- Add `z80` CPU option in validation logic.
- Allow `-system gameboy`, `-system zx-spectrum`, and `-system generic` for Z80 mode.
- Remove hard-coded m6502 assembly path so selected architecture is actually used.

Definition of done:

- CLI can assemble a small Z80 input end-to-end.
- Existing 6502 CLI behavior remains unchanged.

## Testing Strategy

## Unit Tests

- parser classification and resolver (including `C` condition/register ambiguity)
- address assignment for mixed instruction sizes
- opcode generation for each addressing family

## Integration Tests

- `tests/z80/basic.asm` core instruction smoke test
- `tests/z80/indexed.asm` IX/IY displacement coverage
- `tests/z80/branches.asm` relative range/overflow checks

## Regression Requirements

- Every bug fix adds a focused regression test.
- Use `github.com/retroenv/retrogolib/assert`.
- Use `t.Context()` in tests.

## Key Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Multi-operand AST mismatch with current pipeline | High | Implement Phase 0 first; do not start resolver/opcode work before it |
| Wrong opcode table assumptions (CB vs ED/DD/FD) | High | Use actual retrogolib exports and add tests for table completeness |
| Prefix-chain bugs (`DD CB`, `FD CB`) | High | Explicit encoding path and dedicated integration cases |
| CLI still ignores selected architecture | High | Make architecture selection refactor explicit in Phase 6 |
| Ambiguous operand parsing (`C`, `(IX+d)`) | Medium | Context-aware parser rules + table-driven tests |

## Recommended Execution Order

1. Phase 0 (AST/assembler plumbing)
2. Phase 1 (adapter + instruction grouping)
3. Phase 2 (parser/resolver minimum slice)
4. Phase 3 (address assignment)
5. Phase 4 (opcode generation core)
6. Phase 5 (coverage expansion)
7. Phase 6 (CLI/runtime integration)

This order gets a small but real end-to-end Z80 path working early, then scales coverage safely.
