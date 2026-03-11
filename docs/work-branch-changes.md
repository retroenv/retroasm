# Work Branch Changes

This document tracks every file changed in the `work` branch compared to `main`. It must be kept up-to-date as features are developed. When extracting features to `main`, mark the relevant entries as merged.

**Branch:** `work`
**Base:** `main`
**Last updated:** 2026-03-11

---

## Build & Configuration

### `.gitignore`
**Why:** Allow Z80 test fixture `.asm` files to be tracked while keeping other test artifacts ignored.
**What:** Changed `tests/` exclusion to `tests/*` with explicit `!tests/z80/` and `!tests/z80/*.asm` exceptions.

### `.golangci.yml`
**Why:** New architecture code (Z80 resolver, M68000 encoder) requires slightly higher cyclomatic complexity allowance; formatting cleanup.
**What:** Increased `cyclop.max-complexity` from 15 to 18. Added blank lines between YAML sections for readability.

### `go.mod`
**Why:** Development requires local retrogolib changes (new Z80/M68000/Chip-8 CPU definitions).
**What:** Added `replace` directive pointing `retrogolib` to local checkout. **Must be removed before merging to main.**

### `README.md`
**Why:** Document newly supported architectures and updated CLI options.
**What:** Added architecture support section (6502, Chip-8, Z80, M68000). Updated CLI usage examples and flag descriptions to reflect multi-architecture support.

---

## CLI (`cmd/retroasm/`)

### `cmd/retroasm/main.go`
**Why:** Support multi-architecture assembling from the command line.
**What:**
- Added imports for Chip-8, M68000, Z80, and Z80 profile packages
- Replaced single-architecture constants with lookup tables (`supportedSystemsByCPU`, `defaultSystemByCPU`, `defaultCPUBySystem`)
- Added `--z80-profile` flag for Z80 instruction set filtering
- Refactored `validateAndProcessArchitecture()` into smaller functions: `normalizeArchitectureOptions`, `setDefaultArchitecture`, `applyDerivedArchitectureDefaults`, `validateArchitectureCompatibility`, `validateZ80Profile`
- Added `registerArchitectureForCPU()` to create correct architecture adapter per CPU
- Added `assembleChip8File()` for direct Chip-8 assembler invocation (bypasses high-level API)
- Replaced `map[string]struct{}` with `set.Set[string]`

### `cmd/retroasm/main_test.go`
**Why:** Test coverage for new multi-architecture CLI validation logic.
**What:** Extended test tables for CPU/system validation, Z80 profile validation, architecture compatibility checks. Reordered exported tests before unexported types per funcorder lint rule.

### `cmd/retroasm/z80_fixture_test.go` (new)
**Why:** End-to-end integration tests for Z80 assembler via CLI.
**What:** Test fixtures assembling Z80 `.asm` files from `tests/z80/` and verifying binary output byte-for-byte.

---

## Core Assembler (`pkg/assembler/`)

### `pkg/assembler/assembler.go`
**Why:** Support multi-architecture assembly and symbol export.
**What:**
- Added `Symbols()` method to expose resolved label addresses after assembly
- Added single-segment auto-initialization in `parseASTNodes()` for architectures that don't use ca65 segment config

### `pkg/assembler/config/ca65.go`
**Why:** Satisfy funcorder linter (types grouped before methods).
**What:** Moved `ca65Area` struct definition after constructor/exported functions.

### `pkg/assembler/memory.go`
**Why:** Satisfy funcorder linter (constructor before type).
**What:** Moved `newMemory()` constructor before `memory` struct definition.

### `pkg/assembler/nodes.go`
**Why:** Support Z80 register-value instruction arguments; satisfy funcorder linter.
**What:**
- Added `RegisterValueArgument` and `RegisterRegisterValueArgument` exported types for Z80-style `LD reg, value` instructions
- Reordered all type definitions to appear grouped together before their methods (funcorder compliance)

### `pkg/assembler/parse_ast_nodes.go`
**Why:** Handle new AST node types for register-value instructions.
**What:**
- Added `ast.RegisterValue` and `ast.RegisterRegisterValue` cases in `convertInstructionArgument()`
- Added `convertRegisterValue()` and `convertRegisterRegisterValue()` helper functions
- Minor error message cleanup

### `pkg/assembler/parse_ast_nodes_test.go`
**Why:** Test coverage for register-value argument conversion; funcorder compliance.
**What:** Added test cases for typed instruction arguments (single and multi-operand). Moved `testTypedInstructionArgument` type to end of file.

### `pkg/assembler/steps.go`
**Why:** Satisfy funcorder linter (exported function before unexported type).
**What:** Moved `step[T]` type definition after `Steps()` method.

---

## AST & Parser (`pkg/parser/`)

### `pkg/parser/ast/register.go` (new)
**Why:** AST representation for Z80/M68000 instructions that pair registers with values.
**What:** Added `RegisterValue` and `RegisterRegisterValue` AST node types with constructors and `Copy()` methods.

### `pkg/parser/directives/directives_test.go`
**Why:** Funcorder compliance; improved test organization.
**What:** Moved `newMockParser` constructor before exported test functions. Moved `mockParser` type after benchmarks.

---

## Scope (`pkg/scope/`)

### `pkg/scope/scope.go`
**Why:** Support symbol export from assembler (needed by `Assembler.Symbols()`).
**What:** Added `AllLabels()` method that returns resolved addresses of all label/function-type symbols in a scope.

---

## High-Level API (`pkg/retroasm/`)

### `pkg/retroasm/default.go`
**Why:** Make the high-level assembler API architecture-agnostic.
**What:**
- Refactored `AssembleAST()` and `AssembleText()` to use `resolveArchitectureConfig()` which dispatches to the correct generic assembler based on registered architecture type
- Added generic helper functions: `assembleASTWithConfig()`, `assembleTextWithConfig()`, `readAssemblerConfig()`, `applyBaseAddress()`, `adapterConfig()`
- Added `AddressWidth()` dynamic dispatch on `ArchitectureAdapter`
- Added new sentinel errors: `errAmbiguousArchitecture`, `errArchitectureAdapterMismatch`, `errArchitectureNotRegistered`, `errUnsupportedArchitectureConfig`
- Funcorder compliance: reordered types and constructors

---

## Architecture: 6502 (`pkg/arch/m6502/`)

### `pkg/arch/m6502/parser/instruction.go`
**Why:** Funcorder compliance; use `any` instead of `interface{}`.
**What:** Moved `instruction` struct after `ParseIdentifier()` function. Replaced `interface{}` with `any`. Added doc comment on `ParseIdentifier`.

### `pkg/arch/m6502/assembler/generate_opcode_step_test.go`
**Why:** Funcorder compliance (exported tests before unexported types).
**What:** Reordered: exported `TestGenerateInstructionOpcode_IndirectXY` first, then mock types grouped together, then mock methods.

---

## Architecture: Chip-8 (`pkg/arch/chip8/`) — ALL NEW

### `pkg/arch/chip8/chip8.go`
**Why:** New architecture support for Chip-8 virtual machine.
**What:** Architecture entry point. Creates `config.Config` with Chip-8 instruction set, parser, address assigner, and opcode generator. 12-bit address width.

### `pkg/arch/chip8/chip8_test.go`
**Why:** Unit tests for Chip-8 architecture initialization.
**What:** Tests instruction lookup, addressing mode validation, architecture config creation.

### `pkg/arch/chip8/chip8_assemble_test.go`
**Why:** Integration tests for Chip-8 end-to-end assembly.
**What:** Assembles Chip-8 programs and verifies binary output byte-for-byte.

### `pkg/arch/chip8/parser/instruction.go`
**Why:** Chip-8 instruction parsing.
**What:** Parses Chip-8 mnemonics and operands (registers V0-VF, I register, immediates, addresses) into AST nodes with correct addressing mode resolution.

### `pkg/arch/chip8/assembler/address_assigning_step.go`
**Why:** Chip-8 address assignment in assembler pipeline.
**What:** Assigns addresses to Chip-8 instructions (all 2 bytes, starting at `$200`).

### `pkg/arch/chip8/assembler/generate_opcode_step.go`
**Why:** Chip-8 opcode generation.
**What:** Encodes Chip-8 instructions into 2-byte big-endian opcodes based on addressing mode and operand values.

---

## Architecture: Z80 (`pkg/arch/z80/`) — ALL NEW

### `pkg/arch/z80/z80.go`
**Why:** New architecture support for Z80 CPU.
**What:** Architecture entry point. Uses `*InstructionGroup` as generic type T (groups instruction variants by mnemonic). Builds instruction groups from retrogolib Z80 definitions. Supports profile-based filtering.

### `pkg/arch/z80/z80_test.go`
**Why:** Unit tests for Z80 architecture initialization and instruction groups.
**What:** Tests instruction group building, profile filtering, architecture config creation.

### `pkg/arch/z80/options.go`
**Why:** Functional options pattern for Z80 architecture configuration.
**What:** `WithProfile()` option to select Z80 instruction subset (full, strict-documented, gameboy).

### `pkg/arch/z80/parser/instruction.go`
**Why:** Z80 instruction parsing.
**What:** Parses Z80 mnemonics and operands (registers, register pairs, conditions, indexed `(IX+d)`, indirect `(HL)`, port `(C)`, bit numbers, immediates) into AST nodes.

### `pkg/arch/z80/parser/instruction_test.go`
**Why:** Comprehensive test coverage for Z80 instruction parsing.
**What:** 1191 lines of test cases covering all Z80 addressing modes, edge cases, error conditions.

### `pkg/arch/z80/parser/register.go`
**Why:** Z80 register name resolution.
**What:** Maps register/pair/condition names to retrogolib constants. Handles case-insensitive lookup.

### `pkg/arch/z80/parser/register_test.go`
**Why:** Tests for register name resolution.
**What:** Tests all register names, pairs, conditions, and invalid inputs.

### `pkg/arch/z80/parser/resolver.go`
**Why:** Z80 instruction resolution (matching parsed operands to instruction variants).
**What:** Top-level dispatcher: no-operand, single-operand, two-operand. Shared helper functions.

### `pkg/arch/z80/parser/resolver_single_operand.go`
**Why:** Single-operand Z80 instruction resolution.
**What:** Handles register, immediate, indirect, indexed, condition, port, and restart vector operands.

### `pkg/arch/z80/parser/resolver_two_operand.go`
**Why:** Two-operand Z80 instruction resolution.
**What:** Dispatches to register-pair, special-pair, and general two-operand resolution.

### `pkg/arch/z80/parser/resolver_indirect.go`
**Why:** Indirect addressing resolution for Z80.
**What:** Handles `(HL)`, `(BC)`, `(DE)`, `(nn)` indirect loads/stores and indirect-immediate patterns.

### `pkg/arch/z80/parser/resolver_extended.go`
**Why:** Extended memory operation resolution for Z80.
**What:** Handles `LD (nn),rr` and `LD rr,(nn)` 16-bit memory operations.

### `pkg/arch/z80/parser/resolver_port.go`
**Why:** Port I/O instruction resolution for Z80.
**What:** Handles `IN`/`OUT` with port register `(C)` and immediate port `(n)`.

### `pkg/arch/z80/parser/resolver_indexed.go`
**Why:** Indexed register operation resolution for Z80.
**What:** Handles `(IX+d)` and `(IY+d)` addressing with displacement.

### `pkg/arch/z80/parser/resolver_value.go`
**Why:** Value-register and bit operation resolution for Z80.
**What:** Handles `BIT`/`SET`/`RES` bit operations and `IM` interrupt mode.

### `pkg/arch/z80/parser/resolver_diagnostics.go`
**Why:** Error diagnostics for Z80 instruction resolution failures.
**What:** Generates detailed error messages with suggestions when instruction resolution fails.

### `pkg/arch/z80/parser/mock_parser_test.go`
**Why:** Shared test mock for Z80 parser tests.
**What:** Mock parser implementing `arch.Parser` interface for unit testing.

### `pkg/arch/z80/parser/fuzz_test.go`
**Why:** Fuzz testing for Z80 instruction parser robustness.
**What:** Fuzz targets for register parsing and instruction parsing with random token sequences.

### `pkg/arch/z80/parser/profile_test.go`
**Why:** Tests for Z80 profile-based instruction filtering.
**What:** Verifies that profile filtering correctly accepts/rejects instructions.

### `pkg/arch/z80/parser/doc.go`
**Why:** Package documentation.
**What:** Package doc comment for `z80/parser`.

### `pkg/arch/z80/profile/profile.go`
**Why:** Z80 instruction set profiles for different target platforms.
**What:** Defines `Default`, `StrictDocumented`, and `GameBoyZ80Subset` profiles. Filters instruction sets based on platform requirements.

### `pkg/arch/z80/profile/profile_test.go`
**Why:** Tests for Z80 profile filtering.
**What:** Verifies profile parsing, filtering behavior, and instruction subset correctness.

### `pkg/arch/z80/profile/doc.go`
**Why:** Package documentation.
**What:** Package doc comment for `z80/profile`.

### `pkg/arch/z80/assembler/address_assigning_step.go`
**Why:** Z80 address assignment in assembler pipeline.
**What:** Assigns addresses based on instruction group opcode lengths (variable-length: 1-4 bytes).

### `pkg/arch/z80/assembler/address_assigning_step_test.go`
**Why:** Tests for Z80 address assignment.
**What:** Tests various instruction sizes and addressing modes.

### `pkg/arch/z80/assembler/generate_opcode_step.go`
**Why:** Z80 opcode generation.
**What:** Encodes Z80 instructions into byte sequences, handling prefixed opcodes (CB, DD, ED, FD), displacement bytes, and 8/16-bit immediates.

### `pkg/arch/z80/assembler/generate_opcode_step_test.go`
**Why:** Tests for Z80 opcode generation.
**What:** Verifies correct byte output for various Z80 instructions.

### `pkg/arch/z80/assembler/coverage_test.go`
**Why:** Ensure all Z80 instruction groups can be assembled.
**What:** Iterates all registered instruction groups and verifies they produce valid output.

### `pkg/arch/z80/assembler/doc.go`
**Why:** Package documentation.
**What:** Package doc comment for `z80/assembler`.

---

## Architecture: M68000 (`pkg/arch/m68000/`) — ALL NEW

### `pkg/arch/m68000/m68000.go`
**Why:** New architecture support for Motorola 68000 CPU.
**What:** Architecture entry point. Uses `*m68000.Instruction` as generic type T. 24-bit address width. Big-endian opcode output.

### `pkg/arch/m68000/m68000_test.go`
**Why:** Integration tests for M68000 assembly.
**What:** Tests MOVE instruction assembly with various addressing modes.

### `pkg/arch/m68000/parser/instruction.go`
**Why:** M68000 instruction parsing.
**What:** Parses M68000 mnemonics with size suffixes (`.B`/`.W`/`.L`), effective addresses, condition codes. Uses `lastMnemonic` for Bcc/DBcc/Scc resolution.

### `pkg/arch/m68000/parser/effective_address.go`
**Why:** M68000 effective address parsing.
**What:** Parses all M68000 addressing modes: register direct, register indirect, pre/post-increment/decrement, displacement, absolute, immediate, PC-relative.

### `pkg/arch/m68000/parser/condition.go`
**Why:** M68000 condition code parsing.
**What:** Maps condition code mnemonics (EQ, NE, GT, etc.) to opcodes for Bcc/DBcc/Scc instructions.

### `pkg/arch/m68000/parser/condition_test.go`
**Why:** Tests for condition code parsing.
**What:** Tests all condition codes and invalid inputs.

### `pkg/arch/m68000/parser/register.go`
**Why:** M68000 register name resolution.
**What:** Maps register names (D0-D7, A0-A7, SP, PC, SR, CCR, USP) to constants.

### `pkg/arch/m68000/parser/register_list.go`
**Why:** M68000 register list parsing for MOVEM.
**What:** Parses register range expressions like `D0-D3/A0-A2` into bitmasks.

### `pkg/arch/m68000/parser/register_list_test.go`
**Why:** Tests for register list parsing.
**What:** Tests single registers, ranges, and combined lists.

### `pkg/arch/m68000/parser/resolved.go`
**Why:** M68000 resolved instruction representation.
**What:** `ResolvedInstruction` struct holding parsed instruction, size, source/destination effective addresses, and extra data.

### `pkg/arch/m68000/parser/size.go`
**Why:** M68000 operation size suffix parsing.
**What:** Parses `.B`, `.W`, `.L` size suffixes from token stream.

### `pkg/arch/m68000/parser/size_test.go`
**Why:** Tests for size suffix parsing.
**What:** Tests valid and invalid size suffixes.

### `pkg/arch/m68000/assembler/address_assigning_step.go`
**Why:** M68000 address assignment.
**What:** Calculates instruction sizes based on opcode + extension words (EA modes determine extension word count).

### `pkg/arch/m68000/assembler/address_assigning_step_test.go`
**Why:** Tests for M68000 address assignment.
**What:** Tests instruction sizes for NOP, MOVE variants, CLR, Bcc, DBcc, MOVEM.

### `pkg/arch/m68000/assembler/generate_opcode_step.go`
**Why:** M68000 opcode generation.
**What:** Dispatches to instruction-specific encoders based on instruction name.

### `pkg/arch/m68000/assembler/generate_opcode_step_test.go`
**Why:** Tests for M68000 opcode generation.
**What:** Verifies correct big-endian opcode output.

### `pkg/arch/m68000/assembler/encode.go`
**Why:** M68000 shared encoding helpers.
**What:** Effective address encoding, extension word generation, register field encoding.

### `pkg/arch/m68000/assembler/encode_alu.go`
**Why:** M68000 ALU instruction encoding.
**What:** Encodes ADD, SUB, AND, OR, EOR, CMP, MULU, MULS, DIVU, DIVS.

### `pkg/arch/m68000/assembler/encode_misc.go`
**Why:** M68000 miscellaneous instruction encoding.
**What:** Encodes MOVE, MOVEQ, MOVEM, LEA, PEA, CLR, NEG, NOT, TST, EXT, SWAP, NOP, RTS, RTE, TRAP, LINK, UNLK, Bcc, DBcc, Scc, BSR, JSR, JMP, STOP, ILLEGAL.

### `pkg/arch/m68000/assembler/coverage_test.go`
**Why:** Ensure all M68000 instructions can be assembled.
**What:** Coverage tests for instruction encoding with various EA combinations.

---

## Architecture: x86 (`pkg/arch/x86/`) — ALL NEW

### `pkg/arch/x86/x86.go`
**Why:** New architecture support for Intel 8086/80286 CPU.
**What:** Architecture entry point. Creates config with x86 instruction set. 16-bit address width.

### `pkg/arch/x86/x86_test.go`
**Why:** Unit tests for x86 architecture initialization.
**What:** Tests instruction lookup, config creation, address width.

### `pkg/arch/x86/types.go`
**Why:** x86 type definitions.
**What:** Addressing mode constants, instruction type definition.

### `pkg/arch/x86/instruction.go`
**Why:** x86 instruction definitions.
**What:** Instruction table with opcode mappings for 8086/286 instructions.

### `pkg/arch/x86/parser/instruction.go`
**Why:** x86 instruction parsing.
**What:** Parses x86 mnemonics and operands (registers, memory references, immediates, segment overrides).

### `pkg/arch/x86/assembler/address_assigning_step.go`
**Why:** x86 address assignment.
**What:** Assigns addresses to x86 instructions based on encoding length.

### `pkg/arch/x86/assembler/generate_opcode_step.go`
**Why:** x86 opcode generation.
**What:** Encodes x86 instructions into byte sequences with ModR/M bytes, displacements, and immediates.

---

## Z80 Test Fixtures (`tests/z80/`) — ALL NEW

### `tests/z80/basic.asm`
Basic Z80 instructions (NOP, LD, HALT, arithmetic).

### `tests/z80/branches.asm`
Z80 branch/jump instructions (JR, JP, CALL, RET with conditions).

### `tests/z80/branches_overflow.asm`
Tests for branch displacement range limits (±127 bytes).

### `tests/z80/compatibility.asm`
Z80 instruction compatibility edge cases.

### `tests/z80/expressions.asm`
Expression evaluation in Z80 operands.

### `tests/z80/indexed.asm`
IX/IY indexed addressing tests.

### `tests/z80/indexed_boundaries.asm`
IX/IY displacement boundary tests (±128).

### `tests/z80/io_extended.asm`
Z80 I/O and extended (ED-prefix) instructions.

### `tests/z80/offsets.asm`
Offset/displacement calculation tests.

### `tests/z80/offsets_chained.asm`
Chained offset calculations.

### `tests/z80/profile_gameboy_subset.asm`
Game Boy Z80 subset instruction validation.

### `tests/z80/profile_gameboy_subset_rejects.asm`
Instructions rejected by Game Boy Z80 profile.

### `tests/z80/profile_strict_documented.asm`
Strict documented-only Z80 instruction validation.

### `tests/z80/profile_strict_documented_rejects.asm`
Instructions rejected by strict documented profile.

---

## Chip-8 Examples (`examples/chip8/`) — ALL NEW

### `examples/chip8/README.md`
Documentation for Chip-8 example programs.

### `examples/chip8/hello.asm`
Simple "hello world" Chip-8 program (draws a sprite).

### `examples/chip8/cube.asm`
Rotating cube Chip-8 demo program.

---

## Documentation (`docs/`) — ALL NEW

### `docs/compatibility-mode-plan.md`
Shared assembler compatibility mode infrastructure plan (mode enum, CLI flag, anonymous labels, colon-optional labels, number formats, directive registration).

### `docs/x816-compatibility-plan.md`
x816 assembler-specific compatibility plan (directives, keyword operators, `.dcb` strings, `*` as PC, bank byte modifier).

### `docs/asm6-compatibility.md`
asm6/asm6f assembler compatibility plan (current support status, `@` local labels, `ENUM`/`ENDE`, undocumented opcodes, NES 2.0 headers).

### `docs/ca65-compatibility.md`
ca65 assembler compatibility plan (segment model, `.proc` scoping, `.define` macros, `.feature` flags, keyword operators).

### `docs/nesasm-compatibility.md`
NESASM assembler compatibility plan (bank model, dot-prefixed local labels, `name .macro` syntax, `LOW()`/`HIGH()` functions).

### `docs/z80-support-plan.md`
Z80 architecture implementation plan (instruction groups, resolver, profiles, assembler pipeline).

### `docs/z80-branch-changes.md`
Detailed changelog of Z80 branch development.

### `docs/m68000-support-plan.md`
M68000 architecture implementation plan (effective addressing, opcode encoding, size suffixes).
