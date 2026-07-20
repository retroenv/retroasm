# Work Branch Changes

This document is the extraction plan for moving the remaining `work2` changes to
`main` as small, independently reviewable parts. It describes the live branch
delta; already-extracted historical groups are intentionally omitted.

**Work branch:** `work2` (`d4b21c9`, 2026-07-20)
**Target branch:** `main` (`af8513a`, 2026-07-20)
**Merge base:** `111b97b`
**Last reviewed:** 2026-07-20

## Current Snapshot

Use the three-dot diff while planning extractions:

```sh
git diff --name-status main...HEAD
git diff --stat main...HEAD
```

At the committed branch tips, the raw three-dot delta is **141 files, 20,365
insertions, and 242 deletions**. It still includes extracted hunks until `main`
is merged back into `work2` and advances the merge base.

Do not use `git diff main..HEAD` for the extraction inventory until `main` has
been synced into `work2`. The endpoint diff currently reports 142 files,
but it presents these newer `main` changes as reverse changes:

- `.golangci.yml`
- `Makefile`
- `pkg/arch/arch.go`
- `pkg/arch/m6502/parser/{instruction.go,instruction_test.go}`
- `pkg/assembler/{assembler.go,assembler_asm6_test.go,parse_ast_nodes.go,process_macros_step.go}`
- `pkg/assembler/config/compatibility.go`
- `pkg/parser/{alias.go,parser.go,parser_asm6_test.go,parser_test.go}`
- `pkg/parser/directives/{data.go,directives.go,directives_test.go,noop_test.go}`

These paths contain newer `main` changes and must be split by hunk rather than
copied from `work2`.

Current verification:

- P01 (`.align` correctness and regression coverage) was merged to `main` as
  `7fd62bf`.
- P02 (compatibility-mode transport and independent parser handler maps) is
  merged to `main` as `f26344c`.
- P03 (shared label-resolution hooks) is merged to `main` as `38f84d7`.
- P04 (parenthesized M6502 immediate expressions) is merged to `main` as
  `af8513a`. P05 is the next extraction part.
- `make lint` and `make test` passed for P04 on `main`.
- `make lint` and `make test` passed for P03 on `main`.
- `make lint` and `make test` passed for P02 on `main`.
- `go test ./...` passes on `work2`.
- The pass uses the local `retrogolib` replacement in `go.mod`.
- The pinned `retrogolib` version contains Chip-8, x86, and Z80 packages, but
  does not contain `arch/cpu/m65816`, `arch/cpu/m68000`, or `arch/cpu/sm83`.
  Those three architecture extractions are blocked until a released module
  version contains their dependencies.

## Corrections to the Previous Plan

- The previous count of 129 files was stale; the live three-dot count is 141.
- Chip-8 was incorrectly marked merged. Its package and examples are still
  branch-only.
- The old AST/opcode and source-include groups are already on `main`; they are
  no longer extraction groups. Only the compatibility-mode calls layered on
  those paths remain.
- CLI files and `pkg/parser/parser.go` contain several unrelated features and
  must be split by hunk. Copying any of those files wholesale would combine
  dialects and architectures.
- The current local `replace github.com/retroenv/retrogolib => ...` is a
  developer-only setting, not a mergeable dependency change.
- x86 is a library package only in this branch. The CLI still rejects `x86`, so
  README or CLI support claims must not be added with the package.
- The AST helper additions in `pkg/parser/ast/node.go` and
  `pkg/parser/ast/instruction.go` have no non-test callers and should not be
  merged unless a real consumer is added.

## Extraction Rules

Each part below should be a separate PR or merge commit.

1. Start from current `main`, not by merging the whole `work2` branch.
2. Split mixed files by hunk and take only the behavior named by the part.
3. Include focused tests in the same part as the behavior they protect.
4. Run the focused command listed for the part, then `go test ./...` and
   `git diff --check`.
5. Do not use a local module replacement in a merge candidate.
6. Add architecture or dialect README claims only when that exact public path
   is usable on `main`.

## Ordered Merge Parts

### Phase 0: Synchronize and Establish a Clean Baseline

#### P00 — Sync `main` into the extraction base

**Scope:** no feature extraction. Preserve the `main` versions of
`.golangci.yml`, `Makefile`, and `pkg/assembler/config/compatibility.go`, then
recompute the three-dot inventory.

**Validation:** `make lint`, `go test ./...`, `git diff --check`.

### Phase 2: One Compatibility Dialect Per Part

#### P05 — x816 compatibility

**Scope:** x816-only parser behavior, directives, and tests:

- x816 hunks in `pkg/parser/parser.go` and
  `pkg/parser/directives/directives.go`
- `pkg/parser/directives/x816.go`
- x816 data-width hunks in `pkg/parser/directives/data.go`
- `pkg/parser/directives/helper.go`
- x816 cases in `pkg/parser/directives/noop_test.go`
- `pkg/parser/parser_x816_test.go`
- `docs/x816-compatibility-plan.md`

This includes colon-optional and anonymous labels, `* = value`, `.equ`, source
include aliases, comment blocks, 3/4-byte data directives, and x816 no-ops.

**Prerequisite:** P00.
**Validation:** `go test ./pkg/parser/... -run X816`.

#### P06 — asm6 and asm6f compatibility

**Scope:**

- asm6 local/anonymous label and handler hunks in `pkg/parser/parser.go` and
  `pkg/parser/directives/directives.go`
- `pkg/parser/ast/configuration.go`
- NES 2.0 hunks in `pkg/parser/directives/nesasm.go`
- asm6 cases in `pkg/parser/directives/noop_test.go`
- `pkg/parser/parser_asm6_test.go`
- remaining asm6-specific tests in `pkg/assembler/assembler_asm6_test.go`
- `docs/asm6-compatibility.md`

Source include infrastructure itself is already on `main`; do not re-extract
it here.

**Prerequisite:** P00.
**Validation:**
`go test ./pkg/parser/... ./pkg/assembler/... -run 'Asm6|Nes2'`.

#### P07 — ca65 compatibility

**Scope:**

- ca65 unnamed/local-label hunks in `pkg/parser/parser.go` and
  `pkg/arch/m6502/parser/instruction.go`
- ca65 handler hunks in `pkg/parser/directives/directives.go`
- `pkg/parser/directives/ca65.go`
- `.endmacro` support in `pkg/parser/directives/macro.go`
- ca65 cases in `pkg/parser/directives/noop_test.go`
- `pkg/parser/parser_ca65_test.go`
- `docs/ca65-compatibility.md`

Scope AST nodes, bank-byte references, and their assembler pipeline support are
already on `main`.

**Prerequisite:** P00.
**Validation:** `go test ./pkg/parser/... -run Ca65`.

#### P08 — NESASM compatibility

**Scope:**

- NESASM dot-local and macro-definition hunks in `pkg/parser/parser.go`
- NESASM handlers and `ds` width hunks in
  `pkg/parser/directives/{directives,data}.go`
- positional macro expansion in `pkg/assembler/process_macros_step.go`
- `pkg/parser/parser_nesasm_test.go`
- `docs/nesasm-compatibility.md`

Backslash tokenization is already on `main` and is not part of the remaining
delta.

**Prerequisite:** P00.
**Validation:**
`go test ./pkg/parser/... ./pkg/assembler/... -run 'Nesasm|Positional'`.

#### P09 — Expose compatibility mode in the CLI

**Scope:** only `--compat`/`-m`, logging, parsing, and the 6502 registration path
in these mixed files:

- `cmd/retroasm/main.go`
- `cmd/retroasm/assemble.go`
- `cmd/retroasm/architecture.go`
- `cmd/retroasm/main_test.go`
- `docs/compatibility-mode-plan.md`

Do not add new CPU registrations or Z80 profile handling in this part.

**Prerequisites:** P05-P08.
**Validation:** `go test ./cmd/retroasm/... -run Compat`.

### Phase 3: Architectures Available in the Pinned Dependency

#### P10 — Chip-8 library support and examples

**Scope:** `pkg/arch/chip8/**` and `examples/chip8/**`.

**Validation:** `go test ./pkg/arch/chip8/...`.

#### P11 — Chip-8 CLI assembly path

**Scope:** only the Chip-8 direct-assembler path in
`cmd/retroasm/assemble.go`, plus focused CLI coverage. Do not mix compatibility
or other CPU registration changes into this part.

**Prerequisite:** P10.
**Validation:** `go test ./cmd/retroasm/... -run 'Chip8|CHIP8'`.

#### P12 — x86 library package

**Scope:** `pkg/arch/x86/**` only.

This is intentionally package-only. Before merging, add focused parser and
encoder tests or explicitly accept the current limited root-package coverage.
Do not advertise CLI support; `validateCPU` still rejects x86.

**Validation:** `go test ./pkg/arch/x86/...`.

#### P13 — Z80 profile package

**Scope:** `pkg/arch/z80/profile/**`.

**Validation:** `go test ./pkg/arch/z80/profile/...`.

#### P14 — Z80 parser and resolver

**Scope:** `pkg/arch/z80/parser/**`.

**Prerequisite:** P13.
**Validation:** `go test ./pkg/arch/z80/parser/...`.

#### P15 — Z80 opcode generation

**Scope:** `pkg/arch/z80/assembler/**`.

**Prerequisite:** P14.
**Validation:** `go test ./pkg/arch/z80/assembler/...`.

#### P16 — Z80 architecture adapter

**Scope:**

- `pkg/arch/z80/options.go`
- `pkg/arch/z80/z80.go`
- `pkg/arch/z80/z80_test.go`

**Prerequisites:** P13-P15.
**Validation:** `go test ./pkg/arch/z80/...`.

#### P17 — Z80 CLI profiles and fixture integration

**Scope:**

- Z80-only hunks in `cmd/retroasm/{architecture,assemble,main,main_test}.go`
- `cmd/retroasm/z80_fixture_test.go`
- `.gitignore` exceptions for `tests/z80`
- `tests/z80/**`
- `docs/z80-support-plan.md`
- `docs/z80-branch-changes.md`

**Prerequisite:** P16.
**Validation:**
`go test ./cmd/retroasm/... -run Z80` followed by `go test ./pkg/arch/z80/...`.

### Phase 4: Release the Missing `retrogolib` Dependencies

#### P18 — Replace the local dependency with a released module version

**Scope:** update the `retrogolib` requirement in `go.mod` to a published
version containing M65816, M68000, and SM83. Never copy the local `replace`
directive.

This part is blocked until such a module version exists.

**Validation:** with no `replace` directive, run `go mod tidy`,
`go test ./...`, and `go list -m all`.

### Phase 5: Architectures Requiring P18

#### P19 — M65816 parser

**Scope:** `pkg/arch/m65816/parser/**`.

**Prerequisite:** P18.
**Validation:** `go test ./pkg/arch/m65816/parser/...`.

#### P20 — M65816 opcode generation

**Scope:** `pkg/arch/m65816/assembler/**`.

**Prerequisite:** P19.
**Validation:** `go test ./pkg/arch/m65816/assembler/...`.

#### P21 — M65816 adapter, CLI, and feature plan

**Scope:**

- `pkg/arch/m65816/m65816.go`
- `pkg/arch/m65816/m65816_test.go`
- M65816-only CLI hunks
- `docs/m65816-support-plan.md`

**Prerequisite:** P20.
**Validation:**
`go test ./pkg/arch/m65816/... ./cmd/retroasm/... -run 'M65816|65816'`.

#### P22 — SM83 parser

**Scope:** `pkg/arch/sm83/parser/**`.

**Prerequisite:** P18.
**Validation:** `go test ./pkg/arch/sm83/parser/...`.

#### P23 — SM83 opcode generation

**Scope:** `pkg/arch/sm83/assembler/**`.

**Prerequisite:** P22.
**Validation:** `go test ./pkg/arch/sm83/assembler/...`.

#### P24 — SM83 adapter, CLI, and feature plan

**Scope:**

- `pkg/arch/sm83/sm83.go`
- `pkg/arch/sm83/sm83_test.go`
- SM83-only CLI hunks
- `docs/sm83-support-plan.md`

**Prerequisite:** P23.
**Validation:**
`go test ./pkg/arch/sm83/... ./cmd/retroasm/... -run SM83`.

#### P25 — M68000 parser

**Scope:** `pkg/arch/m68000/parser/**`.

**Prerequisite:** P18.
**Validation:** `go test ./pkg/arch/m68000/parser/...`.

#### P26 — M68000 opcode generation

**Scope:** `pkg/arch/m68000/assembler/**`.

**Prerequisite:** P25.
**Validation:** `go test ./pkg/arch/m68000/assembler/...`.

#### P27 — M68000 adapter, CLI, and feature plan

**Scope:**

- `pkg/arch/m68000/m68000.go`
- `pkg/arch/m68000/m68000_test.go`
- M68000-only CLI hunks
- `docs/m68000-support-plan.md`

**Prerequisite:** P26.
**Validation:**
`go test ./pkg/arch/m68000/... ./cmd/retroasm/... -run M68000`.

### Phase 6: Public Documentation and Final Cleanup

#### P28 — README support matrix and usage

**Scope:** `README.md`, rewritten to describe only architectures, dialects, CLI
flags, and examples already present on `main`. Do not copy the current README
wholesale before all of its claims are true.

**Prerequisites:** all feature parts represented in the README.
**Validation:** compare every documented CPU/system/flag with `retroasm -h` and
run one smoke assembly for every documented architecture.

#### P29 — Remove branch-only scaffolding

Do not merge the current `go.mod` replacement or
`docs/work-branch-changes.md`. Remove the tracking document after all applicable
parts have landed. Finish with `make lint`, `make test`, `go test ./...`, and a
clean worktree.

## Files to Drop or Rework Instead of Merge

| Path | Disposition |
|---|---|
| `go.mod` | Drop the local `replace`; create P18 with a real released version. |
| `.golangci.yml`, `Makefile`, `pkg/assembler/config/compatibility.go` | Keep `main`; current two-dot differences come from newer `main`. |
| `pkg/arch/m6502/assembler/address_assigning_step.go` | Branch delta is explanatory comments only; behavior is already on `main`. |
| `pkg/arch/m6502/assembler/generate_opcode_step.go` | Branch delta is explanatory comments only; behavior is already on `main`. |
| `pkg/parser/ast/node.go`, `pkg/parser/ast/node_test.go`, `pkg/parser/ast/instruction.go` | Drop unless a production caller is added; all new helpers are currently test-only. |
| `pkg/retroasm/default.go` | Drop; remaining changes are formatting/comment churn. |
| `docs/work-branch-changes.md` | Branch tracking only; do not merge as product documentation. |

## Live Delta Ownership

This table accounts for the 141 files in `git diff --name-only main...HEAD`.
Mixed rows require hunk-level extraction.

| Live path | Files | Owner |
|---|---:|---|
| `.gitignore` | 1 | P17 |
| `README.md` | 1 | P28 |
| `cmd/retroasm/**` | 5 | P09, P11, P17, P21, P24, P27; split by hunk |
| `docs/**` | 11 | Dialect/architecture parts; work document drops in P29 |
| `examples/chip8/**` | 3 | P10 |
| `go.mod` | 1 | Rework in P18 |
| `pkg/arch/chip8/**` | 6 | P10 |
| `pkg/arch/m6502/**` | 3 | P07; two comments-only files drop |
| `pkg/arch/m65816/**` | 7 | P19-P21 |
| `pkg/arch/m68000/**` | 20 | P25-P27 |
| `pkg/arch/sm83/**` | 7 | P22-P24 |
| `pkg/arch/x86/**` | 7 | P12 |
| `pkg/arch/z80/**` | 29 | P13-P17 |
| `pkg/assembler/**` | 4 | P08; split by hunk |
| `pkg/parser/**` | 20 | P05-P08; split by dialect, with unused AST helpers dropped |
| `pkg/retroasm/default.go` | 1 | Drop |
| `tests/z80/**` | 14 | P17 |

After every extraction, refresh the count and remove paths that no longer
differ from `main`. A part is complete only when its behavior, tests, and
feature-specific documentation are all present on `main`.
