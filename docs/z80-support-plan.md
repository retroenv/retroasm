# Z80 Architecture Support Plan (Revised)

## Overview

This plan adds Z80 assembler support to retroasm with an implementation order that matches the current codebase constraints. It focuses on a working, testable assembler path first, then CLI/system expansion.

## Progress

- Completed on February 28, 2026: Phase 0 (AST/assembler plumbing for typed and multi-operand instruction arguments).
- Completed on February 28, 2026: Phase 1 (Z80 architecture adapter and instruction grouping).
- Completed on February 28, 2026: Phase 2 (operand classifier and resolver minimum slice).
- Completed on February 28, 2026: Phase 3 (address assignment).
- Completed on February 28, 2026: Phase 4 (opcode generation core).
- Completed on February 28, 2026: Phase 5 (extended instruction coverage).
- Completed on March 1, 2026: Phase 6 (CLI/runtime integration).
- Completed on March 1, 2026: Phase 7 (integration fixtures and regression harness).
- Completed on March 1, 2026: Phase 8 (indexed and parenthesized operand parsing).
- Completed on March 1, 2026: Phase 9 (extended indirect/register and port I/O operand resolution).
- Completed on March 1, 2026: Phase 10 (tokenized offset operand parsing for value and parenthesized forms).
- Completed on March 1, 2026: Phase 11 (chained tokenized offset operand parsing).
- Completed on March 1, 2026: Phase 12 (expression-backed operand values and displacement support).
- Completed on March 1, 2026: Phase 13 (profile strictness and undocumented-op policy).
- Completed on March 1, 2026: Phase 14 (parser/resolver diagnostic quality pass).
- Next implementation target: Phase 15 (robustness and compatibility expansion).

## What Is Missing (Post-Phase 14)

The assembler path is now functional and broadly covered, but these gaps remain for a production-ready Z80 frontend:

1. Robustness testing can be strengthened.
   - Missing: fuzz/property tests and a wider compatibility fixture corpus.

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
4. `cmd/retroasm/main.go` now validates and defaults CPU/system combinations for both 6502 and Z80, and `pkg/retroasm/default.go` now assembles using the registered architecture config rather than a hard-wired m6502 path.
5. retrogolib opcode sources are:
   - `z80.Opcodes`
   - `z80.EDOpcodes`
   - `z80.DDOpcodes`
   - `z80.FDOpcodes`
   - CB, DDCB, and FDCB families are exposed via instruction vars (not a standalone `OpcodesCB` array).
6. `pkg/arch/z80/z80.go` now provides an architecture adapter with mnemonic grouping lookup, parser delegation, address assignment delegation, and opcode generation delegation.
7. `pkg/arch/z80/parser/` now resolves the minimum instruction slice and emits typed `ResolvedInstruction` payloads via `ast.InstructionArgument`.
8. `pkg/arch/z80/assembler/address_assigning_step.go` now computes instruction size from resolved opcode metadata (including prefixed 4-byte opcodes), without operand-value size heuristics.
9. `pkg/arch/z80/assembler/generate_opcode_step.go` now emits opcode bytes for core addressing families, including single-prefix (`CB`, `DD`, `ED`, `FD`) and indexed bit prefix chains (`DD CB`, `FD CB`).
10. Phase 5 coverage tests now generate resolved instructions from the Z80 opcode variant inventory (`Opcodes`, `EDOpcodes`, `DDOpcodes`, `FDOpcodes`, plus CB/indexed-bit families) to guard against silent encoding gaps.
11. Z80 parser operand handling now supports parenthesized indirect/register forms and indexed displacement forms used by IX/IY instruction families.
12. Z80 resolver matching now supports parenthesized extended indirect/register transfer forms (`ld r,(nn)`, `ld (nn),r`) and both port I/O families (`in a,(n)` / `out (n),a` and `in r,(c)` / `out (c),r`) with opcode-direction disambiguation for ambiguous `ld` variants.
13. Z80 operand parsing now supports tokenized offset expressions for instruction values and parenthesized values (`label+1`, `label-1`, `(label+1)`, `($10+1)`), while preserving indexed IX/IY displacement parsing.
14. Z80 operand parsing now supports chained tokenized offset expressions (`label+3-1`, `(label+3-1)`, `($10+3-1)`) with accumulation overflow/underflow validation.
15. Z80 instruction operands and indexed displacements now support expression-backed AST values, including mixed symbolic arithmetic (for example `target+delta`, `table+index`, `ix+disp`).
16. Z80 architecture now supports profile-driven validation (`default`, `strict-documented`, `gameboy-z80-subset`) through `pkg/arch/z80/profile` and CLI flag `-z80-profile`.
17. Z80 resolver diagnostics now include targeted ambiguity guidance and expected addressing-family hints for common operand mismatch failures.

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

## Phase 2: Operand Classifier + Resolver (Completed)

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

Completed result:

- Added `pkg/arch/z80/parser/register.go` with register and condition parameter classification, including `C` ambiguity handling via candidate sets.
- Added `pkg/arch/z80/parser/resolver.go` to resolve variants for the minimum operand patterns and produce typed `ResolvedInstruction` payloads.
- Added `pkg/arch/z80/parser/instruction.go` to parse 0/1/2 operand forms and build `ast.Instruction` nodes with typed arguments.
- Wired `pkg/arch/z80/z80.go` `ParseIdentifier` to delegate to the Z80 parser package.
- Added parser unit tests covering the minimum slice (`NOP`, `RET`, `LD A,B`, `LD A,n`, `LD HL,nn`, `JR`, `JR NZ`, `JP`, `JP NZ`, `CALL`) and error cases.

## Phase 3: Address Assignment (Completed)

Files:

- `pkg/arch/z80/assembler/address_assigning_step.go`

Tasks:

- Set instruction address from program counter.
- Determine size from resolved opcode info (not from operand value heuristics).
- Store finalized size/addressing for opcode generation.

Definition of done:

- Address assignment tests pass for 1/2/3/4-byte instructions.

Completed result:

- Added `pkg/arch/z80/assembler/address_assigning_step.go` with resolved-argument decoding and opcode-info based size/address assignment.
- Added `pkg/arch/z80/assembler/address_assigning_step_test.go` covering 1/2/3/4-byte instruction sizes and expected error paths.
- Wired `pkg/arch/z80/z80.go` `AssignInstructionAddress` to delegate to the Z80 assembler package.

## Phase 4: Opcode Generation (Core) (Completed)

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
- `JR NZ,label` -> conditional relative form
- `BIT 3,A` -> `CB 5F`
- `LD IX,$1234` -> `DD 21 34 12`
- `BIT 3,(IX+5)` -> `DD CB 05 5E`

Completed result:

- Added `pkg/arch/z80/assembler/generate_opcode_step.go` with opcode byte emission for implied, register, immediate, extended, relative, register-indirect, and port addressing forms.
- Added explicit handling for bit-operation opcode synthesis:
  - CB bit family (`BIT/RES/SET b,r`) opcode construction from bit index and register code.
  - Indexed bit prefix chains (`DD CB d op` / `FD CB d op`) with displacement and synthesized final opcode.
- Added `pkg/arch/z80/assembler/generate_opcode_step_test.go` coverage for core encodings and error paths (missing operands, invalid bit index, and relative-range failures).
- Updated `pkg/arch/z80/assembler/address_assigning_step.go` opcode lookup fallback so instructions with register parameters but addressing-based opcode maps (for example, CB/indexed-bit families) resolve correctly.
- Extended `pkg/assembler/address_assigning_step.go` argument value resolution to support `ast.Number`, `ast.Label`, and `ast.Identifier` nodes used by typed Z80 operand payloads.
- Wired `pkg/arch/z80/z80.go` `GenerateInstructionOpcode` to delegate to the Z80 assembler package.

## Phase 5: Extended Instruction Coverage (Completed)

Tasks:

- Expand resolver/opcode coverage to all instruction families in grouped variants.
- Add CB/ED/DD/FD family coverage and edge conditions.
- Decide and implement behavior for undocumented ops (include or reject explicitly).

Definition of done:

- Coverage tests generated from instruction groups ensure no silent gaps in supported subset.

Completed result:

- Added metadata-driven opcode coverage test `pkg/arch/z80/assembler/coverage_test.go` that:
  - enumerates instruction variants from `Opcodes`, `EDOpcodes`, `DDOpcodes`, `FDOpcodes`, and explicit CB/indexed-bit families,
  - synthesizes a valid `ResolvedInstruction` per variant,
  - validates both address assignment and opcode generation for every variant.
- Expanded resolver matching in `pkg/arch/z80/parser/resolver.go`:
  - added value-first register operand support (`BIT/RES/SET b,r` style),
  - added numeric register-opcode support (`IM n`, `RST n` style).
- Added numeric register candidate mapping in `pkg/arch/z80/parser/register.go` for interrupt modes and restart vectors.
- Added parser tests covering the new resolver paths:
  - `BIT 3,A`
  - `IM 1`
  - `RST $38`
- Fixed immediate-opcode emission in `pkg/arch/z80/assembler/generate_opcode_step.go` to handle zero-byte immediate payload forms used by `IM` register-opcode variants.
- Undocumented opcode behavior is now explicit: undocumented instruction variants are included and assembled when present in retrogolib opcode metadata.

## Phase 6: CLI and Runtime Integration (Completed)

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

Completed result:

- Updated `cmd/retroasm/main.go` architecture validation to support both CPU modes:
  - `-cpu 6502` and `-cpu z80`
  - compatible systems per CPU with explicit incompatibility errors
  - defaults: `6502 -> nes`, `z80 -> generic`, and system-driven CPU defaults for `nes`, `gameboy`, `zx-spectrum`, and `generic`
- Updated CLI flag help text to include Z80/system options.
- Replaced hard-coded m6502 registration in CLI assembly path with CPU-selected architecture registration (`6502` or `z80`).
- Updated `pkg/retroasm/default.go` runtime assembly path to:
  - resolve config from registered architecture adapters,
  - assemble with either m6502 or Z80 config types,
  - keep backward-compatible m6502 fallback when no architecture is registered.
- Added/updated tests in `cmd/retroasm/main_test.go` for:
  - extended CPU/system validation matrix,
  - CPU-specific architecture registration,
  - assembly with config file for both 6502 and Z80.

## Phase 7: Integration Fixtures and Regression Harness (Completed)

Files:

- `cmd/retroasm/z80_fixture_test.go`
- `tests/z80/basic.asm`
- `tests/z80/indexed.asm`
- `tests/z80/branches.asm`
- `tests/z80/branches_overflow.asm`

Tasks:

- Add fixture-driven Z80 integration tests that parse and assemble real source files through the runtime architecture path.
- Verify byte-accurate output for core, prefixed, and branch/control-flow samples.
- Add a regression fixture that intentionally exceeds relative-branch range and assert the assembler returns an error.

Definition of done:

- `go test ./cmd/retroasm` assembles fixture sources end-to-end with `-cpu z80` architecture registration.
- Expected bytes are asserted for passing fixtures.
- Out-of-range relative branch fixture fails with a clear error.

Completed result:

- Added fixture-based integration tests in `cmd/retroasm/z80_fixture_test.go` that:
  - read source files from `tests/z80` using a dynamic project-root path (no hardcoded absolute paths),
  - register Z80 architecture via the same runtime registration path used by CLI logic,
  - assemble fixtures through `retroasm.AssembleText`,
  - assert exact output bytes for `basic.asm`, `indexed.asm`, and `branches.asm`.
- Added `branches_overflow.asm` regression fixture and assertion that assembly fails with a relative-offset range error.
- Added fixture inputs for:
  - core smoke path (`basic.asm`),
  - prefix-heavy instruction path (`indexed.asm`),
  - control-flow branch/call/jump path (`branches.asm`).

## Phase 8: Indexed and Parenthesized Operand Parsing (Completed)

Files:

- `pkg/arch/z80/parser/instruction.go`
- `pkg/arch/z80/parser/resolver.go`
- `pkg/arch/z80/parser/register.go`
- `pkg/arch/z80/parser/instruction_test.go`
- `pkg/arch/z80/parser/register_test.go`
- `tests/z80/indexed.asm`
- `cmd/retroasm/z80_fixture_test.go`

Tasks:

- Extend operand parsing beyond scalar tokens to support:
  - parenthesized indirect operands (`(hl)`, `(ix)`, `(iy)`),
  - indexed displacement operands (`(ix+d)`, `(iy+d)`),
  - compact lexer forms where negative displacement is tokenized as one identifier (`(iy-1)`).
- Extend resolver matching for indexed forms:
  - `r,(ix+d)` / `r,(iy+d)`,
  - `(ix+d),r` / `(iy+d),r`,
  - `bit n,(ix+d)` / `bit n,(iy+d)`.
- Disambiguate IX/IY indexed LD direction for opcodes sharing the same register key by validating opcode layout against operand order.

Definition of done:

- Parser unit tests cover parenthesized/indexed success and error paths.
- End-to-end fixture assembly accepts IX/IY indexed syntax and emits expected bytes.
- Existing non-indexed Z80 parsing behavior remains green.

Completed result:

- Updated `pkg/arch/z80/parser/instruction.go` to parse multi-token operand forms and produce typed indexed/parenthesized operand payloads.
- Updated `pkg/arch/z80/parser/register.go` with explicit indirect-register candidate helpers and indexed-base mapping.
- Updated `pkg/arch/z80/parser/resolver.go` to:
  - resolve indexed register/memory operand combinations,
  - resolve indexed bit family forms with displacement propagation,
  - prioritize indexed register forms before generic register+value matching,
  - enforce correct LD indexed direction selection (`LD r,(IX/IY+d)` vs `LD (IX/IY+d),r`).
- Added parser unit coverage in `pkg/arch/z80/parser/instruction_test.go` and `pkg/arch/z80/parser/register_test.go` for:
  - `(ix+5)`, `(iy-2)`, compact `(iy-1)` tokenization,
  - `JP (IX)`,
  - indexed-bit and indexed-LD resolution.
- Expanded `tests/z80/indexed.asm` and updated `cmd/retroasm/z80_fixture_test.go` expected bytes to validate end-to-end indexed parsing and opcode emission.

## Phase 9: Extended Indirect/Register and Port I/O Operand Resolution (Completed)

Files:

- `pkg/arch/z80/parser/resolver.go`
- `pkg/arch/z80/parser/instruction_test.go`
- `cmd/retroasm/z80_fixture_test.go`
- `tests/z80/io_extended.asm`
- `.gitignore`

Tasks:

- Extend resolver matching for parenthesized extended-indirect register transfer forms:
  - `ld r,(nn)` and `ld (nn),r` for accumulator and register-pair forms.
- Add explicit opcode-direction checks for ambiguous extended `ld` register keys so load/store variants are selected deterministically.
- Extend resolver matching for port I/O operand families:
  - immediate port forms: `in a,(n)` and `out (n),a`,
  - C-register port forms: `in r,(c)` and `out (c),r`.
- Add parser regression coverage and end-to-end fixture coverage for the new operand forms.

Definition of done:

- Parser unit tests resolve extended-indirect and port I/O operand forms to correct instruction variants.
- Invalid immediate-port forms (for example `out (n),b`) are rejected with parser errors.
- End-to-end fixture assembly emits expected bytes for extended-indirect and port I/O instructions.

Completed result:

- Updated `pkg/arch/z80/parser/resolver.go` with dedicated resolver passes for:
  - extended register/memory transfers (`ld r,(nn)` and `ld (nn),r`),
  - immediate-port forms (`in a,(n)`, `out (n),a`),
  - C-port register forms (`in r,(c)`, `out (c),r`).
- Added extended `ld` direction filtering so register-key collisions resolve to the correct opcode family (`load` vs `store`) based on operand order.
- Added a parenthesized-immediate guard in register+value matching to avoid misclassifying `ld r,(nn)` as immediate addressing.
- Expanded `pkg/arch/z80/parser/instruction_test.go` with success cases for:
  - `ld a,($1234)`, `ld ($2345),a`,
  - `ld bc,($3456)`, `ld ($4567),bc`,
  - `in a,($12)`, `out ($34),a`,
  - `in b,(c)`, `out (c),e`,
  and an error regression for `out ($34),b`.
- Added `tests/z80/io_extended.asm` and expected fixture bytes in `cmd/retroasm/z80_fixture_test.go` for integrated runtime-path verification.
- Updated `.gitignore` to include the new Z80 fixture file.

## Phase 10: Tokenized Offset Operand Parsing (Completed)

Files:

- `pkg/arch/z80/parser/instruction.go`
- `pkg/arch/z80/parser/instruction_test.go`
- `cmd/retroasm/z80_fixture_test.go`
- `tests/z80/offsets.asm`
- `.gitignore`

Tasks:

- Extend operand parsing for tokenized offset forms in value positions:
  - `label+n` / `label-n` as `Identifier +/- Number` token sequences.
- Extend parenthesized value operand parsing to accept offset forms:
  - `(label+n)` / `(label-n)` and `($nn+n)` / `($nn-n)`.
- Keep IX/IY indexed parsing behavior unchanged by routing `(ix+d)` / `(iy+d)` through indexed displacement logic.
- Add parser and fixture regressions to verify runtime assembly output for the new forms.

Definition of done:

- Parser resolves tokenized offset operands without requiring compact lexer tokens.
- Parenthesized offset values are accepted for extended and port-addressing forms.
- Invalid offset syntax (non-numeric offset after `+`/`-`) returns parser errors.
- End-to-end fixture assembly validates byte output for offset-based operands.

Completed result:

- Updated `pkg/arch/z80/parser/instruction.go` to parse offset token sequences for both unparenthesized and parenthesized value operands.
- Added dedicated parsing helpers for:
  - numeric offset composition on number bases,
  - symbolic offset composition on label bases,
  - parenthesized offset operands with closing-parenthesis validation.
- Preserved indexed register displacement behavior by keeping `(ix+d)` / `(iy+d)` routed to indexed parsing before generic parenthesized-offset handling.
- Expanded `pkg/arch/z80/parser/instruction_test.go` with coverage for:
  - `jp target+2`,
  - `ld a,(table+1)`,
  - `in a,($10+1)`,
  - invalid `target+next` offset syntax.
- Added `tests/z80/offsets.asm` and expected bytes in `cmd/retroasm/z80_fixture_test.go` for end-to-end regression coverage.
- Updated `.gitignore` to include the new Z80 offset fixture file.

## Phase 11: Chained Tokenized Offset Operand Parsing (Completed)

Files:

- `pkg/arch/z80/parser/instruction.go`
- `pkg/arch/z80/parser/instruction_test.go`
- `cmd/retroasm/z80_fixture_test.go`
- `tests/z80/offsets_chained.asm`
- `.gitignore`

Tasks:

- Extend tokenized offset parsing from single-term offsets to chained additive/subtractive terms:
  - `label+3-1`,
  - `(label+3-1)`,
  - `($10+3-1)`.
- Reuse chained offset parsing for both plain and parenthesized value operands.
- Validate offset term parsing errors for invalid forms (missing numeric term after `+`/`-`).
- Add parser and integration fixture regressions for chained-offset assembly output.

Definition of done:

- Parser accepts chained `+/-` numeric offset terms on identifier and numeric bases.
- Parenthesized chained offsets resolve correctly for extended and port-addressing instruction forms.
- Invalid chained offset syntax returns deterministic parser errors.
- End-to-end fixture assembly validates expected bytes for chained offset source.

Completed result:

- Updated `pkg/arch/z80/parser/instruction.go` to parse and accumulate chained `+/- number` offset terms.
- Reused accumulated offset handling across both unparenthesized and parenthesized operand parsing paths.
- Added range checks for signed offset accumulation and numeric base overflow/underflow.
- Expanded `pkg/arch/z80/parser/instruction_test.go` coverage for:
  - `jp target+3-1`,
  - `ld a,(table+3-1)`,
  - `in a,($10+3-1)`,
  - invalid `target+` trailing-operator syntax.
- Added `tests/z80/offsets_chained.asm` and expected fixture bytes in `cmd/retroasm/z80_fixture_test.go`.
- Updated `.gitignore` to include the chained-offset fixture file.

## Phase 12: Expression-Backed Operand Values and Displacements (Completed)

Files:

- `pkg/arch/z80/parser/instruction.go`
- `pkg/arch/z80/parser/resolver.go`
- `pkg/assembler/address_assigning_step.go`
- `pkg/arch/z80/parser/instruction_test.go`
- `cmd/retroasm/z80_fixture_test.go`
- `tests/z80/expressions.asm`

Tasks:

- Replace token-only instruction operand value handling with expression-backed values where needed.
- Support richer operand expressions in instruction contexts:
  - `jp target+other`,
  - `ld a,(table+index)`,
  - `(ix+label-$)` style displacement expressions (if representable safely in current pipeline).
- Ensure address assignment can evaluate these expressions via scope symbols without string-encoded offsets.
- Preserve existing fast paths for simple literal/register operands.

Definition of done:

- Instruction operands can carry and resolve expression AST payloads, not only labels/numbers.
- Existing offset-chain syntax keeps working unchanged.
- New expression fixture assembles with expected bytes and no regressions in current fixtures.

Completed result:

- Added AST expression operand node support in `pkg/parser/ast/expression.go` and copy coverage in `pkg/parser/ast/node_test.go`.
- Updated `pkg/arch/z80/parser/instruction.go` to emit expression-backed operand values for non-trivial value expressions and parenthesized expressions.
- Extended indexed displacement parsing to accept expression displacements (for example `(ix+disp)`), while preserving numeric fast paths.
- Updated `pkg/assembler/address_assigning_step.go` to evaluate `ast.Expression` operands during argument resolution, including `$` program-counter expressions.
- Updated generic opcode generation setup in `pkg/assembler/generate_opcode_step.go` so expression evaluation has architecture width and instruction address context.
- Expanded parser tests in `pkg/arch/z80/parser/instruction_test.go` for symbolic expressions in jump/extended/port/indexed-displacement forms.
- Added end-to-end expression fixture `tests/z80/expressions.asm` with expected output in `cmd/retroasm/z80_fixture_test.go`.
- Updated `.gitignore` fixture allowlist with `tests/z80/expressions.asm`.

## Phase 13: Profile Strictness and Undocumented-Op Policy (Completed)

Files:

- `cmd/retroasm/main.go`
- `pkg/arch/z80/options.go`
- `pkg/arch/z80/z80.go`
- `pkg/arch/z80/parser/instruction.go`
- `pkg/arch/z80/profile/profile.go`
- `cmd/retroasm/main_test.go`
- `cmd/retroasm/z80_fixture_test.go`

Tasks:

- Add explicit Z80 profile selection (for example: default, strict documented-only, gameboy-subset).
- Enforce profile-specific opcode acceptance at resolver or opcode generation boundary.
- Keep current default behavior backward compatible.

Definition of done:

- CLI/profile config can enable strict profile validation.
- Unsupported-by-profile instructions fail with clear errors.
- Default profile behavior remains unchanged.

Completed result:

- Added a new type-safe `pkg/arch/z80/profile` package with:
  - profile parsing (`default`, `strict-documented`, `gameboy-z80-subset`),
  - validation gates for undocumented opcodes,
  - gameboy subset gating for unsupported prefixes/mnemonics.
- Added Z80 architecture options in `pkg/arch/z80/options.go` and wired profile selection through `pkg/arch/z80/z80.go`.
- Updated parser entry points in `pkg/arch/z80/parser/instruction.go` to enforce profile checks at resolution time.
- Added CLI flag `-z80-profile` in `cmd/retroasm/main.go`, including normalization/defaulting and CPU/profile compatibility validation.
- Expanded regression coverage in:
  - `pkg/arch/z80/profile/profile_test.go`,
  - `pkg/arch/z80/parser/profile_test.go`,
  - `cmd/retroasm/main_test.go`,
  - `cmd/retroasm/z80_fixture_test.go`.

## Phase 14: Parser/Resolver Diagnostic Quality Pass (Completed)

Files:

- `pkg/arch/z80/parser/resolver.go`
- `pkg/arch/z80/parser/instruction_test.go`

Tasks:

- Replace generic resolution failures with actionable diagnostics that include expected operand families.
- Add focused errors for top confusion points:
  - condition vs register `c`,
  - immediate vs indirect/addressed forms,
  - indexed vs non-indexed load direction conflicts.
- Add regression tests that assert message quality for representative failure cases.

Definition of done:

- High-frequency parse/resolution failures produce deterministic, specific messages.
- Existing success-path behavior remains unchanged.

Completed result:

- Updated `pkg/arch/z80/parser/resolver.go` to replace generic no-match failures with diagnostic-aware errors.
- Added expected addressing-family hints to resolver mismatch messages for faster user recovery.
- Added focused diagnostics for the three high-confusion cases:
  - condition vs register `c` ambiguity,
  - immediate vs addressed/parenthesized form mismatch,
  - indexed load direction mismatch.
- Added regression tests in `pkg/arch/z80/parser/instruction_test.go` that assert message quality for representative failure inputs.

## Phase 15: Robustness and Compatibility Expansion (Planned)

Files:

- `pkg/arch/z80/parser/*_test.go`
- `pkg/arch/z80/assembler/*_test.go`
- `cmd/retroasm/z80_fixture_test.go`
- `tests/z80/*` (new compatibility fixtures)

Tasks:

- Add fuzz/property tests for parser operand forms and resolver variant selection.
- Expand fixture corpus with compatibility-style sources (including tricky expression and control-flow cases).
- Add matrix assertions for boundary values (relative branches, displacements, port values, extended addresses).

Definition of done:

- Fuzz/property tests run in CI without flakiness.
- Fixture matrix covers all critical operand categories and boundary conditions.
- No regressions across existing Z80 and 6502 test suites.

## Testing Strategy

## Unit Tests

- parser classification and resolver (including `C` condition/register ambiguity)
- address assignment for mixed instruction sizes
- opcode generation for each addressing family

## Integration Tests

- `tests/z80/basic.asm` core instruction smoke test
- `tests/z80/indexed.asm` IX/IY indexed operand syntax + prefixed opcode coverage (`DD`, `FD`, `ED`)
- `tests/z80/branches.asm` relative and absolute control-flow encoding checks
- `tests/z80/branches_overflow.asm` relative branch out-of-range regression check
- `tests/z80/io_extended.asm` extended indirect register transfer and port I/O coverage (`LD (nn),r` / `LD r,(nn)`, `IN/OUT`)
- `tests/z80/offsets.asm` tokenized offset operand coverage (`label+n`, `(label+n)`, `($nn+n)`)
- `tests/z80/offsets_chained.asm` chained tokenized offset coverage (`label+n-m`, `(label+n-m)`, `($nn+n-m)`)
- `tests/z80/expressions.asm` expression-backed operand and indexed displacement coverage (`target+delta`, `table+index`, `(ix+disp)`)
- `cmd/retroasm/z80_fixture_test.go` profile-gated assembly checks for default, strict-documented, and gameboy-z80-subset behavior

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
8. Phase 7 (fixture-based integration regression)
9. Phase 8 (indexed/parenthesized operand parser completion)
10. Phase 9 (extended indirect/register and port I/O operand resolution)
11. Phase 10 (tokenized offset operand parsing)
12. Phase 11 (chained tokenized offset operand parsing)
13. Phase 12 (expression-backed operand values and displacements)
14. Phase 13 (profile strictness and undocumented-op policy)
15. Phase 14 (parser/resolver diagnostic quality pass)
16. Phase 15 (robustness and compatibility expansion)

This order gets a small but real end-to-end Z80 path working early, then scales coverage safely.
