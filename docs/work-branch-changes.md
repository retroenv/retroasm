# Work Branch Changes

This document tracks every file changed in the `work2` branch compared to `main`. It must be kept up-to-date as features are developed. When extracting features to `main`, mark the relevant entries as merged.

**Branch:** `work2`
**Base:** `main`
**Last updated:** 2026-05-01

`origin/main` was merged into `work2` on 2026-04-30. The entries below describe the remaining `work2` delta after that sync.

---

## Refresh Notes

This document was checked against `git diff --name-only origin/main...HEAD` on 2026-05-01.

Current remaining branch-only delta: 141 files.

Status refresh after the 2026-05-01 review:
- The earlier note that Group 4 had been "applied to `main` worktree" was stale. Those files still differ from `origin/main`.
- Group 4 is now marked merged in this progress plan. The file count above still reflects the live working tree until the extraction is actually committed to `main`.
- Group 5 has narrowed: the only architecture-agnostic extraction still intended for this slice is source `.include` recursion in `pkg/assembler/parse_ast_nodes.go` plus its assembler test coverage.
- The remaining `work2` diffs in `pkg/assembler/assembler.go`, `pkg/assembler/process_macros_step.go`, and `pkg/parser/ast/configuration.go` belong to later compatibility groups even though older notes mentioned them near Group 5.
- On 2026-05-01, the source-include part of Group 5 was applied to the `main` working tree without commit. Until that is committed on `main`, live diff counts will continue to show those hunks in `work2`.
- No planned merge group has disappeared completely from the remaining branch delta yet.
- Early groups are reduced compared with older branch snapshots, but they still contain work left to extract.

## Merge Plan to Main

Planned extraction is grouped to minimize risk and keep each merge window reviewable.

### Group 1: Foundation Sanity
**Goal:** make the branch safe to extract before large feature files are introduced.

**Status:** still remaining. The branch delta still includes `go.mod`, `go.sum`, and `.gitignore`.

- Merge `go.mod`/`go.sum` updates only if they are required by already-approved work. Keep dependency changes that exist only for deferred architecture waves (Chip-8, Z80, M68000, or others not yet extracted) out of `main` until those waves are ready. Remove the local `replace retrogolib` directive before the final step.
- Merge only foundation config that is architecture-agnostic at this stage. Keep Z80-specific fixture tracking in `.gitignore` out of this group.
- Run a baseline validation on `main` with only foundation changes: `make lint`, `go test ./pkg/...` (or minimal focused packages where needed).
- Verify nothing in this group depends on new architecture packages.

### Group 2: CLI Multi-Architecture Plumbing
**Goal:** split the command-line architecture expansion away from parser compatibility work.

**Status:** still remaining. `cmd/retroasm/architecture.go`, `cmd/retroasm/assemble.go`, `cmd/retroasm/main.go`, and `cmd/retroasm/main_test.go` still differ; deferred Z80-specific CLI coverage also remains in `cmd/retroasm/z80_fixture_test.go`.

Manual diff notes:
- `cmd/retroasm/architecture.go` is no longer just generic CPU/system normalization. The remaining diff also registers `m65816`, `m68000`, `sm83`, and `z80`, threads compatibility mode into architecture configs, and adds Z80-profile parsing/defaulting/validation. Group 2 is therefore still mixed with later architecture-wave and compatibility work.
- `cmd/retroasm/assemble.go` now parses `--compat`, routes Chip-8 through a direct low-level assembler path, and passes both compatibility mode and Z80 profile into architecture registration. The direct `assembleChip8File()` path belongs with the Chip-8 extraction, not with a pure generic CLI slice.
- `cmd/retroasm/main.go` adds `z80Profile` and `compat` fields to CLI options and logs them. That means the remaining delta is not limited to architecture-agnostic flag plumbing.
- `cmd/retroasm/main_test.go` has expanded well past basic CPU/system validation. The remaining tests cover defaulting for new architectures, Z80-profile compatibility/errors, and config-driven assembly through the new registration path.
- `cmd/retroasm/z80_fixture_test.go` is entirely branch-only and is clearly Group 12 material: fixture-based Z80 integration, profile acceptance/rejection coverage, and resolver-path smoke tests.

- Merge the generic CLI plumbing across `cmd/retroasm/architecture.go`, `cmd/retroasm/assemble.go`, `cmd/retroasm/main.go`, and `cmd/retroasm/main_test.go` that adds CPU/system normalization, defaulting, validation, and non-dialect-specific assembly flow.
- Keep architecture-specific CLI behavior out of this group until the corresponding architecture wave lands. That includes `--z80-profile`, Z80 registration branches, Z80-only validation/tests, and one-off paths like `assembleChip8File()` if Chip-8 itself is not in the same extraction.
- Keep `--compat` parsing and compatibility-mode wiring out of this group.
- Keep `cmd/retroasm/z80_fixture_test.go` out of this group; it belongs with the Z80 wave.
- Validate with `go test ./cmd/retroasm -run 'Architecture|CPU|System'` or the closest focused subset, then `go test ./cmd/retroasm/...`.

### Group 3: High-Level API Generalization
**Goal:** make `pkg/retroasm` and the generic assembler path architecture-agnostic before dialect work lands.

**Status:** still remaining. `pkg/retroasm/default.go` and `pkg/assembler/assembler.go` still differ from `origin/main`.

Manual diff notes:
- `pkg/assembler/assembler.go` contains the only substantive remaining behavior change in this group: `Assembler.Process()` now calls `parser.New[T](..., asm.cfg.CompatibilityMode)`. That is compatibility-mode threading, not broad architecture-dispatch work.
- `pkg/retroasm/default.go` no longer carries a meaningful architecture-generalization delta. The remaining diff there is effectively formatting/comment cleanup, not a real feature gap.
- Compared with the original plan, Group 3 is much smaller now. The remaining real work is mostly the parser-construction signature change needed by compatibility mode, which makes this group overlap Group 6 more than earlier revisions implied.

- Merge `pkg/retroasm/default.go`, `pkg/assembler/assembler.go`, and any minimal supporting changes needed for dynamic address width, architecture adapter config access, and generic assemble-text/assemble-AST dispatch.
- Keep compatibility-mode-specific parser changes out of this group.
- Validate with `go test ./pkg/retroasm/... ./pkg/assembler/...`.

### Group 4: AST and Opcode Plumbing
**Goal:** land the generic AST/assembler changes that architecture parsers and generators depend on, without bringing in dialect parsing yet.

**Status:** merged in the progress plan. `pkg/arch/arch.go`, `pkg/assembler/nodes.go`, `pkg/assembler/parse_ast_nodes.go`, and `pkg/assembler/generate_opcode_step.go` are the Group 4 shared AST/opcode plumbing slice.

Manual diff notes:
- `pkg/arch/arch.go` extends the generic `arch.Parser` interface with `ScopeLocalLabel`, `ResolveUnnamedLabel`, and `ResolveDotLocalLabel`. Those hooks are needed by compatibility-mode parsers, so this file now straddles opcode plumbing and dialect support.
- `pkg/assembler/nodes.go` adds `bankAddressByte` as a new generic reference kind, and `pkg/assembler/generate_opcode_step.go` emits the high third byte for that reference type. That part is concrete shared assembler plumbing.
- `pkg/assembler/parse_ast_nodes.go` contains the largest remaining Group 4 delta, but it is not limited to opcode threading. It now:
- handles `ast.Scope` and `ast.ScopeEnd`
- accepts `ast.BankAddressByte` data references
- supports source `.include` parsing by recursively invoking `parser.New(...)`
- threads compatibility mode through included-file parsing
- Because of that, the current remaining diff for this file overlaps Group 5 and Group 6 directly. Group 4 is no longer a clean “opcode-only” slice in the branch’s present state.

- Group 4 has been extracted to `main` in the progress plan.
- Keep the historical notes below as the record of what moved in this slice: `pkg/assembler/nodes.go`, `pkg/assembler/parse_ast_nodes.go`, `pkg/assembler/generate_opcode_step.go`, and `pkg/arch/arch.go`.
- The shared opcode-plumbing work included `OpcodeID` threading and register-value / register-register-value conversion support.
- Scope nodes, include-source recursion, bank-byte references, and compatibility parser APIs were intentionally left for later groups.
- Validate with parser/assembler unit tests on touched packages plus architecture tests that rely on opcode IDs.

### Group 5: Include, Scope, and Data-Pipeline Extensions
**Goal:** isolate assembler-pipeline enhancements that are not inherently tied to one compatibility mode.

**Status:** applied to the `main` working tree on 2026-05-01, pending commit there. In the current branch state, the only remaining architecture-agnostic work for this slice was source `.include` recursion in `pkg/assembler/parse_ast_nodes.go` plus the related assembler test coverage.

- Historical note: earlier Group 5 planning also mentioned scope/data AST and bank-byte support, but those pieces are already present on `main` and are no longer part of the live extraction delta.
- Extract only source `.include` recursion in `pkg/assembler/parse_ast_nodes.go` and its focused coverage in `pkg/assembler/assembler_asm6_test.go`.
- Keep parser constructor compatibility threading, macro reparsing changes, and NES 2.0 configuration constants out of this group; those belong to later compatibility slices.
- Validate with `go test ./pkg/assembler/...` and the focused source-include coverage.

### Group 6: Compatibility Framework
**Goal:** introduce the shared `--compat` infrastructure and parser/directive dispatch without enabling every dialect feature at once.

**Status:** still remaining. The shared parser/directive framework files still differ, including `pkg/parser/parser.go`, `pkg/parser/alias.go`, `pkg/parser/directives/directives.go`, `pkg/parser/directives/helper.go`, `pkg/parser/directives/macro.go`, and related tests.

- Merge `pkg/assembler/config/compatibility*`, `pkg/parser/directives/directives.go`, `pkg/parser/directives/helper.go`, `pkg/parser/alias.go`, `pkg/parser/parser.go` constructor/handler plumbing, and the `--compat` flag parsing in `cmd/retroasm/main.go`.
- Include only the shared parser behaviors needed across multiple modes: per-instance handler maps, `parseToken()` extraction, constructor signature changes, and compat-mode threading through macro reparsing.
- Exclude dialect-specific semantics such as asm6 local labels, ca65 unnamed labels, NESASM dot locals, and x816 operator/directive support.
- Validate with `go test ./pkg/parser/... ./pkg/assembler/... ./pkg/parser/directives/...` and focused CLI tests for `--compat`.

### Group 7: asm6 / asm6f Compatibility
**Goal:** land asm6-specific behavior independently from the other dialects.

**Status:** still remaining. asm6-related parser/assembler changes are still in the branch delta, including `pkg/parser/parser_asm6_test.go`, `pkg/assembler/assembler_asm6_test.go`, and the shared support they depend on.

- Merge asm6 local-label scoping, asm6f `NES2*` directives, source-include support used by asm6 `.include`, and asm6-specific tests in `pkg/parser/parser_asm6_test.go`, `pkg/assembler/assembler_asm6_test.go`, and `pkg/parser/directives/noop_test.go`.
- Files touched will include `pkg/parser/parser.go`, `pkg/parser/directives/data.go`, `pkg/parser/directives/nesasm.go`, `pkg/parser/directives/directives.go`, `pkg/parser/ast/configuration.go`, and `pkg/assembler/parse_ast_nodes.go`.
- Validate with `go test ./pkg/parser/... -run Asm6` and `go test ./pkg/assembler/... -run Asm6`.

### Group 8: ca65 Compatibility
**Goal:** isolate ca65 scope/data/label semantics into a reviewable extraction.

**Status:** still remaining. ca65-specific directive and parser coverage still differs, including `pkg/parser/directives/ca65.go`, `pkg/parser/parser_ca65_test.go`, and the shared bank-byte-capable assembler/parser changes they rely on.

- Merge ca65 unnamed labels, `.scope`/`.endscope`, `.asciiz`, `.faraddr`, `.bankbytes`, `.warning`, `.out`, `.endmacro`, and the ca65 no-op directive surface.
- Files touched will include `pkg/parser/directives/ca65.go`, `pkg/parser/directives/macro.go`, `pkg/parser/parser.go`, `pkg/parser/ast/scope.go`, `pkg/parser/ast/data.go`, `pkg/assembler/nodes.go`, `pkg/assembler/parse_ast_nodes.go`, and `pkg/assembler/generate_opcode_step.go`.
- Validate with `go test ./pkg/parser/... -run Ca65` and focused assembler tests covering bank-byte emission and scope behavior.

### Group 9: NESASM Compatibility
**Goal:** keep NESASM dot-local and positional-macro behavior separate from asm6/ca65/x816.

**Status:** still remaining. NESASM-specific directive/parser changes still differ, including `pkg/parser/directives/nesasm.go` and `pkg/parser/parser_nesasm_test.go`.

- Merge NESASM dot-local labels, `name .macro`, positional `\\1`-`\\9` macro substitution, `.ds`, `.endp`, `.fail`, and listing/section no-op directives.
- Files touched will include `pkg/parser/parser.go`, `pkg/assembler/process_macros_step.go`, `pkg/parser/directives/data.go`, `pkg/parser/directives/directives.go`, and `pkg/lexer/token/token.go`.
- Validate with `go test ./pkg/parser/... -run Nesasm` and targeted macro-expansion tests in `pkg/assembler/...`.

### Group 10: x816 Compatibility
**Goal:** isolate x816 syntax and expression support from the other dialects.

**Status:** still remaining. x816-specific directive/parser changes still differ, including `pkg/parser/directives/x816.go` and `pkg/parser/parser_x816_test.go`.

- Merge colon-optional labels, anonymous `+`/`-` labels, `* = value`, `name .equ`, x816 no-op directives, `.comment` blocks, `.src`, 3-byte/4-byte data directives, and keyword bitwise operators.
- Files touched will include `pkg/parser/parser.go`, `pkg/parser/directives/x816.go`, `pkg/parser/directives/directives.go`, `pkg/parser/directives/data.go`, `pkg/lexer/token/token.go`, `pkg/lexer/token/type.go`, `pkg/expression/expression.go`, and `pkg/expression/operator.go`.
- Validate with `go test ./pkg/parser/... -run X816` and `go test ./pkg/expression/...`.

### Group 11: Architecture Wave A (Low-Risk)
**Goal:** land smaller/contained architecture additions after shared layers are stable, but still keep each architecture reviewable.

**Status:** still remaining. The branch delta still includes the Chip-8, M65816, SM83, and x86 architecture packages, their tests, and their supporting examples/docs.

- Do not treat this as one giant merge. Extract Chip-8, M65816, SM83, and x86 as separate PRs or merge commits within the same wave.
- Merge each architecture package with its own deferred integration points. Examples: Chip-8 gets `assembleChip8File()`, any Chip-8-specific CLI registration/tests, `examples/chip8/`, and Chip-8 README/docs updates in the same slice; SM83 gets its docs with SM83; M65816 gets its docs with M65816; x86 gets any x86-specific CLI/docs with x86.
- Keep architecture-specific README claims out of earlier generic groups. Only document support for an architecture when that architecture has actually landed on `main`.
- Validate per architecture rather than as a single batched step: run `go test ./pkg/arch/chip8/...`, `./pkg/arch/m65816/...`, `./pkg/arch/sm83/...`, or `./pkg/arch/x86/...` for the architecture being extracted, plus the focused CLI tests that reference it.

### Group 12: Architecture Wave B (Higher Complexity)
**Goal:** extract the largest parser/resolver-heavy architecture once lower-risk waves are stable.

**Status:** still remaining. The branch delta still includes `pkg/arch/z80/`, `pkg/arch/z80/profile/`, Z80 fixtures under `tests/z80/`, Z80 CLI coverage, and the related docs/README updates.

- Merge `pkg/arch/z80/` and `pkg/arch/z80/profile/` incrementally with parser/assembler tests and CLI profile plumbing.
- Merge all deferred Z80-only integration points here: CLI registration, `--z80-profile`, Z80-specific validation/tests in `cmd/retroasm/main_test.go`, `cmd/retroasm/z80_fixture_test.go`, `.gitignore` exceptions for `tests/z80/`, Z80 fixture files, and Z80-specific README/docs updates.
- Validate with `go test ./pkg/arch/z80/...`.
- Validate with `go test ./cmd/retroasm -run Z80` through fixture-driven integration paths.
- Validate with `go test ./cmd/retroasm/...`.
- Keep this in a separate PR or merge commit if review load is high.

### Group 13: Architecture Wave C (M68000) and Cross-Checks
**Goal:** add the final large architecture and ensure end-to-end behavior.

**Status:** still remaining. The branch delta still includes the full `pkg/arch/m68000/` slice plus its supporting docs and integration points.

- Merge `pkg/arch/m68000/` plus parser/addressing and assembler coverage tests.
- Merge all deferred M68000-specific integration points here: CLI registration/tests, any dependency updates needed only for M68000 support, and M68000 README/docs updates.
- Validate with `go test ./pkg/arch/m68000/...` and cross-arch smoke tests from previous groups.
- Run full project checks (`make lint`, `make test`) and fix any integration-level regressions before removing compatibility flags or command-line leftovers.

### Group 14: Documentation and Finalization
**Goal:** close branch metadata and production readiness.

**Status:** still remaining. The branch still carries branch-planning documentation, compatibility plans, and README updates that are not yet in `origin/main`.

- Merge only branch-level or cross-cutting documentation here (`docs/compatibility-mode-plan.md`, `docs/x816-compatibility-plan.md`, and any truly architecture-independent `README.md` cleanup).
- Do not leave architecture-specific documentation queued here. Ship Z80 docs with Group 12, Wave A architecture docs with the corresponding Wave A extraction, and M68000 docs with Group 13.
- Update `docs/work-branch-changes.md` as you complete each group by marking entries as merged.
- Final verification on `main`: full test pass, one clean `go test ./...` run, then remove temporary branch-only notes (including any `replace` directives).
- Merge `work2` into `main` and keep a short post-merge log for any follow-up cleanup.

### Suggested Order of PRs
1. Foundation Sanity
2. CLI Multi-Architecture Plumbing
3. High-Level API Generalization
4. AST and Opcode Plumbing
5. Include, Scope, and Data-Pipeline Extensions
6. Compatibility Framework
7. asm6 / asm6f Compatibility
8. ca65 Compatibility
9. NESASM Compatibility
10. x816 Compatibility
11. Architecture Wave A (chip8/m65816/sm83/x86)
12. Architecture Wave B (z80 + profile)
13. Architecture Wave C (m68000)
14. Documentation and Finalization

| Group | Merge Criteria | Primary Validation |
|-------|----------------|-------------------|
| Foundation Sanity | No architecture-specific behavior introduced yet | `make lint`, core package tests |
| CLI Multi-Architecture Plumbing | CLI accepts/normalizes generic cpu/system combinations without deferred architecture-specific branches | `cmd/retroasm` focused tests |
| High-Level API Generalization | `pkg/retroasm` can dispatch by registered architecture | `pkg/retroasm` + `pkg/assembler` tests |
| AST and Opcode Plumbing | Generic AST/opcode APIs compile without dialect features | Parser/assembler + touched arch tests |
| Include, Scope, and Data-Pipeline Extensions | Include/scope/reference-byte flows work in isolation | Focused parser/assembler tests |
| Compatibility Framework | `--compat` and mode-specific handler dispatch compile cleanly | Parser/directive framework tests |
| asm6 / asm6f Compatibility | asm6 parser and include suite green | asm6 parser/assembler tests |
| ca65 Compatibility | ca65 labels/scopes/data features green | ca65 parser + bank-byte tests |
| NESASM Compatibility | dot-local and positional macro paths green | NESASM parser/macro tests |
| x816 Compatibility | x816 syntax and expression operators green | x816 parser + expression tests |
| Architecture Wave A | One architecture slice at a time, including its own CLI/docs/examples | Per-package architecture tests plus focused CLI coverage |
| Architecture Wave B | All Z80-specific code and docs land together | Full fixture + CLI profile validation |
| Architecture Wave C | M68000 code, CLI wiring, and docs land together | M68000 package tests + full lint/test |
| Documentation/Finalization | No pending branch-only TODOs | `make lint`, `make test` full run |

## Build & Configuration

### `.gitignore`
**Why:** Allow Z80 test fixture `.asm` files to be tracked while keeping other test artifacts ignored.
**What:** Changed `tests/` exclusion to `tests/*` with explicit `!tests/z80/` and `!tests/z80/*.asm` exceptions.

### `go.mod`
**Why:** Development requires local retrogolib changes (new Z80/M68000/Chip-8 CPU definitions).
**What:** Added `replace` directive pointing `retrogolib` to local checkout. **Must be removed before merging to main.**

### `README.md`
**Why:** Document newly supported architectures and updated CLI options.
**What:** Added architecture support section (6502, Chip-8, Z80, M68000). Updated CLI usage examples and flag descriptions to reflect multi-architecture support.

---

## CLI (`cmd/retroasm/`)

### `cmd/retroasm/architecture.go`
**Why:** Split architecture normalization, defaulting, validation, and adapter selection out of `main.go`.
**What:** Holds the multi-architecture CLI plumbing that maps CPU/system inputs to the correct runtime configuration, including Z80-profile-aware validation and architecture registration helpers.

### `cmd/retroasm/assemble.go`
**Why:** Split assembly execution paths out of `main.go` as CLI support broadened beyond the original 6502-only flow.
**What:** Holds the branch-only assemble-path wiring, including architecture-specific entry points such as the direct Chip-8 assembly path.

### `cmd/retroasm/main.go`
**Why:** Support multi-architecture assembling from the command line while keeping flag parsing and top-level CLI flow separate from the extracted helpers.
**What:**
- Added imports for Chip-8, M65816, M68000, SM83, Z80, and Z80 profile packages
- Added `--z80-profile` flag for Z80 instruction set filtering
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

### `pkg/assembler/nodes.go`
**Why:** Support Z80 register-value instruction arguments; satisfy funcorder linter.
**What:**
- Added `RegisterValueArgument` and `RegisterRegisterValueArgument` exported types for Z80-style `LD reg, value` instructions
- Reordered all type definitions to appear grouped together before their methods (funcorder compliance)

### `pkg/assembler/generate_opcode_step.go`
**Why:** Shared data/reference emission still differs from `main` because compatibility work added branch-only reference handling.
**What:** Added remaining shared opcode/data-generation plumbing used by the compatibility waves, including bank-byte reference emission for ca65-style data paths.

### `pkg/assembler/parse_ast_nodes.go`
**Why:** Handle new AST node types for register-value instructions.
**What:**
- Added `ast.RegisterValue` and `ast.RegisterRegisterValue` cases in `convertInstructionArgument()`
- Added `convertRegisterValue()` and `convertRegisterRegisterValue()` helper functions
- Minor error message cleanup

### `pkg/assembler/assembler_asm6_test.go`
**Why:** Validate assembler edge cases introduced by compatibility and addressing changes.
**What:**
- Added immediate-expression parsing coverage for asm6 syntax `(LABEL - 1)` and related generated bytes.
- Added align behavior test when current location is already aligned.
- Added forward-reference absolute-addressing regression test (`LDA forward,X`) to protect first-pass width selection.
- Added source include integration test for asm6-style `.include` flow and scoped parse context.

### `pkg/assembler/process_macros_step.go`
**Why:** Macro reparsing differs from `main` because compatibility modes and NESASM positional arguments now need branch-only handling.
**What:** Threads compatibility mode through macro reparsing and adds the positional macro substitution path used by NESASM.

---

## AST & Parser (`pkg/parser/`)

### `pkg/parser/directives/directives_test.go`
**Why:** Funcorder compliance; improved test organization.
**What:** Moved `newMockParser` constructor before exported test functions. Moved `mockParser` type after benchmarks.

---

## High-Level API (`pkg/retroasm/`)

### `pkg/arch/arch.go`
**Why:** Shared parser interfaces changed to support compatibility-mode-specific label resolution across architectures.
**What:** Added parser hooks for scoped local labels, unnamed-label resolution, and dot-local resolution.

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

### `pkg/arch/m6502/assembler/address_assigning_step.go`
**Why:** Forward-reference correctness for label addresses in 1-pass sizing.
**What:** Switched instruction definition lookup to prefer numeric `OpcodeID` for direct dispatch when available. Added handling for unresolved forward references by keeping a conservative maximum-width addressing mode during the first pass so width remains stable before opcode resolution.

### `pkg/arch/m6502/assembler/generate_opcode_step.go`
**Why:** Better 6502 opcode generation behavior for symbolic values.
**What:** Added `OpcodeID`-based instruction lookup and automatic `ZeroPage` to `Absolute` addressing upgrade when index operands do not fit in one byte. Added helper methods for upgrade path (`upgradeToAbsolute`, `upgradeAndGenerateWord`) and adjusted tests for opcode ID support.

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

## Architecture: SM83 (`pkg/arch/sm83/`) — ALL NEW

### `pkg/arch/sm83/sm83.go`
**Why:** New architecture support for Sharp SM83 (LR35902) CPU used in Game Boy.
**What:** Architecture entry point. Uses `*InstructionGroup` as generic type T (same pattern as Z80). Builds instruction groups from retrogolib SM83 definitions. 16-bit address width. No profile system (unlike Z80).

### `pkg/arch/sm83/sm83_test.go`
**Why:** Unit tests for SM83 architecture initialization and instruction groups.
**What:** Tests instruction group building, case-insensitive lookup, CB family instructions (SWAP, RLC), LDH, and unknown instruction handling.

### `pkg/arch/sm83/parser/register.go`
**Why:** SM83 register name resolution.
**What:** Maps register/pair/condition names to retrogolib SM83 constants. Handles case-insensitive lookup. Includes RST vector lookup, indirect register lookup, and condition detection.

### `pkg/arch/sm83/parser/instruction.go`
**Why:** SM83 instruction parsing and variant resolution.
**What:** Parses SM83 mnemonics and operands (registers, conditions, indirect `(HL)`, `(HL+)`, `(HL-)`, immediates, bit numbers) into AST nodes. Resolves matching instruction variant from operand patterns. Handles SM83-specific forms: LDH, LD (C),A, LD (HL+),A, BIT/SET/RES, RST vectors, condition+address pairs.

### `pkg/arch/sm83/assembler/address_assigning_step.go`
**Why:** SM83 address assignment in assembler pipeline.
**What:** Assigns addresses based on instruction opcode lengths. Resolves CB bit instruction addressing via base addressing map entry.

### `pkg/arch/sm83/assembler/generate_opcode_step.go`
**Why:** SM83 opcode generation.
**What:** Encodes SM83 instructions into byte sequences. Handles CB-prefix opcodes, bit number encoding (bits 5-3), register code encoding (bits 2-0), immediate/extended/relative operands.

### `pkg/arch/sm83/assembler/generate_opcode_step_test.go`
**Why:** Tests for SM83 opcode generation and address assignment.
**What:** Tests NOP, LD BC,nn, LD A,n, JR, JR NZ, BIT 3,A, SWAP A, RLC B, JP nn, PUSH BC, RST 38H, INC B, LD A,B, SET 7,A, RES 0,(HL). Boundary tests for address limits and relative offsets.

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

### `docs/m65816-support-plan.md`
M65816 architecture support plan (instruction set coverage, encoding strategy, testing roadmap).

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

### `docs/sm83-support-plan.md`
SM83 architecture implementation plan (instruction groups, register-based opcode lookup, CB-prefix instructions, SM83-specific instructions).

---

## Shared Compatibility & Parser Infrastructure — NEW

### Compatibility mode configuration and CLI plumbing

#### `pkg/assembler/config/compatibility.go` (new)
**Why:** Support multiple legacy assembler syntaxes (x816, asm6, ca65, NESASM) with one shared mode system.
**What:** Defines `CompatibilityMode` enum (`CompatDefault`, `CompatX816`, `CompatAsm6`, `CompatCa65`, `CompatNesasm`), `ParseCompatibilityMode()`, `String()`, and shared feature queries including `ColonOptionalLabels()`, `AnonymousLabels()`, `AsteriskProgramCounter()`, `BankByteOperator()`, `LocalLabelScoping()`, `UnnamedLabels()`, `DotLocalLabels()`, and `NesasmMacroSyntax()`.

#### `pkg/assembler/config/compatibility_test.go` (new)
**Why:** Keep compatibility mode parsing and feature switches stable as dialect support grows.
**What:** Tests parsing, string conversion, case/whitespace handling, and all feature query combinations across supported modes.

#### `pkg/assembler/config/config.go`
**Why:** Thread compatibility mode through generic assembler configuration.
**What:** Added `CompatibilityMode` field to `Config[T]`.

#### `cmd/retroasm/main.go`
**Why:** Expose compatibility mode selection in the CLI.
**What:** Added `--compat` / `-m`, `parseCompatMode()`, compat mode logging, and plumbing from CLI options into `registerArchitectureForCPU()` and downstream assembler config.

#### `pkg/assembler/assembler.go`
**Why:** Ensure normal assembly uses the selected compatibility mode.
**What:** Updated `Process()` to construct the parser with `asm.cfg.CompatibilityMode`.

#### `pkg/assembler/process_macros_step.go`
**Why:** Keep macro reparse behavior consistent with the parent file's dialect.
**What:** Updated `macroTokensToAStNodes()` to call `parser.NewWithTokens()` with `asm.cfg.CompatibilityMode`.

### Parser handler wiring and directive dispatch

#### `pkg/parser/directives/directives.go`
**Why:** Build directive behavior per compatibility mode instead of relying on one global map.
**What:**
- Added `BuildHandlers()` to overlay mode-specific handlers on top of the base handler set.
- Added `x816Handlers()`, `asm6Handlers()`, `ca65Handlers()`, and `nesasmHandlers()`.
- Added `NoOp()` and `mergeHandlers()` and deprecated the global `Handlers` map.

#### `pkg/parser/directives/noop_test.go` (new)
**Why:** Lock down mode-specific handler registration and token consumption.
**What:** Tests `NoOp()` behavior and verifies handler maps for default and compatibility modes.

#### `pkg/parser/alias.go`
**Why:** Make alias parsing use the active parser's directive set.
**What:** Switched directive lookup from global handlers to `p.handlers`.

#### `pkg/parser/directives/helper.go`
**Why:** Reuse data-token parsing from parser-level compatibility logic.
**What:** Added `ReadDataTokensExported()` wrapper around `readDataTokens()`.

### Core parser behavior needed by multiple dialects

#### `pkg/parser/parser.go`
**Why:** Centralize shared compatibility behavior in the parser instead of scattering it across architecture parsers.
**What:**
- Added `compatMode`, `handlers`, `lastNonLocalLabel`, and unnamed/anonymous label tracking to `Parser[T]`.
- Updated `New()` and `NewWithTokens()` to accept `config.CompatibilityMode`.
- Extracted `parseToken()` from `TokensToAstNodes()`.
- Added shared parsing for anonymous `+`/`-` labels, `* = value`, and colon-optional labels.
- Added shared helpers for local-label scoping, unnamed-label resolution, dot-local resolution, and `name .keyword` patterns.
- Updated `parseDot()`, `parseAlias()`, and `parseIdentifier()` to route through per-mode handlers and dialect-specific helpers.

#### `pkg/parser/parser_test.go`
**Why:** Keep the base parser expectations aligned with the new constructor and opcode-aware AST behavior.
**What:** Updated parser creation to pass `config.CompatDefault` and added `m6502Instruction()` helper that includes opcode IDs in expected AST nodes.

### Shared parser interface extensions for dialect-aware operand parsing

#### `pkg/arch/arch.go`
**Why:** Let architecture-specific parsers resolve scoped label forms consistently across compatibility modes.
**What:** Added `ScopeLocalLabel(name string) string`, `ResolveUnnamedLabel(forward bool, level int) string`, and `ResolveDotLocalLabel(name string) string` to the `Parser` interface.

#### `pkg/arch/m6502/parser/instruction.go`
**Why:** Keep 6502 operand parsing compatible with asm6, ca65, and NESASM label semantics.
**What:** Added helpers so operand resolution can scope `@` labels, resolve ca65 `:+`/`:-` unnamed labels, and resolve NESASM `.label` references before opcode matching.

### Include and source-reparse behavior

#### `pkg/assembler/parse_ast_nodes.go`
**Why:** Support source includes and keep included files parsed under the same architecture and compatibility mode.
**What:**
- Split `parseInclude()` into `parseBinaryInclude()` and `parseSourceInclude()`.
- `parseSourceInclude()` now lexes, parses, and recursively processes included source files with the current architecture and compat mode.

#### `pkg/assembler/assembler_asm6_test.go`
**Why:** Protect dialect-sensitive include and addressing behavior.
**What:** Added asm6 coverage for source includes, immediate expressions, alignment behavior, and forward-reference width selection.

---

## Dialect Features: asm6 / asm6f — NEW

### Label scoping and parser behavior

#### `pkg/parser/parser.go`
**Why:** asm6 uses `@` local labels scoped to the most recent non-local label.
**What:** Added non-local label tracking plus `ScopeLocalLabel()`/`scopeLocalLabel()` behavior and applied that scoping during label creation and operand parsing.

#### `pkg/parser/directives/data.go`
**Why:** Data expressions must resolve asm6 local labels the same way instruction operands do.
**What:** Updated `readDataTokens()` to apply `ScopeLocalLabel()` to identifier tokens.

### asm6f NES 2.0 configuration directives

#### `pkg/parser/ast/configuration.go`
**Why:** Represent new NES 2.0 header fields emitted by asm6f directives.
**What:** Added `ConfigNes2ChrRAM`, `ConfigNes2PrgRAM`, `ConfigNes2Sub`, `ConfigNes2TV`, `ConfigNes2VS`, `ConfigNes2BRam`, and `ConfigNes2ChrBRam`.

#### `pkg/parser/directives/nesasm.go`
**Why:** Parse asm6f-style `NES2*` directives into AST configuration nodes.
**What:** Added `nes2Directives` and `Nes2Config()`.

#### `pkg/parser/directives/directives.go`
**Why:** Register asm6/asm6f-specific directives.
**What:** Expanded `asm6Handlers()` with `UNSTABLE`, `HUNSTABLE`, `IGNORENL`, `ENDINL`, and the `NES2*` directive set.

### Tests
- `pkg/parser/parser_asm6_test.go` — Switched asm6 parser tests to `CompatAsm6` and added local-label scoping coverage.
- `pkg/parser/directives/noop_test.go` — Added asm6 handler checks and `TestNes2Config`.
- `pkg/arch/z80/parser/mock_parser_test.go` — Added `ScopeLocalLabel()` to the parser mock.
- `pkg/parser/directives/directives_test.go` — Added `ScopeLocalLabel()` to the parser mock.

---

## Dialect Features: ca65 — NEW

### Unnamed labels and local scope handling

#### `pkg/parser/parser.go`
**Why:** ca65 uses bare `:` unnamed labels plus `:+`/`:-` style references.
**What:**
- Added `unnamedLabelCount` and `parseUnnamedLabel()`.
- Added `token.Colon` handling in `parseToken()` for unnamed label definitions.
- Added `isUnnamedLabelRef()` and `ResolveUnnamedLabel()` so colon-prefixed operand references resolve to synthetic names.

#### `pkg/assembler/config/compatibility.go`
**Why:** Enable ca65-specific label parsing switches.
**What:** Extended `LocalLabelScoping()` to cover `CompatCa65` and added `UnnamedLabels()`.

### ca65 scope and data directives

#### `pkg/parser/ast/scope.go` (new)
**Why:** Model ca65 `.scope` / `.endscope` blocks in the AST.
**What:** Added `Scope` and `ScopeEnd` nodes with constructors and `Copy()` methods.

#### `pkg/parser/ast/data.go`
**Why:** Support ca65 bank-byte data references.
**What:** Added `BankAddressByte` to `ReferenceType`.

#### `pkg/parser/directives/ca65.go` (new)
**Why:** Implement ca65-specific directive parsing.
**What:** Added handlers for `.scope`, `.endscope`, `.asciiz`, `.faraddr`, `.bankbytes`, `.warning`, and `.out`.

#### `pkg/parser/directives/directives.go`
**Why:** Register ca65 aliases, data directives, diagnostics, and no-op stubs.
**What:** Expanded `ca65Handlers()` with scope directives, data emitters, `repeat`/`endrepeat` aliases, diagnostics (`warning`, `fatal`, `out`, `assert`), and the larger no-op directive set (`export`, `import`, `feature`, `charmap`, `autoimport`, etc.).

#### `pkg/parser/directives/macro.go`
**Why:** ca65 terminates macros with `.endmacro` as well as `.endm`.
**What:** Updated macro token reading to recognize both terminators.

### Assembler pipeline support for ca65 AST/data features

#### `pkg/assembler/nodes.go`
**Why:** Carry bank-byte reference requests through the assembler pipeline.
**What:** Added `bankAddressByte` to the internal `referenceType` enum.

#### `pkg/assembler/parse_ast_nodes.go`
**Why:** Translate ca65 scope/data AST nodes into assembler state.
**What:** Added `ast.Scope`, `ast.ScopeEnd`, and `ast.BankAddressByte` handling, including child-scope creation and scope restoration.

#### `pkg/assembler/generate_opcode_step.go`
**Why:** Emit the bank byte for `.bankbytes`-style data references.
**What:** Added `bankAddressByte` handling in `generateReferenceDataBytes()`.

### Tests
- `pkg/parser/parser_ca65_test.go` (new) — Covers unnamed labels, `@` local scoping, `.scope`, `.asciiz`, `.warning`, `.endmacro`, and ca65 no-op directives.
- `pkg/assembler/config/compatibility_test.go` — Added ca65 coverage for `LocalLabelScoping()` and `UnnamedLabels()`.
- `pkg/arch/z80/parser/mock_parser_test.go` — Added `ResolveUnnamedLabel()` to the parser mock.
- `pkg/parser/directives/directives_test.go` — Added `ResolveUnnamedLabel()` to the parser mock.

---

## Dialect Features: NESASM — NEW

### Dot-local labels and macro syntax

#### `pkg/parser/parser.go`
**Why:** NESASM uses dot-local labels plus `name .macro` definitions with positional parameters.
**What:**
- Added `parseDotLocalLabel()`, `scopeDotLocalLabel()`, and `ResolveDotLocalLabel()`.
- Updated `parseDot()` to fall back to dot-local label parsing in NESASM mode.
- Added `parseNesAsmMacro()` and detection for `name .macro`.
- Updated label-scope tracking for `DotLocalLabels()` mode.

#### `pkg/assembler/config/compatibility.go`
**Why:** Gate NESASM-only parser features.
**What:** Added `DotLocalLabels()` and `NesasmMacroSyntax()` feature checks.

#### `pkg/lexer/token/token.go`
**Why:** NESASM macro arguments use `\1` through `\9`.
**What:** Added `Backslash` token support.

#### `pkg/assembler/process_macros_step.go`
**Why:** Expand NESASM positional macro arguments correctly.
**What:** Split macro resolution into standard named handling and `resolvePositionalMacro()` for `\1`-`\9` substitution.

### NESASM directive and data support

#### `pkg/parser/directives/directives.go`
**Why:** Register NESASM-specific directives and stubs.
**What:** Added `.ds`, `.endp`, `.fail`, listing controls, and section-switching no-op handlers in `nesasmHandlers()`.

#### `pkg/parser/directives/data.go`
**Why:** Treat `.ds` as a storage directive.
**What:** Added `"ds": 1` to `dataByteWidth`.

### Tests
- `pkg/parser/parser_nesasm_test.go` (new) — Covers dot-local label scoping, `name .macro`, `.fail`, no-op directives, and default-mode rejection.
- `pkg/assembler/config/compatibility_test.go` — Added `DotLocalLabels()` and `NesasmMacroSyntax()` checks.
- `pkg/arch/z80/parser/mock_parser_test.go` — Added `ResolveDotLocalLabel()` to the parser mock.
- `pkg/parser/directives/directives_test.go` — Added `ResolveDotLocalLabel()` to the parser mock.

---

## Dialect Features: x816 — NEW

### Expression and token-system extensions

#### `pkg/lexer/token/token.go`
**Why:** x816 expressions use bitwise and shift operators not previously tokenized.
**What:** Added `ShiftLeft`, `ShiftRight`, `Ampersand`, and `BitwiseXor` tokens.

#### `pkg/lexer/token/type.go`
**Why:** Make the new tokens participate in expression parsing.
**What:** Added `Pipe`, `ShiftLeft`, `ShiftRight`, `Ampersand`, and `BitwiseXor` to the operator set.

#### `pkg/expression/operator.go`
**Why:** Evaluate the new bitwise and shift operators.
**What:** Added operator priorities plus `evaluateBitwiseIntInt()` and routed int-int evaluation through it.

#### `pkg/expression/expression.go`
**Why:** x816 accepts keyword operators such as `SHL`, `SHR`, `AND`, `OR`, and `XOR`.
**What:** Added `keywordOperators`, `resolveKeywordOperator()`, and RPN parsing support for keyword-to-token conversion.

#### `pkg/expression/keyword_operator_test.go` (new)
**Why:** Prevent regressions in x816 operator parsing.
**What:** Added coverage for keyword operators, symbol operators, and case-insensitive behavior.

### x816 directives and parser syntax

#### `pkg/parser/directives/directives.go`
**Why:** Register the broader x816 directive surface.
**What:** Expanded `x816Handlers()` with long/dword data directives, `.src`, `.comment`, `.end`, `.mem`, `.index`, listing/diagnostic directives, ROM mode directives, and settings directives.

#### `pkg/parser/directives/x816.go` (new)
**Why:** x816 `.comment` blocks consume lines until `.end`.
**What:** Added `CommentBlock()` handler.

#### `pkg/parser/directives/data.go`
**Why:** x816 adds 3-byte and 4-byte data/storage directives.
**What:** Added `dcl`, `dl`, `dcd`, `dd`, `dsl`, and `dsd` widths.

#### `pkg/parser/parser.go`
**Why:** x816 relies on `name .equ` / `name .rs` / `name .macro` patterns and colon-optional labels.
**What:** Added `isDotIdentifierKeyword()` and `parseDotIdentifier()`, refactored `parseIdentifier()`, and fixed the colon-optional label column check.

### Tests
- `pkg/parser/parser_x816_test.go` (new) — Covers no-op directives, `.comment`, `.src`, `.equ`, colon-optional labels, anonymous labels, and `.end`.
- `pkg/expression/keyword_operator_test.go` (new) — Covers keyword operators and bitwise token synonyms.
- `pkg/parser/directives/noop_test.go` — Expanded x816 handler assertions (`mem`, `index`, `opt`, `symbol`, `dcl`, `dcd`, `src`, `comment`).
