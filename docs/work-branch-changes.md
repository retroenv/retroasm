# Work Branch Changes

This document tracks every file changed in the `work` branch compared to `main`. It must be kept up-to-date as features are developed. When extracting features to `main`, mark the relevant entries as merged.

**Branch:** `work`
**Base:** `main`
**Last updated:** 2026-03-12

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
- Added imports for Chip-8, M65816, M68000, Z80, and Z80 profile packages
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

## Architecture: M65816 (`pkg/arch/m65816/`) — ALL NEW

### `pkg/arch/m65816/m65816.go`
**Why:** New architecture support for WDC 65C816 (65816) CPU.
**What:** Architecture entry point. Uses `*m65816.Instruction` as generic type T (same as M6502). 24-bit address width. Little-endian opcode output.

### `pkg/arch/m65816/m65816_test.go`
**Why:** Integration tests for M65816 assembly.
**What:** Tests implied, immediate, direct page, absolute, absolute long, accumulator, indirect long, stack relative, block move, branch, and BRL instructions.

### `pkg/arch/m65816/parser/addressing.go`
**Why:** M65816 addressing mode constants and helpers.
**What:** Defines ambiguous addressing mode combinations (AbsoluteDirectPageAddressing, XAddressing, YAddressing). Supports `a:`/`z:`/`f:` address size prefixes. Handles X, Y, S second operand dispatch.

### `pkg/arch/m65816/parser/instruction.go`
**Why:** M65816 instruction parsing.
**What:** Parses all 21 addressing modes including square bracket indirect long (`[dp]`, `[dp],Y`), stack relative (`sr,S`, `(sr,S),Y`), block move (`MVN $src,$dst`), and three-way DP/Absolute/Long disambiguation.

### `pkg/arch/m65816/assembler/address_assigning_step.go`
**Why:** M65816 address assignment in assembler pipeline.
**What:** Assigns addresses based on `BaseSize` from instruction definitions. Resolves ambiguous DP-vs-Absolute addressing by checking if value fits in a byte.

### `pkg/arch/m65816/assembler/generate_opcode_step.go`
**Why:** M65816 opcode generation.
**What:** Encodes all addressing modes: byte (DP, immediate, stack relative), word (absolute), long (24-bit), relative (8-bit), relative long (16-bit for BRL/PER), and block move (dst,src byte order).

### `pkg/arch/m65816/assembler/generate_opcode_step_test.go`
**Why:** Unit tests for M65816 opcode generation.
**What:** Tests byte addressing, long address encoding, relative long offset, and block move byte order.

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

---

## Compatibility Mode Infrastructure — NEW

### `pkg/assembler/config/compatibility.go` (new)
**Why:** Support multiple legacy assembler syntaxes (x816, asm6, ca65, NESASM) with a shared infrastructure.
**What:** Defines `CompatibilityMode` enum (`CompatDefault`, `CompatX816`, `CompatAsm6`, `CompatCa65`, `CompatNesasm`), `ParseCompatibilityMode()` parser, `String()` method, and feature query methods (`ColonOptionalLabels()`, `AnonymousLabels()`, `AsteriskProgramCounter()`, `BankByteOperator()`).

### `pkg/assembler/config/compatibility_test.go` (new)
**Why:** Test coverage for compatibility mode parsing and feature queries.
**What:** Tests for `ParseCompatibilityMode` (valid modes, case insensitivity, whitespace trimming, errors), `String()` for all modes, and feature query methods for all mode combinations.

### `pkg/assembler/config/config.go`
**Why:** Thread compatibility mode through the assembler pipeline.
**What:** Added `CompatibilityMode` field to `Config[T]`.

### `cmd/retroasm/main.go`
**Why:** CLI flag for selecting assembler compatibility mode.
**What:** Added `--compat` / `-m` flag, `parseCompatMode()` helper, compat mode logging, and threading compat mode through `registerArchitectureForCPU()` to `Config[T]`.

### `pkg/parser/parser.go`
**Why:** Core parser changes for compatibility mode features.
**What:**
- Added `compatMode`, `handlers`, and anonymous label tracking fields to `Parser[T]`
- Updated `New()` and `NewWithTokens()` constructors to accept `config.CompatibilityMode`
- Extracted `parseToken()` from `TokensToAstNodes()` to reduce function length
- Added `token.Plus`/`token.Minus` handling for anonymous `+`/`-` label definitions
- Added `token.Asterisk` handling for `* = value` program counter assignments
- Added `parseAnonymousLabel()` for generating synthetic label names from +/- tokens
- Added `parseAsteriskPC()` for `* = value` program counter assignment
- Added `isColonOptionalLabel()` for recognizing labels without trailing colon in x816/asm6 modes
- Updated `parseDot()` and `parseAlias()` to use per-instance `handlers` map instead of global

### `pkg/parser/directives/directives.go`
**Why:** Mode-specific directive registration and no-op handler.
**What:**
- Added `BuildHandlers()` function that builds mode-specific directive maps by overlaying mode-specific handlers on base handlers
- Added `NoOp()` handler that consumes tokens until EOL without producing AST nodes
- Added `x816Handlers()`, `asm6Handlers()`, `ca65Handlers()`, `nesasmHandlers()` for mode-specific directives
- Added `mergeHandlers()` using `maps.Copy`
- Deprecated global `Handlers` variable in favor of `BuildHandlers()`

### `pkg/parser/directives/noop_test.go` (new)
**Why:** Test coverage for NoOp directive and BuildHandlers.
**What:** Tests NoOp handler token consumption, and verifies mode-specific handler maps for default, x816, and ca65 modes.

### `pkg/parser/directives/helper.go`
**Why:** Export token reading for use by parser package.
**What:** Added `ReadDataTokensExported()` wrapper around `readDataTokens()`.

### `pkg/parser/alias.go`
**Why:** Use per-instance directive handlers instead of global.
**What:** Changed directive lookup to use `p.handlers` instead of `directives.Handlers`. Removed unused `directives` import.

### `pkg/assembler/assembler.go`
**Why:** Pass compatibility mode to parser.
**What:** Updated `Process()` to pass `asm.cfg.CompatibilityMode` to `parser.New()`.

### `pkg/assembler/process_macros_step.go`
**Why:** Pass compatibility mode to parser during macro expansion.
**What:** Updated `macroTokensToAStNodes()` to pass `asm.cfg.CompatibilityMode` to `parser.NewWithTokens()`.

---

## asm6/asm6f Compatibility Features — NEW

### `pkg/parser/ast/configuration.go`
**Why:** AST representation for NES 2.0 header configuration items.
**What:** Added `ConfigNes2ChrRAM`, `ConfigNes2PrgRAM`, `ConfigNes2Sub`, `ConfigNes2TV`, `ConfigNes2VS`, `ConfigNes2BRam`, `ConfigNes2ChrBRam` constants to `ConfigurationItem` enum.

### `pkg/parser/directives/nesasm.go`
**Why:** NES 2.0 header directive parsing (asm6f extension).
**What:** Added `nes2Directives` map and `Nes2Config()` handler function that converts NES 2.0 directives to AST configuration nodes (following the existing `NesasmConfig` pattern).

### `pkg/parser/directives/directives.go`
**Why:** Register asm6f-specific directives.
**What:** Updated `asm6Handlers()` to include:
- `UNSTABLE`, `HUNSTABLE` — undocumented opcode tier directives (no-op)
- `IGNORENL`, `ENDINL` — symbol file control (no-op)
- `NES2CHRRAM`, `NES2PRGRAM`, `NES2SUB`, `NES2TV`, `NES2VS`, `NES2BRAM`, `NES2CHRBRAM` — NES 2.0 header directives

### `pkg/assembler/config/compatibility.go`
**Why:** Feature flag for `@` local label scoping.
**What:** Added `LocalLabelScoping()` method returning `true` for `CompatAsm6`.

### `pkg/arch/arch.go`
**Why:** Allow architecture-specific parsers to apply `@` local label scoping.
**What:** Added `ScopeLocalLabel(name string) string` method to `Parser` interface.

### `pkg/parser/parser.go`
**Why:** `@` local label scoping between non-local labels.
**What:**
- Added `lastNonLocalLabel` field to `Parser[T]` struct
- Added `scopeLocalLabel()` method that prefixes `@`-names with the current non-local label scope
- Added `updateLabelScope()` method to track non-local labels
- Added `ScopeLocalLabel()` exported method implementing `arch.Parser` interface
- Updated label creation in `parseIdentifier()` (colon and colon-optional paths) to apply scoping

### `pkg/parser/directives/data.go`
**Why:** Scope `@` local label references in data expression tokens.
**What:** Updated `readDataTokens()` to apply `ScopeLocalLabel()` on identifier tokens.

### `pkg/arch/m6502/parser/instruction.go`
**Why:** Scope `@` local label references in instruction operands.
**What:** Updated `parseInstruction()` to scope `arg1.Value` when it's an identifier. Updated `parseInstructionImmediateAddressingWithToken()` to scope identifier token values.

### `pkg/assembler/parse_ast_nodes.go`
**Why:** Source file inclusion support.
**What:**
- Split `parseInclude()` into `parseBinaryInclude()` and `parseSourceInclude()`
- `parseSourceInclude()` reads the file, creates a parser with the same architecture and compat mode, lexes and parses the included file, then recursively processes the resulting AST nodes

### Tests
- `pkg/assembler/config/compatibility_test.go` — Added `LocalLabelScoping()` test
- `pkg/parser/parser_asm6_test.go` — Added `TestParserAsm6LocalLabelScoping` and `TestParserAsm6LocalLabelScopingDisabledInDefault`; updated existing tests to use `config.CompatAsm6` mode
- `pkg/parser/directives/noop_test.go` — Added `TestBuildHandlers_Asm6` and `TestNes2Config`
- `pkg/assembler/assembler_asm6_test.go` — Added `TestAssemblerAsm6SourceInclude`
- `pkg/arch/z80/parser/mock_parser_test.go` — Added `ScopeLocalLabel()` to mock
- `pkg/parser/directives/directives_test.go` — Added `ScopeLocalLabel()` to mock

---

## ca65 Compatibility Features — NEW

### `pkg/parser/ast/scope.go` (new)
**Why:** AST nodes for ca65-style `.scope`/`.endscope` blocks.
**What:** Added `Scope` (with `Name` field) and `ScopeEnd` struct types with constructors (`NewScope`, `NewScopeEnd`) and `Copy()` methods.

### `pkg/parser/ast/data.go`
**Why:** Support bank byte (bits 16-23) address references for `.bankbytes`/`.faraddr`.
**What:** Added `BankAddressByte` constant to `ReferenceType` enum.

### `pkg/parser/directives/ca65.go` (new)
**Why:** ca65-specific directive handler implementations.
**What:** Added handlers:
- `Scope()` — parses `.scope [name]` with optional name
- `EndScope()` — parses `.endscope`
- `Asciiz()` — parses `.asciiz` null-terminated string data (appends `0` token)
- `FarAddr()` — parses `.faraddr` 24-bit address data (width=3)
- `BankBytes()` — parses `.bankbytes` bank byte address emission
- `Warning()` — parses `.warning` diagnostic messages (reuses `ast.Error`)
- `Out()` — parses `.out` print messages (no-op, consumes tokens)

### `pkg/parser/directives/directives.go`
**Why:** Register ca65-specific directives.
**What:** Updated `ca65Handlers()` to include:
- Scoping: `scope`, `endscope`
- Data: `asciiz`, `faraddr`, `bankbytes`, `hibytes`, `lobytes`
- Repeat aliases: `repeat`→`Rept`, `endrepeat`→`Endr`
- Diagnostics: `warning`, `fatal`→`Error`, `out`, `assert`→`NoOp`
- No-op stubs: `list`, `listbytes`, `debuginfo`, `export`, `exportzp`, `import`, `importzp`, `global`, `globalzp`, `feature`, `charmap`, `autoimport`, `local`, `condes`, `linecont`, `define`, `undefine`

### `pkg/parser/directives/macro.go`
**Why:** Support `.endmacro` as macro terminator (ca65 syntax).
**What:** Added detection of `.endm` and `.endmacro` inside the macro token reader via dot+identifier pattern check.

### `pkg/parser/parser.go`
**Why:** ca65 unnamed label parsing and unnamed label reference disambiguation.
**What:**
- Added `unnamedLabelCount` field for tracking ca65-style `:` label definitions
- Added `parseUnnamedLabel()` generating synthetic `__unnamed_N` names
- Added `token.Colon` case in `parseToken()` for unnamed label definitions
- Added `isUnnamedLabelRef()` to disambiguate `:` as label-colon vs unnamed-reference prefix
- Updated `parseIdentifier()` to check `isUnnamedLabelRef()` before treating colon as label definition
- Added `ResolveUnnamedLabel()` method for resolving `:-`/`:+` references to synthetic names

### `pkg/assembler/config/compatibility.go`
**Why:** Feature flags for ca65-specific parsing behavior.
**What:**
- Updated `LocalLabelScoping()` to return `true` for `CompatCa65` (in addition to `CompatAsm6`)
- Added `UnnamedLabels()` method returning `true` for `CompatCa65`

### `pkg/arch/arch.go`
**Why:** Support unnamed label reference resolution from architecture-specific parsers.
**What:** Added `ResolveUnnamedLabel(forward bool, level int) string` method to `Parser` interface.

### `pkg/arch/m6502/parser/instruction.go`
**Why:** Handle ca65-style unnamed label references (`:-`, `:+`, `:--`, `:++`) in instruction operands.
**What:** Added `resolveUnnamedLabelRef()` helper that detects `Colon`+`Plus`/`Minus` token sequences and resolves them to synthetic `__unnamed_N` label names. Called from `parseInstruction()` when `arg1` is a colon token.

### `pkg/assembler/nodes.go`
**Why:** Support bank address byte references in assembler pipeline.
**What:** Added `bankAddressByte` constant to internal `referenceType` enum.

### `pkg/assembler/parse_ast_nodes.go`
**Why:** Handle `Scope`/`ScopeEnd` AST nodes and `BankAddressByte` in assembler pipeline.
**What:**
- Added `ast.Scope` and `ast.ScopeEnd` cases in `parseASTNode()` switch
- Added `parseScope()` — creates child scope; if named, also creates a symbol in parent scope
- Added `parseScopeEnd()` — restores parent scope (mirrors `parseFunctionEnd()`)
- Added `ast.BankAddressByte` case in `parseData()` setting `bankAddressByte` ref type and width=1

### `pkg/assembler/generate_opcode_step.go`
**Why:** Emit bank address byte (bits 16-23) for references.
**What:** Added `bankAddressByte` case in `generateReferenceDataBytes()` extracting `byte(address >> 16)`.

### Tests
- `pkg/assembler/config/compatibility_test.go` — Updated `LocalLabelScoping()` test for ca65, added `UnnamedLabels()` test
- `pkg/parser/parser_ca65_test.go` (new) — Tests for unnamed label definition/reference, `@` local scoping, `.scope`/`.endscope`, `.asciiz`, `.warning`, `.endmacro`, and all no-op directives
- `pkg/arch/z80/parser/mock_parser_test.go` — Added `ResolveUnnamedLabel()` to mock
- `pkg/parser/directives/directives_test.go` — Added `ResolveUnnamedLabel()` to mock

---

## NESASM Compatibility Features — NEW

### `pkg/assembler/config/compatibility.go`
**Why:** Feature flags for NESASM-specific parsing behavior.
**What:**
- Added `DotLocalLabels()` method returning `true` for `CompatNesasm`
- Added `NesasmMacroSyntax()` method returning `true` for `CompatNesasm`

### `pkg/lexer/token/token.go`
**Why:** Support `\1`-`\9` positional parameter references in NESASM macros.
**What:** Added `Backslash` token type with `'\\'` mapping in `toToken` and `"\\"` in `toString`.

### `pkg/parser/parser.go`
**Why:** NESASM dot-prefixed local labels and `name .macro` syntax.
**What:**
- Added `parseDotLocalLabel()` for `.label` definitions scoped between non-local labels
- Added `scopeDotLocalLabel()` for prefixing dot-local names with current scope
- Added `ResolveDotLocalLabel()` exported method implementing `arch.Parser` interface
- Updated `parseDot()` to fall back to `parseDotLocalLabel()` when directive lookup fails in NESASM mode
- Added `parseNesAsmMacro()` for `name .macro` syntax with `\1`-`\9` positional parameters
- Updated `parseIdentifier()` to detect NESASM macro definitions (`name .macro`)
- Updated `updateLabelScope()` to handle NESASM `DotLocalLabels()` mode

### `pkg/arch/arch.go`
**Why:** Allow architecture-specific parsers to resolve NESASM dot-local label references.
**What:** Added `ResolveDotLocalLabel(name string) string` method to `Parser` interface.

### `pkg/arch/m6502/parser/instruction.go`
**Why:** Handle NESASM dot-local label references (`.label`) in instruction operands.
**What:**
- Extracted `resolveArg1Token()` helper from `parseInstruction()` to reduce cyclop complexity
- Added `resolveDotLocalLabelRef()` that detects `Dot`+`Identifier` sequences and resolves via `ResolveDotLocalLabel()`
- `resolveArg1Token()` chains: identifier scoping → unnamed label refs → dot-local label refs

### `pkg/assembler/process_macros_step.go`
**Why:** Support NESASM positional macro expansion with `\1`-`\9` parameters.
**What:**
- Split `resolveMacroUsage()` into `resolveNamedMacro()` (standard) and `resolvePositionalMacro()` (NESASM)
- `resolvePositionalMacro()` scans for `Backslash`+`Number` token pairs and substitutes with caller arguments

### `pkg/parser/directives/directives.go`
**Why:** Register NESASM-specific directives.
**What:** Added `nesasmHandlers()` with:
- `.ds` — storage directive (maps to `DataStorage`)
- `.endp` — procedure end alias (maps to `EndProc`)
- `.fail` — error directive (maps to `Error`)
- `.list`, `.nolist`, `.mlist`, `.nomlist`, `.opt` — listing control (no-op)
- `.zp`, `.bss`, `.code`, `.data` — section switching stubs (no-op)

### `pkg/parser/directives/data.go`
**Why:** Support `.ds` as NESASM storage directive.
**What:** Added `"ds": 1` entry to `dataByteWidth` map.

### Tests
- `pkg/assembler/config/compatibility_test.go` — Split `TestCompatibilityMode_Features` into two functions for funlen compliance; added `DotLocalLabels()` and `NesasmMacroSyntax()` tests
- `pkg/parser/parser_nesasm_test.go` (new) — Tests for dot-local label definition/scoping, NESASM macro definition, no-op directives, `.fail`, and dot-local-not-in-default-mode
- `pkg/arch/z80/parser/mock_parser_test.go` — Added `ResolveDotLocalLabel()` to mock
- `pkg/parser/directives/directives_test.go` — Added `ResolveDotLocalLabel()` to mock

---

## x816 Compatibility Features — NEW

### `pkg/lexer/token/token.go`
**Why:** Support bitwise/shift operators in expressions for x816 keyword operators.
**What:** Added `ShiftLeft`, `ShiftRight`, `Ampersand`, `BitwiseXor` token types with toString entries.

### `pkg/lexer/token/type.go`
**Why:** Register new bitwise/shift token types as expression operators.
**What:** Added `Pipe`, `ShiftLeft`, `ShiftRight`, `Ampersand`, `BitwiseXor` to operators set.

### `pkg/expression/operator.go`
**Why:** Evaluate bitwise/shift operators in expressions.
**What:**
- Added `ShiftLeft`, `ShiftRight`, `Ampersand`, `Pipe`, `BitwiseXor` to `operatorPriority` map
- Added `evaluateBitwiseIntInt()` for shift/bitwise int64 operations
- Updated `evaluateOperatorIntInt()` to delegate bitwise ops (reduces cyclop complexity)

### `pkg/expression/expression.go`
**Why:** Support x816 keyword operators (`SHL`, `SHR`, `AND`, `OR`, `XOR`) in expressions.
**What:**
- Added `keywordOperators` map converting keyword identifiers to operator token types
- Added `resolveKeywordOperator()` that converts keyword identifiers before RPN processing
- Updated `parseToRPN()` to use `resolveKeywordOperator()` (reduces gocognit complexity)

### `pkg/parser/directives/directives.go`
**Why:** Register x816-specific directives.
**What:** Expanded `x816Handlers()` with:
- Data: `.dcl`/`.dl` (3-byte), `.dcd`/`.dd` (4-byte), `.dsl` (3-byte storage), `.dsd` (4-byte storage)
- Include: `.src` (source include alias)
- Comment: `.comment` (multi-line comment block, skip to `.end`)
- Block terminator: `.end` (no-op)
- Bitwidth: `.mem`, `.index` (no-op for NES)
- Optimization/listing: `.opt`, `.optimize`, `.list`, `.nolist`, `.sym`, `.symbol`, `.detect`, `.dasm`, `.echo`
- Diagnostics: `.cerror`, `.cwarn`, `.message`
- ROM/output: `.hrom`, `.lrom`, `.hirom`, `.smc` (no-op for NES)
- Settings: `.localsymbolchar`, `.locchar`, `.par`, `.parenthesis` (no-op)

### `pkg/parser/directives/x816.go` (new)
**Why:** Multi-line comment block handler for x816.
**What:** Added `CommentBlock()` handler that skips all tokens until a matching `.end` directive is found.

### `pkg/parser/directives/data.go`
**Why:** Support x816 long (3-byte) and double-word (4-byte) data directives.
**What:** Added `"dcl": 3`, `"dl": 3`, `"dcd": 4`, `"dd": 4`, `"dsl": 3`, `"dsd": 4` to `dataByteWidth` map.

### `pkg/parser/parser.go`
**Why:** Support x816 `.equ` alias syntax and consolidate dot-identifier patterns.
**What:**
- Added `isDotIdentifierKeyword()` to check for recognized `name .keyword` patterns (equ, rs, macro)
- Added `parseDotIdentifier()` to handle consolidated `name .keyword` patterns
- Refactored `parseIdentifier()` to use `parseDotIdentifier()` (reduces cyclop complexity)
- Fixed `isColonOptionalLabel()` column check from `!= 0` to `> 1` (lexer uses 1-based columns)

### Tests
- `pkg/parser/parser_x816_test.go` (new) — Tests for no-op directives, `.comment` block, `.src` include, `.equ` alias, colon-optional labels, anonymous labels, `.end` directive
- `pkg/expression/keyword_operator_test.go` (new) — Tests for keyword operators (SHL, SHR, AND, OR, XOR), case insensitivity, and bitwise token type operators (Pipe, Ampersand, ShiftLeft, ShiftRight)
- `pkg/parser/directives/noop_test.go` — Updated `TestBuildHandlers_X816` with new x816-specific directive checks (mem, index, opt, symbol, dcl, dcd, src, comment)
