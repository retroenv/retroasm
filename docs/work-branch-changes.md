# Gradual Merge Plan for work2

## Scope and verified base

Move the shared assembler, compatibility, and 6502 changes to `main` in small
parts. Each part must build and pass its checks with only the existing 6502
architecture. Do not merge the complete `work2` branch.

| Item | Verified value |
|---|---|
| Review date | 2026-10-06 |
| Source branch | `work2` |
| Source commit | `1a3ab37a50401077c6324e8203da1af90f12abff` |
| Local target commit | `0e4317aa7dc0fd94dfb33e93f26a7ef8e285d3b3` |
| Merge base | Same commit as local `main` |
| Working tree before this plan | Clean |
| Complete branch difference | 277 files; 40,558 added lines; 970 removed lines |

These values refer to local Git refs. The remote target was not fetched.
`main` is an ancestor of the source commit. At this base, the two-dot and
three-dot diffs describe the same changes. The old instruction to synchronize
`main` into `work2` before extraction is no longer necessary.

The counts describe the fixed source commit before this document update.
The previous plan used `f29b126`. The current source also contains reservation
conversion and bounds checks, codec reservation tests, and a CPU65816 forward
storage fix. P02a and P15/P16 include the shared 6502 changes. The CPU65816 fix
remains deferred with its architecture. The documentation index is now tracked.

This plan retains the existing 6502 scope. The complete branch inventory is in
[CHANGES_SUMMARY.md](../CHANGES_SUMMARY.md). The follow-up table below assigns
the remaining architecture file groups to separate proposals.

## Excluded work

Exclude these architecture packages and all their integration changes:

- `pkg/arch/chip8/**` and `examples/chip8/**`
- `pkg/arch/cpu65816/**`
- `pkg/arch/cpu68000/**`
- `pkg/arch/sm83/**`
- `pkg/arch/x86/**`
- `pkg/arch/z80/**`, `tests/z80/**`, and `cmd/retroasm/z80_fixture_test.go`

Also exclude their codec test files, CPU support plans, Z80 branch report,
CLI imports, registrations, profiles, system defaults, help text, and README
support claims. The `.gitignore` change enables Z80 fixtures and the documentation index.
Exclude the fixture rules. P23 can extract the documentation-index rule.

For this plan, also defer the explicit 6502 variant option and added variant
instruction modes. This includes `WithVariant`, zero-page-relative BBR/BBS,
zero-page indirect, and absolute-X indirect additions and their specific tests.
The default 6502 typed codec and fixes for existing addressing modes remain
in scope. A later variant proposal can define its own scope.

Shared contracts can remain in scope when the 6502 path uses them. A shared
file is not permission to include a new CPU feature. Defer state-transition
integration used only by excluded CPUs. Defer packed relocation fields used
only by excluded instruction encoders. Generic stream state ownership can
remain for stream joins, with tests that use a small test state type.

The remaining paths include mixed files, branch reports, and the local module
replacement. Use the per-part file lists below to select changes. A listed
file can contain changes for several parts. Its presence is not permission
to copy the complete file.

## Merge method and common checks

1. At the start of each part, inspect the current target and recompute its
   difference from the fixed source commit. If remote access is available,
   verify the target against the remote before implementation.
2. Build the merge candidate from current `main` in an isolated checkout.
   Preserve the source branch and its working files.
3. Extract the named behavior and its tests by hunk. Adapt the code to the
   target interfaces. Do not copy a mixed file in full.
4. Compare the candidate with its target base. Check that its imports, test
   helpers, docs, and command output have no excluded CPU integration.
5. Run the focused checks for the part. Then run `make build`, `make lint`,
   `make test`, and `git diff --check` once on the complete code candidate.
   `make test` already runs `go test -race ./...`.
6. Record the tested commit, dependency version, commands, and results in the
   PR. A passing test on the complete source branch does not prove that an
   extracted candidate passes.
7. Merge one approved part at a time. Wait for its target CI result. Use the
   new target as the base of the next part. Do not prepare a long chain whose
   correctness depends on unmerged changes.
8. If a check fails, stop progression and repair the candidate. Do not proceed
   to a dependent part until the required checks pass.

Most historical commits have titles such as `update` or combine several
features. Use commit patches to find the source of a change. Do not assume that
one commit is one merge part. Cherry-pick a complete commit only after its
full patch has the exact required scope and dependencies.

After an extraction, the new target commit need not be an ancestor of the
fixed source commit. A three-dot diff can then show already-extracted changes.
Use the progress record and the target code to identify completed hunks.
Compare endpoint diffs too, but retain target-only fixes that the old source
does not contain. Do not synchronize branches only to make an inventory smaller.

This task creates the plan only. It does not authorize staging, commits,
pushes, rebases, or merges. Use those operations only when requested.

For a documentation-only part, check the text, references, and
`git diff --check`. Do not run Go checks for a prose-only change.

### Dependency control

Both branch tips require
`github.com/retroenv/retrogolib v0.0.0-20260924213440-9b578e97f6b3`.
The only `go.mod` difference is the source branch's absolute local `replace`.
Do not extract that replacement. No dependency upgrade is currently part of
this plan.

Run code checks on candidates with the pinned module and no local replacement.
Also inspect `go.work` and `go env GOWORK` for workspace replacements. Use
`GOWORK=off` for candidate gates if a workspace would substitute local modules.
If a part requires an API missing from the pinned version, stop that part and
identify the exact API. Do not copy the developer path or revive the old
architecture dependency-release phase.

## Order and dependencies

The order below is the recommended merge order. Dependencies name required
code; they do not imply that other source-branch changes must be imported.

| Part | Behavior | Required parts | Review risk |
|---|---|---|---|
| P00 | Target baseline and extraction inventory | None | Low |
| P01 | Prefixed hexadecimal literals | P00 | Low |
| P02 | Memory ranges, bank fill, and segment addresses | P00 | High |
| P02a | Storage reservation conversion and bounds | P02 | High |
| P03 | Default library output length | P02 | High |
| P04 | Three-byte number support | P00 | Low |
| P05 | AST copy ownership | P00 | Medium |
| P06 | Independent data expressions and reference offsets | P04, P05 | High |
| P07 | asm6 and asm6f parser additions | P06 | Medium |
| P08 | ca65 parser and directive additions | P06 | High |
| P09 | NESASM labels and positional macros | P06 | High |
| P10 | CLI compatibility selection | P07, P08, P09 | Medium |
| P11 | Rename the existing 6502 package | P10 | High API risk |
| P12 | Scoped instruction identity | P05, P11 | High API risk |
| P13 | Typed default 6502 operands and instruction codec | P01, P12 | High |
| P14 | Owned stream and symbol metadata | P05, P06, P12 | High |
| P15 | Parser stream output and basic codec operations | P02a, P13, P14 | High |
| P16 | Data and directive stream formatting | P07-P09, P15 | High |
| P17 | Instruction relocation output | P02, P06, P15 | High |
| P18 | Instruction registration inventory | P13 | Medium |
| P19 | Exact AST equality | P05, P12, P14 | Medium |
| P20 | Stream joins and metadata reindexing | P17 | High |
| P21 | Atomic symbol rename | P16, P17 | High |
| P22 | Explicit stream rewrites and native node edits | P19-P21 | High |
| P23 | User docs and final scope audit | All included parts | Medium |

P01, P04, and P05 are independent of the output fixes. Keep the listed order
for a simple review queue. If urgent, move an independent part earlier without
changing its scope. P22 has two review steps described below.

### How to use the file lists

Paths in the tables are relative to the repository root. A brace list names
each file: `pkg/lexer/{lexer.go,lexer_test.go}` names two files. Each row states
the change to extract, not a whole-file copy operation. Keep imports and test
helpers limited to the parts already present on the candidate.

Use the fixed source SHA from the scope table for extraction. For example,
these read-only commands show the P01 patch and the current target file:

```sh
git diff 0e4317aa7dc0fd94dfb33e93f26a7ef8e285d3b3 1a3ab37a50401077c6324e8203da1af90f12abff -- pkg/lexer/lexer.go pkg/lexer/lexer_test.go
git show main:pkg/lexer/lexer.go
```

The first command shows the original source difference. It does not remove
parts already merged into a later target. Use the progress record for that.
Rows marked **Add on target** describe required candidate work that is not
provided as a complete patch in the source. Do not report those checks as
existing source coverage or as passing results.

## Phase A: Correctness and data ownership

### P00 — Establish the target baseline

| Files to inspect | Action |
|---|---|
| `go.mod` | Confirm the pinned dependency. Exclude the local replacement. |
| `Makefile` | Confirm the build, lint, and race-test commands on the target. |
| `docs/work-branch-changes.md` | Record target/source SHAs and candidate results on the source branch. Do not copy this plan to the product branch. |

Record the current target SHA, source SHA, and merge base. Confirm that the
candidate has only the existing CPU implementation and the pinned dependency.
Run the common code gates before extracting code. Record any pre-existing
failure separately.

Keep a hunk inventory for `parser.go`, `arch.go`, CPU6502 files, assembler
pipeline files, CLI files, and codec tests. Use the ownership table below.
Do not mark a part complete from a commit title or an old report.

**Exit condition:** A known target baseline and a list of changes for P01.
No synchronization merge is required at the verified base.

### P01 — Accept prefixed hexadecimal literals

| Source file | Change to extract |
|---|---|
| `pkg/lexer/lexer.go` | Retain the token prefix when reading a `0x` number after `#`. |
| `pkg/lexer/lexer_test.go` | Include the `#0x3c` token case. |

Extract the `#0x3c` lexer handling and its regression case from
`pkg/lexer/{lexer.go,lexer_test.go}`.

**Focused check:** `go test ./pkg/lexer/...`.
**Exit condition:** Prefixed and plain hexadecimal forms produce the expected
number token. Existing decimal, binary, and hexadecimal token cases still pass.

### P02 — Preserve memory-relative output and bank fill

| Source file | Change to extract |
|---|---|
| `pkg/assembler/memory.go` | Use offsets relative to memory start. Check write bounds. |
| `pkg/assembler/memory_test.go` | Include offset, fill, and invalid-write cases. |
| `pkg/assembler/address_assigning_step.go` | Start each segment at `SegmentStart`. Reject memory overflow after address assignment. Leave relocation recording for P17. |
| `pkg/assembler/write_output_step.go` | Write the memory buffer with its requested padding. |
| `pkg/assembler/banked_output_test.go` | Include bank fill, segment address, and overflow regressions. Use target `m6502` imports. |
| `pkg/assembler/assembler_asm6_test.go` | Adjust only the out-of-range branch fixture so the new memory check does not hide the branch error. |

Extract the memory layout changes as one coherent part:

- `pkg/assembler/memory.go` and `memory_test.go`
- memory and segment hunks in `address_assigning_step.go`
- output hunks in `write_output_step.go`
- `banked_output_test.go`, with imports adapted to `m6502`

Use `SegmentStart` for label and instruction addresses. Store output bytes
relative to each memory area's start. Write the full memory buffer, including
requested fill. Check the memory range during address assignment and writes.
Do not extract relocation fields from the same address-assignment file.

**Evidence to retain:** The branch tests use distinct fill bytes for three
banks, two banks at the same CPU address, an explicit segment start, and an
oversized segment rejected before opcode generation.

**Review:** Exact-end writes, empty writes, data-only overflow, non-filled gaps,
and more than one segment in a memory area. Separate bank identity from CPU
address. Keep branch-range tests meaningful: their `.org` adjustments must
still test branch distance rather than fail first on a memory limit.

**Focused check:** `go test ./pkg/assembler/... ./pkg/retroasm/...`.
**Exit condition:** Expected bank bytes and symbol addresses match; overflow
returns an error without a panic. Existing output fixtures still pass.

Source reference: `1dcc23f` and `586df3b`. Inspect their patches before use.

### P02a — Retain storage reservations and check their bounds

| Source or target file | Change to extract or add |
|---|---|
| `pkg/assembler/parse_ast_nodes.go` | Return the variable nodes from `parseVariable`. Reject `UseOffsetCounter` until its separate address model is implemented. Keep data and byte-order changes for P06/P15. |
| `pkg/assembler/address_assigning_step.go` | Make `assignVariableAddress` return an error. Reject negative sizes, unsigned overflow, unsupported address widths, and CPU address overflow. Retain P02 memory checks. Leave relocation code for P17. |
| `pkg/assembler/address_assigning_step_test.go` | Adapt `TestAssignVariableAddress` to an architecture with a valid address width. Use the target `m6502` package before P11. |
| `pkg/assembler/parse_ast_nodes_test.go` | **Add on target:** variable conversion and offset-counter rejection cases. The current source diff only changes a CPU import. |
| `pkg/assembler/address_assigning_step_test.go` | **Add on target:** negative sizes, zero sizes, exact address end, integer overflow, CPU address overflow, and RAM capacity cases. |
| `pkg/codec/reservation_test.go` | Use the 6502 cases as regression specifications for assembler tests in this part. Extract codec tests later in P15/P16. Remove all other CPU imports from candidate tests. |

This fix can reach the existing assembler before the codec exists. Test text
assembly and `ProcessAST` with two reservations in RAM. With start `$0200`
and sizes three and five, labels must have addresses `$0200`, `$0203`, and
`$0208`. A RAM-only program must emit no load bytes. A reservation between
code bytes must preserve the following byte address and the output gap.

A reservation can end exactly at the CPU address limit. A following zero-size
reservation must remain valid. A reservation beyond that limit must fail.
Offset-counter reservations must return an error, not use the code address.

The source regression tests use the codec and several new architectures.
Adapt their assertions to the existing assembler API for this phase. Do not
import the codec early only to copy those tests. The CPU65816 wider-form fix
and `pkg/arch/cpu65816/forward_storage_test.go` remain outside this phase.

**Focused check:** `go test ./pkg/assembler/...`.
**Exit condition:** Reservations reach address assignment, preserve separate
RAM addresses, emit no RAM load bytes, and reject invalid extents. Existing
segment bounds and filled output checks still pass.

### P03 — Change the default library output length separately

| Source or target file | Change to extract or add |
|---|---|
| `pkg/retroasm/default.go` | Remove `fill = yes` from `defaultConfig`. Keep the target imports and dispatcher. |
| `pkg/retroasm/assembler_test.go` | **Add on target:** explicit default, filled, and unfilled length comparisons. Its source diff changes imports and names, not output-length assertions. |
| `pkg/retroasm/example_test.go` | Check the existing example output after P02/P03. Its source diff belongs to P11. |
| `docs/library-usage.md` | **Add on target:** show an explicit filled memory config for fixed-size output. |

Extract the removal of `fill = yes` from the private `defaultConfig` in
`pkg/retroasm/default.go`, plus explicit output-length tests on the candidate.
Do not include the package rename or unrelated comment changes.

This changes public output behavior. With no supplied config, a short program
returns used bytes rather than a complete filled 32 KiB memory area. Explicit
configs must continue to control padding. State the output-size change in the
PR and add a migration example for callers that need fixed-size output.

**Focused check:** `go test ./pkg/retroasm/... ./cmd/retroasm/...`.
**Exit condition:** Tests compare default output length, explicit filled output,
and explicit unfilled output. Treat this part as a separate review decision
from the memory correctness fix.

### P04 — Support three-byte numbers

| Source file | Change to extract |
|---|---|
| `pkg/number/number.go` | Accept width three in `CheckDataWidth` and write three little-endian bytes. Leave the byte-order API for P15. |
| `pkg/number/number_test.go` | Include width-three boundary and encoding cases. Split byte-order cases into P15. |

Extract width-three range checks and little-endian output from
`pkg/number/{number.go,number_test.go}`. Preserve `WriteToBytes` behavior for
all existing widths. Keep this part small; byte-order selection can wait for
P15/P17.

**Focused check:** `go test ./pkg/number/...`.
**Exit condition:** `$ffffff` fits in three bytes; `$1000000` fails the width
check; all existing number cases pass. This supports ca65 data, with no new CPU.

### P05 — Make AST copies own mutable data

| Source file | Change to extract |
|---|---|
| `pkg/parser/ast/node.go` | Add nil-safe `copyNode` and `InlineComment`. Use the current string comment field. Omit source handles until P22. |
| `pkg/parser/ast/instruction.go` | Copy instruction and operand metadata separately. Copy modifier operator metadata, not only the modifier slice. Keep the target opcode ID type until P12. |
| `pkg/parser/ast/instruction_argument.go` | Add the opaque copy contract, validation, and `CopyNodes`. Leave form/state/reference providers for their users. |
| `pkg/parser/ast/{alias,bank,base,condition,configuration,data,enum,error,expression,function,identifier,include,label,macro,number,offset_counter,operator,register,rept,scope,segment,variable}.go` | Update each `Copy` method to own metadata and mutable fields. Keep the pre-P06 data field type. |
| `pkg/parser/ast/{node_test,instruction_argument_test}.go` | Extract copy and nil cases. Omit stream, scoped-ID, form, and handle cases until their parts. |
| `pkg/parser/ast/instruction_copy_metadata_test.go` | Retain independent comments, identifier arguments, and nil metadata tests. Remove scoped-ID and handle setup until P12/P22. |
| `pkg/parser/ast/instruction_modifier_ownership_test.go` | Extract `TestInstructionCopyKeepsModifierOperatorsIndependent`. Leave the native edit test for P22. |
| `pkg/parser/ast/node_copy_bench_test.go` | Optional copy benchmark. Retain its shared benchmark variable if later benchmark files use it. |

`comment.go` adds source handles in the source diff. That change belongs to
P22, not this copy phase. The metadata allocation layout is private. Preserve
its ownership behavior without requiring an allocation improvement claim.

Extract `copyNode`, `CopyNodes`, inline comment access, and all dependent node
`Copy` changes under `pkg/parser/ast`. Include nil-safe configuration copying
and the opaque `InstructionArgumentCopier` contract.

Do not introduce entry handles yet. Extract the copy logic without the handle
field and handle propagation. Keep the AST data representation unchanged until
P06. Defer instruction form, reference, and state-transition providers until a
part uses them.

**Review:** Copies must own comments, modifiers, expressions, nested AST lists,
and mutable typed arguments. The new mutable-argument contract can panic at
construction. Explain this API requirement and test its failure behavior.

**Focused check:** `go test ./pkg/parser/ast/...`.
**Exit condition:** Mutation of a copy cannot change its input. Scalar values,
nil values, and typed nils retain their documented behavior.

### P06 — Represent and evaluate each data item separately

| Source file | Change to extract |
|---|---|
| `pkg/parser/ast/data.go` | Change `Values` to an expression slice and copy each expression. Leave codec validation for P15. |
| `pkg/parser/ast/node_test.go` | Migrate data construction and copy assertions to the new field type. |
| `pkg/parser/ast/{symbol_reference.go,symbol_reference_test.go}` | Parse a symbol and its signed addend. Include invalid-reference cases. |
| `pkg/parser/directives/{addr,base,data,hex}.go` | Read one expression per data item. Retain storage/fill and address-byte behavior. Leave NESASM `ds` for P09. |
| `pkg/parser/directives/directives_test.go` | Include independent data expressions and update data assertions. |
| `pkg/assembler/nodes.go` | Store independent values and reference offsets. Leave opcode IDs and source-entry fields for P12/P17. |
| `pkg/assembler/parse_ast_nodes.go` | Reserve each directive's full size and collect its values. Keep the target byte-order behavior. |
| `pkg/assembler/expression_evaluation_step.go` | Evaluate each value and apply reference offsets with overflow checks. |
| `pkg/assembler/generate_opcode_step.go` | Emit values once, with their declared width. Leave instruction relocation callbacks for P17. |
| `pkg/assembler/{assembler_asm6_test,assembler_x816_test,parse_ast_nodes_test}.go` | Include mixed arithmetic and forward-data tests. Update direct data construction without the package rename. |
| `docs/library-usage.md` | **Add on target:** migrate direct `ast.Data.Values` construction to an expression slice. |

Extract the `ast.Data.Values` migration from one expression to an expression
slice. Change all users in the same part:

- `pkg/parser/ast/data.go` and related AST tests
- `pkg/parser/directives/{addr,base,data,hex}.go` and related tests
- data hunks in assembler `nodes.go`, `parse_ast_nodes.go`,
  `expression_evaluation_step.go`, and `generate_opcode_step.go`
- `symbol_reference.go` and its tests
- reference offsets and declared-width encoding in the assembler
- mixed-expression cases in asm6 and x816 assembler tests

Do not split the public field type change from its parser and assembler users.
Do not copy byte-order and relocation integration from these files yet.
Use the existing little-endian number writer, including P04's width-three path.

**Review:** Independent arithmetic items, parenthesized values, strings, fills,
low/high/bank-byte lists, symbol addends, negative results, and offset overflow.
If one item has a forward reference, reserve the complete directive's size
and emit each item exactly once after addresses exist. Preserve the current
x816 `.dl` meaning and immediate-expression behavior.

**Focused check:** `go test ./pkg/parser/... ./pkg/assembler/...`.
**Exit condition:** Cases such as `DB 1+2, 3*4, (5+1)` and
`.dw 2, later+1` produce exact expected bytes and label addresses. Include a
migration example for direct `ast.Data.Values` users.

## Phase B: Compatibility without new CPU registration

### P07 — Extract asm6 and asm6f additions

| Source file | Change to extract |
|---|---|
| `pkg/parser/directives/directives.go` | Add the asm6 overlay and its selection in `BuildHandlers`. Keep the existing x816 map location. |
| `pkg/parser/directives/nesasm.go` | Add `Nes2Config` and the NES 2.0 directive-to-item mapping. The filename does not make these P09 changes. |
| `pkg/parser/ast/configuration.go` | Add the NES 2.0 configuration item constants. |
| `pkg/parser/parser.go` | Enable asm6 local-label scoping and global-label scope updates. Leave ca65/NESASM branches for P08/P09. |
| `pkg/parser/parser_asm6_test.go` | Include local-label scope and default-mode rejection tests. |
| `pkg/parser/directives/noop_test.go` | Extract asm6/asm6f no-op and NES 2.0 parser cases. |
| `docs/asm6-compatibility.md` | Publish the retained syntax and output limits. Omit claims not verified on the candidate. |

Extract asm6 handler additions, NES 2.0 AST configuration items and parser
handlers, local-label scope updates in `parser.go`, focused parser tests, and
`docs/asm6-compatibility.md`.

Preserve the existing anonymous-label counters and tests. Much of that code is
already on `main`; do not import rewritten comments as a new feature. Keep
handler maps independent for every parser instance. Do not add the deprecated
mutable global `directives.Handlers` unless a verified caller requires it.

**Focused check:** `go test ./pkg/parser/... ./pkg/assembler/... -run 'Asm6|Nes2'`.
**Exit condition:** Mode-specific syntax and default-mode rejection are tested.
Document NES 2.0 directives as parsed configuration where applicable. Their
parser nodes alone do not prove that the assembler writes a NES 2.0 header.

### P08 — Extract ca65 additions

| Source file | Change to extract |
|---|---|
| `pkg/parser/parser.go` | Add ca65 colon definitions, unnamed references, and local-label scope behavior. |
| `pkg/parser/directives/ca65.go` | Add scope, string, far-address, bank-byte, and diagnostic handlers with their actual semantics. |
| `pkg/parser/directives/directives.go` | Add and select the ca65 overlay. |
| `pkg/parser/directives/macro.go` | Accept `.endmacro` as a macro terminator. |
| `pkg/parser/parser_ca65_test.go` | Include unnamed/local labels, scopes, strings, macro termination, and directive tests. Adapt imports to the current target. |
| `pkg/parser/directives/noop_test.go` | Extract only ca65 handler/no-op cases. |
| `pkg/assembler/assembler_ca65_test.go` | Include `TestAssemblerCa65FarAddressUsesDeclaredWidth`. |
| `docs/ca65-compatibility.md` | State the limits for warnings, import/export, assertions, and linker behavior. |

Extract the colon-label definition logic, local-label scope updates, and
unnamed-label reference handling. Add ca65 handlers from
`pkg/parser/directives/ca65.go`, their handler registrations, `.endmacro`
termination in `macro.go`, and parser tests. Include `.faraddr` assembly
coverage using P04/P06's declared-width encoding.

**Review:** `.scope` and `.endscope` connect to existing scope processing.
`.asciiz`, `.faraddr`, and bank-byte aliases need byte-output tests. The branch
maps `.warning` to `ast.Error`; do not describe it as a non-fatal warning without
an end-to-end result. Import/export and `.assert` no-ops are limited syntax
acceptance, not linker support or assertion evaluation.

**Focused check:** `go test ./pkg/parser/... ./pkg/assembler/... -run 'Ca65|Scope'`.
**Exit condition:** Labels and directives work through assembly; unsupported
semantics are stated in `docs/ca65-compatibility.md`.

### P09 — Extract NESASM labels and positional macros

| Source file | Change to extract |
|---|---|
| `pkg/parser/parser.go` | Add dot-local definitions, scope updates, and `name .macro` parsing in NESASM mode. |
| `pkg/parser/directives/directives.go` | Add and select the NESASM overlay. |
| `pkg/parser/directives/data.go` | Add the one-byte `ds` width. |
| `pkg/assembler/process_macros_step.go` | Expand positional parameters while retaining named-argument checks. |
| `pkg/parser/parser_nesasm_test.go` | Include label, macro-definition, no-op, fail, and default-mode rejection cases. |
| `docs/nesasm-compatibility.md` | Publish macro and section-directive limits. |

**Add on target:** Assembly tests for positional substitution and named macro
regressions. The source `parser_nesasm_test.go` checks macro definitions; it
does not prove that expanded macros emit the correct bytes.

Extract dot-local label definitions, scope updates, `name .macro`, NESASM
handler overlays, `ds` width, and `process_macros_step.go` positional expansion.
Include parser tests and assembly tests for actual macro expansion.

**Review:** The source branch selects positional expansion for every macro
with no named arguments. Verify zero-argument macros in other modes, missing
positional arguments, repeated parameters, and macro termination. Preserve
named macro count errors. Verify two global scopes can reuse a dot-local name.

**Focused check:** `go test ./pkg/parser/... ./pkg/assembler/... -run 'Nesasm|Macro|Positional'`.
**Exit condition:** Positional macros emit exact bytes, named macros retain
their behavior, and dot-local syntax is rejected in default mode. Include
`docs/nesasm-compatibility.md` with tested limitations.

### P10 — Expose compatibility selection in the CLI

| Source file | Change to extract |
|---|---|
| `cmd/retroasm/main.go` | Add the `compat` field, both flags, and the typed compatibility log field. |
| `cmd/retroasm/assemble.go` | Add `parseCompatMode` and pass its result to architecture registration. Exclude `assembleChip8File`. |
| `cmd/retroasm/architecture.go` | Pass the mode into the existing 6502 config. Exclude new CPU imports, profiles, and system rules. |
| `cmd/retroasm/main_test.go` | Extract compatibility flag, parsing, logging, and propagation cases with only target CPU fixtures. |
| `docs/compatibility-mode-plan.md` | Extract checked compatibility commands and guide links. Keep branch implementation history out of target user docs. |

Extract `compat` options, `-compat` and `-m`, `parseCompatMode`, typed log fields,
and propagation into the existing 6502 configuration. Adapt
`registerArchitectureForCPU` to the target's signature.

Split `cmd/retroasm/{main,assemble,architecture,main_test}.go` by hunk. Leave
CPU/system validation, CPU registrations, Z80 profiles, and the Chip-8 direct
assembly path at their target versions. Include the compatibility guide after
its command examples are checked.

**Focused check:** `go test ./cmd/retroasm/... ./pkg/retroasm/...`.
**Exit condition:** Both flag spellings select the mode; absent mode keeps the
default; invalid values fail. Smoke assembly with an explicit config must prove
that the mode reaches the parser. Existing CLI architecture errors still pass.

## Phase C: Existing 6502 APIs and owned streams

### P11 — Rename m6502 to cpu6502 as an API migration

| Source or target file | Change to extract or add |
|---|---|
| `pkg/arch/m6502/m6502.go` → `pkg/arch/cpu6502/cpu6502.go` | Move the target adapter and change package names. Do not copy the source adapter's added methods yet. |
| `pkg/arch/m6502/assembler/{address_assigning_step,generate_opcode_step,generate_opcode_step_test,instruction_size}.go` → matching `pkg/arch/cpu6502/assembler/` paths | Move the target files with no behavior change. Retain the target test cases. |
| `pkg/arch/m6502/parser/{addressing,instruction,instruction_test}.go` → matching `pkg/arch/cpu6502/parser/` paths | Move the target parser. Leave source correctness and typed-operand changes for P13. |
| `cmd/retroasm/architecture.go`, `examples/ast-first/main.go` | Update imports and constructor names for the existing CPU. |
| `pkg/retroasm/{default,assembler_test,example_test,doc}.go` | Update implementation imports, test constructors, and API examples. |
| `pkg/assembler/{assembler_asm6_test,assembler_ca65_test,assembler_x816_test,banked_output_test,address_assigning_step_test}.go` | Update the tests already present after P02-P09, including P02a. |
| `pkg/parser/{parser_asm6_test,parser_ca65_test,parser_nesasm_test,parser_test,parser_x816_test}.go` | Update parser test imports and constructors. |
| `docs/library-usage.md`, `README.md` | Update retained 6502 examples where needed. Keep the target support matrix. |
| Old `pkg/arch/m6502/` public paths | **Add on target:** forwarding packages if required by the migration decision. These are not provided by the source branch. |

Search the candidate for all old imports after these edits. Parts P02-P10 can
add callers that did not exist on the original target. The source's new
`options.go` and variant-only `cpu6502_test.go` are not rename files.

Move only the existing package implementation from `pkg/arch/m6502` to
`pkg/arch/cpu6502`. Update all repository imports, examples, and library docs.
Preserve behavior and all tests; do not copy the complete branch CPU6502 package.
That package also contains codec, variant, identity, and relocation changes.

The package path is public. Recommend a temporary compatibility package at
`pkg/arch/m6502` that forwards the existing API. If consumers use its exported
parser or assembler subpackages, provide those forwarding paths too. Record
when old paths can be removed. Check known consumers before choosing a direct
breaking rename.

**Focused check:** `go test ./pkg/arch/... ./pkg/retroasm/... ./examples/... ./cmd/retroasm/...`.
**Exit condition:** Old supported imports and new imports compile during the
migration; machine output remains unchanged. Search all code and docs for the
old path and classify each remaining reference.

Source reference: `7826748`. This commit is a starting point, not an automatic
whole-commit extraction.

### P12 — Replace the global opcode lookup with scoped identity

| Source file | Change to extract |
|---|---|
| `pkg/parser/ast/instruction.go` | Add `OpcodeID`, its constructor/validation, and identity assignment helpers. Remove the global callback. |
| `pkg/arch/arch.go` | Change the instruction ID contract and add adapter `OpcodeID`. Keep the target `ParseIdentifier` signature. |
| `pkg/parser/parser.go` | Assign IDs from the active architecture after instruction parsing. |
| `pkg/assembler/nodes.go` | Store and return the scoped ID. |
| `pkg/arch/cpu6502/cpu6502.go` | Implement architecture-local ID lookup. |
| `pkg/arch/cpu6502/assembler/{address_assigning_step,generate_opcode_step}.go` | Read the scoped ID and retain legacy mnemonic lookup for unset IDs. |
| `pkg/arch/cpu6502/assembler/generate_opcode_step_test.go` | Migrate the mock instruction ID and retain default-6502 tests. |
| `pkg/parser/opcode_identity_test.go` | Adapt parser-isolation and foreign-ID cases to 6502 plus a small test adapter. Exclude new CPU imports. |
| `pkg/parser/ast/{node_test,instruction_copy_metadata_test}.go` | Add scoped-ID assignment and copy assertions. Leave handles for P22. |
| `docs/library-usage.md` | **Add on target:** explain scoped IDs, custom adapter changes, and legacy unset IDs. |

Extract `ast.OpcodeID{Architecture, Value}`, its helpers, architecture-local
lookup, and parser identity assignment. Update `arch.Instruction`, assembler
instruction storage, mocks, and the default 6502 adapter together.

`Architecture.OpcodeID(T)` changes the public adapter interface. Explain the
migration and retain an adapter path where practical. Defer the extra
`ParseIdentifier` mnemonic parameter: the branch's 6502 adapter ignores it.
Do not add other architectures to demonstrate isolation. Use a small test
architecture or explicit foreign identity value.

**Focused check:** `go test ./pkg/parser/... ./pkg/assembler/... ./pkg/arch/cpu6502/... ./pkg/retroasm/...`.
**Exit condition:** Parser-created identities are deterministic. A global
callback cannot change another parser's identity. Foreign IDs fail validation.
Direct legacy AST construction with an unset ID still assembles through the
existing library API. Preserve mnemonic lookup for that path.

### P13 — Add the typed default 6502 instruction contract

| Source file | Change to extract |
|---|---|
| `pkg/arch/cpu6502/parser/operand.go` | Add owned typed operands for existing addressing modes. Remove variant-only operand kinds. |
| `pkg/arch/cpu6502/parser/resolved.go` | Add the owned typed-to-native projection and default-mode validation. |
| `pkg/arch/cpu6502/parser/codec.go` | Add typed construction, validation, and instruction formatting. Omit variant-only branches. |
| `pkg/arch/cpu6502/parser/{instruction,addressing}.go` | Retain typed operands and modifiers; fix numeric relative targets and filtered addressing selection. |
| `pkg/arch/cpu6502/cpu6502.go` | Expose the default builder, validator, and formatter. Leave byte order and registrations for P15/P18. |
| `pkg/arch/cpu6502/assembler/generate_opcode_step.go` | Retain the one-byte BRK fix. Leave relocation recording for P17. |
| `pkg/arch/cpu6502/assembler/generate_opcode_step_test.go` | Include BRK and default addressing regressions. Exclude added variant modes. |
| `pkg/arch/cpu6502/parser/resolved_test.go` | Include the isolated expression projection regression. |
| `pkg/parser/ast/{value_format.go,value_format_test.go}` | Add number, symbol, and expression formatting used by the typed codec. |
| `pkg/parser/ast/{node.go,node_test.go}` | Extract only lookup helpers called by the retained implementation and their tests. |

The large round-trip suite in `pkg/codec/cpu6502_test.go` requires P15.
Add focused builder/validator checks to the candidate if that suite contains
the only source coverage for a P13 behavior.

Extract default 6502 operand types, isolated resolved projection, typed builder,
validator, and instruction formatter from CPU6502 `parser/{operand,resolved,codec}.go`
and adapter hunks. Add `ast/value_format.go` and its tests. Include node lookup
helpers only when these paths use them.

Keep existing implied, accumulator, immediate, relative, zero-page, absolute,
and indirect addressing modes. Extract parser correctness fixes for numeric
branch targets, retained branch modifiers, and selection from the filtered
addressing list. Preserve the one-byte BRK encoding. Do not add variants or
new variant addressing modes as part of this contract.

**Focused check:** `go test ./pkg/arch/cpu6502/... ./pkg/parser/ast/...`.
**Exit condition:** Build, validate, format, and parse tests retain operand shape
and expected bytes. Test invalid widths, invalid addressing, explicit address
sizes, symbol modifiers, forward labels, and projection copy ownership.
Codec integration tests follow in P15.

### P14 — Add owned stream entries and symbol metadata

| Source file | Change to extract |
|---|---|
| `pkg/parser/ast/stream.go` | Add entries, positions, boundaries, owned access, metadata types, state copies, and validation. Leave joins, range replacement, removed entries, and native edit revisions for P20/P22. |
| `pkg/parser/ast/stream_symbols.go` | Rebuild symbols and segment assignments from retained nodes. |
| `pkg/parser/ast/instruction_argument.go` | Add `InstructionReference` and its provider for retained relocation validation. |
| `pkg/parser/ast/node.go` | Add symbol/node lookup helpers required by stream validation. |
| `pkg/parser/ast/stream_test.go` | Include ownership, generic state, symbol, and metadata validation cases. Omit packed fields and range replacement until their applicable parts. |
| `pkg/parser/ast/node_test.go` | Include tests for helpers introduced in this part. |

Keep test helper definitions used by these cases. Do not copy later mutation
tests merely because they share this test file.

Extract the shared stream model, source positions, boundaries, annotations,
symbol expressions, symbol rebuilding, owned getters, and validation from
`ast/stream.go`, `stream_symbols.go`, and their tests. Add instruction reference
providers only where shared validation uses them.

Leave joins, symbol rename, explicit rewrites, native edit handles, and their
revision changes for later parts. Keep relocation metadata types required by
stream validation, but defer encoder integration. Exclude packed instruction
fields and target-specific state-transition behavior.

**Focused check:** `go test ./pkg/parser/ast/...`.
**Exit condition:** Nodes, annotations, expressions, symbols, and generic state
snapshots are independently copied. Invalid indices and metadata fail without
changing the stream. Check reusable aliases and scoped definitions; do not
claim linker-ready symbol resolution from flat metadata alone.

### P15 — Connect parser streams and basic codec operations

| Source file | Change to extract |
|---|---|
| `pkg/parser/parser.go` | Add source positions, `TokensToStream`, and symbol rebuilding while retaining the node API. |
| `pkg/parser/parser_test.go` | Extract stream position and symbol-definition cases. Replace the stateful CPU fixture with a default 6502 case where needed. |
| `pkg/codec/codec.go` | Add construction, parse, identity, build, instruction format, validation, and direct assembly operations. Omit directive formatting, stateful CPU APIs, and instruction relocation reconciliation. |
| `pkg/codec/doc.go` | Describe only the API included in the candidate. |
| `pkg/codec/metadata.go` | Extract symbol, data-relocation, and segment metadata used by basic stream operations. Omit later instruction reconciliation dependencies. |
| `pkg/codec/codec_test.go` | Include constructor errors, parse/build/assemble, source diagnostics, and cancellation cases with 6502 fixtures. Split later formatting and instruction relocation assertions. |
| `pkg/codec/cpu6502_test.go` | Include default addressing, relative branches, modifiers, expressions, invalid operands, and instruction format options. Omit variant and stale-relocation cases. |
| `pkg/codec/reservation_test.go` | Extract direct assembly extent, exact-end, offset-counter rejection, and 6502 architecture-bound cases. Add `newReservationCodec` and required helpers with only 6502 imports. Leave `FormatStream` cases for P16. |
| `pkg/parser/ast/{data.go,node_test.go}` | Add `Data.Validate` and its cases required by codec validation. |
| `pkg/arch/arch.go`, `pkg/arch/cpu6502/cpu6502.go` | Add `ByteOrderer` and the default 6502 report. |
| `pkg/arch/byte_order_test.go` | Keep the 6502 case. Replace other CPU fixtures with a test adapter only if needed. |
| `pkg/number/{number.go,number_test.go}` | Add `WriteToBytesWithOrder`, its wrapper, and byte-order tests if used by the retained pipeline. |
| `pkg/assembler/{assembler,parse_ast_nodes,expression_evaluation_step,generate_opcode_step}.go` | Pass byte order through data encoding if required. Preserve a legacy fallback or document the mandatory adapter migration. |
| `docs/library-usage.md` | **Add on target:** basic codec usage and any byte-order adapter migration. |

Extract `TokensToStream(sourceName)` and retain `TokensToAstNodes` as the native
node API. Add the basic `pkg/codec` constructor, parse, single-instruction,
identity, build, validate, and direct assembly operations with default 6502
coverage. Do not route assembly through formatted source.

Add the default 6502 byte-order report and shared byte-order contract needed
for metadata. If `WriteToBytesWithOrder` is included, preserve `WriteToBytes` as
a little-endian wrapper and test both orders with numbers or a test adapter.
Do not import a new CPU to test the shared contract. The branch assembler
rejects adapters without `ByteOrderer`; preserve a documented legacy
little-endian fallback or include an explicit adapter migration before that
requirement reaches the existing public assembly API.

Defer architecture-specific parser state methods that have no 6502 caller.
Split `codec.go`, `metadata.go`, and tests to omit later formatting and
instruction-relocation calls until those parts land.

**Focused check:** `go test ./pkg/codec/... ./pkg/parser/... ./pkg/assembler/... ./pkg/retroasm/...`.
**Exit condition:** Parsed positions reach errors; direct AST assembly and text
assembly agree; cancellation and nil inputs return expected errors. Test both
legacy and stream APIs with macros, conditionals, includes, and reusable aliases.

### P16 — Format data and directive streams

| Source file | Change to extract |
|---|---|
| `pkg/codec/codec.go` | Add `FormatStream`, data/address directive selection, layout/configuration formatting, and supported-node dispatch. |
| `pkg/codec/directive_format.go` | Add structural, conditional, scope, macro, include, and diagnostic formatting. |
| `pkg/codec/codec_test.go` | Include `FormatStream` round-trip, byte-equivalence, and rejection cases. Keep only 6502 constructors and applicable mode cases. |
| `pkg/codec/doc.go`, `docs/library-usage.md` | Describe supported formatting and errors. Explain that original source spacing is not retained. |
| `pkg/codec/reservation_test.go` | Add the typed/text reservation comparison, text RAM capacity, and code-gap cases that use `ParseStream` or `FormatStream`. Keep only 6502 fixtures. |

Extract data formatting and `pkg/codec/directive_format.go`. Add symbols,
layout, conditionals, scopes, repeats, macros, includes, configuration, and
comment cases from `codec_test.go`. Keep formatters mode-aware.

**Review:** Deterministic formatting is not byte-for-byte source reproduction.
Whitespace and case can change. Preserve the represented node fields,
expressions, comments, and operand sizes. For unsupported or unrepresentable
nodes, return an error rather than drop information. Check typed nils and the
pointer/value node forms supported by the public API.

**Focused check:** `go test ./pkg/codec/... ./pkg/parser/...`.
**Exit condition:** Parse-format-parse retains supported structure;
assembly before and after formatting produces the same bytes and symbols.
Include default, asm6, ca65, NESASM, and x816 cases with their actual limitations.

### P17 — Record selected 6502 instruction relocations

| Source file | Change to extract |
|---|---|
| `pkg/arch/arch.go` | Add `RelocationEncoding`, the optional recorder, and its dispatch helper without packed fields. |
| `pkg/arch/relocation_test.go` | Include recorder and unsupported-recorder behavior. Remove packed-field expectations if present. |
| `pkg/assembler/nodes.go` | Add source-entry index and presence fields to instructions. |
| `pkg/assembler/assembler.go` | Reset, collect, and return owned relocations. Assign source indices only to direct source instructions. |
| `pkg/assembler/address_assigning_step.go` | Add the recorder and reference/addend normalization. Keep P02 bounds behavior. |
| `pkg/assembler/address_assigning_step_test.go` | Extract `TestAddressAssign_RecordInstructionRelocation` without packed-field setup or assertions. |
| `pkg/assembler/generate_opcode_step.go` | Connect the recorder to instruction generation. |
| `pkg/arch/cpu6502/assembler/generate_opcode_step.go` | Report selected displacement/address fields from default 6502 encoders. Exclude variant encoders. |
| `pkg/codec/codec.go` | Reconcile selected instruction relocations after assembly. Reject incompatible supplied records. |
| `pkg/codec/codec_test.go` | Include the 6502 instruction relocation case and applicable metadata assertions. |
| `pkg/codec/cpu6502_test.go` | Include `TestCPU6502Codec_RejectsStaleSelectedWidthRelocation` and relocation assertions in default-mode cases. |
| `pkg/parser/ast/stream_test.go` | Retain instruction reference/addend validation cases required by the recorder. |

Defer `pkg/codec/metadata_completion_test.go` to P20 because its regression
uses entry insertion. Do not add range replacement early only for this test.

Extract `RelocationEncoding`, the optional recorder, source-entry indices in
assembler instructions, reference/addend normalization, owned relocation
output, 6502 encoder callbacks, and codec reconciliation. Include
`pkg/arch/relocation_test.go` and the matching assembler and codec cases.

Record the width and offset chosen by the encoder. A relative branch relocation
must describe its encoded displacement field. An absolute or zero-page operand
must use the selected width. Existing direct nodes without source-entry
metadata must continue to assemble.

**Focused check:** `go test ./pkg/arch/cpu6502/... ./pkg/arch ./pkg/assembler/... ./pkg/codec/...`.
**Exit condition:** Bytes, symbols, and relocation records agree for labels,
addends, relative branches, and address-size selection. Reject stale metadata
that differs from the actual encoding. Test that expanded macro/include nodes
are not incorrectly assigned an original entry index. Document any missing
relocation coverage for generated nodes.

Source references: `eb5df59` and `58d2939`. Exclude the later new-CPU encoders.

### P18 — Expose the 6502 instruction registration inventory

| Source file | Change to extract |
|---|---|
| `pkg/arch/arch.go` | Add `InstructionRegistration` and its provider. Retain only fields required by the included contract. |
| `pkg/arch/cpu6502/cpu6502.go` | Return sorted default 6502 registrations with copied addressing selectors. |
| `pkg/arch/instruction_registration_test.go` | Extract the 6502 case and sorting/identity checks. Exclude 65816 fixtures. |

Extract `InstructionRegistration`, its provider, sorted default 6502 output,
and the 6502 case from `instruction_registration_test.go`.

**Focused check:** `go test ./pkg/arch/...`.
**Exit condition:** Names and addressing selectors are unique, stable, sorted,
and use valid 6502 IDs. Do not include the 65816 combined-addressing test or
imports of excluded packages. Add form providers only if an included caller
requires them.

## Phase D: Safe stream transformations

### P19 — Add exact AST equality

| Source file | Change to extract |
|---|---|
| `pkg/parser/ast/equal.go` | Add exact comparisons for supported native nodes, including modifier metadata. Adapt to the current P05 comment representation. |
| `pkg/parser/ast/equal_reflect.go` | Add composite/extension comparisons and cycle tracking. Omit handle-specific exceptions until P22. |
| `pkg/parser/ast/equal_test.go` | Include field inventory, reflection parity, nil/type distinctions, cycles, and allocation assertions. Split handle tests into P22. |
| `pkg/parser/ast/equal_bench_test.go` | Optional equality benchmark. No measured performance result is required. |

Extract `ast/equal.go`, `equal_reflect.go`, their tests, and the optional
benchmark. Keep dynamic types, comments, instruction identities, and nil/empty
slice distinctions. Preserve reflection fallback for extension nodes and
cycle handling.

**Focused check:** `go test ./pkg/parser/ast/... -run Equal`.
**Exit condition:** Field-inventory and reflection-parity tests pass. Leaf
allocation assertions remain meaningful. Source handles are introduced only
in P22; add handle-specific equality tests there. No performance claim is
required to merge this behavior.

### P20 — Join streams without losing metadata

| Source file | Change to extract |
|---|---|
| `pkg/parser/ast/stream_join.go` | Add `AppendStream`, generic state compatibility, and metadata transfer. |
| `pkg/parser/ast/stream.go` | Add range `Replace`, metadata reindexing, and index mapping used by joins and insertion. |
| `pkg/parser/ast/stream_join_test.go` | Include joins, empty/self cases, state copies, and atomic failure tests with generic state fixtures. |
| `pkg/parser/ast/stream_test.go` | Include `TestStream_ReplaceReindexesCompatibleMetadataAtomically`. |
| `pkg/codec/metadata_completion_test.go` | Include metadata completion after insertion, now that both P17 and range replacement exist. |

Extract `AppendStream`, associated insertion/reindexing logic, and tests.
Use generic test state for join compatibility. Retain positions, annotations,
comments, symbols, relocations, and segment changes at their new indices.

**Focused check:** `go test ./pkg/parser/ast/... ./pkg/codec/... -run 'Stream.*(Append|Replace|Join)|CompletesMetadataAfterEntryInsertion'`.
**Exit condition:** Empty streams, self-append, incompatible state, and invalid
metadata have tested behavior. Rejected joins leave the destination unchanged.
Returned metadata cannot mutate either input.

Source reference: `b9b1294`.

### P21 — Rename symbols atomically

| Source file | Change to extract |
|---|---|
| `pkg/parser/ast/symbol_rewrite.go` | Rewrite definitions, expressions, native operands, and supported typed references. |
| `pkg/parser/ast/stream_rename.go` | Validate the complete rename and publish nodes/metadata together. Leave native edit revision fields until P22. |
| `pkg/parser/ast/stream_rename_test.go` | Include swaps, expressions, collisions, capture, unsupported operands, and atomic failure cases. |
| `pkg/codec/stream_rename_test.go` | Extract only the 6502 integration case. Adapt the shared fixture helper described in P22. |

Extract `symbol_rewrite.go`, `stream_rename.go`, and their tests. Include the
6502 codec rename test. Update definitions, operand references, expressions,
and relocation expressions in the same operation.

**Focused check:** `go test ./pkg/parser/ast/... ./pkg/codec/... -run Rename`.
**Exit condition:** Swaps are simultaneous; collisions, symbol capture,
unsupported opaque operands, and unexpanded scope/macro nodes fail atomically.
Retain metadata and produce unchanged assembly where renaming has no address
effect. Test pointer/value support explicitly or document rejected forms.

Source reference: `dce3ccc`.

### P22 — Rewrite entries, then support native node edits

| Step | Source file | Change to extract |
|---|---|---|
| P22a | `pkg/parser/ast/stream_rewrite.go` | Add `EntryEdit`, rewrite validation/publication, metadata remapping, and removed-source records. Adapt handle-dependent code until P22b. |
| P22a | `pkg/parser/ast/stream.go` | Add storage/copy support for removed entries. |
| P22a | `pkg/parser/ast/stream_rewrite_test.go` | Include moves, copies, inserts, comments, replacement ownership, and atomic rejection. |
| P22a | `pkg/parser/ast/stream_rewrite_retained_test.go` | Extract duplicate-node ownership and many-copy relocation cases that do not need native views. |
| P22a | `pkg/codec/stream_rewrite_test.go` | Extract the 6502 explicit rewrite case. Leave `NodeEdit` cases for P22b. |
| P22b | `pkg/parser/ast/node_edit.go` | Add source handles, snapshots, owned `At`/`Nodes` reads, `Len`, `Commit`, and half-open range `Replace`. Validate the complete result before publication. |
| P22b | `pkg/parser/ast/{node,comment,instruction}.go` | Add handle storage and preserve handles in copies. Keep instruction and operand metadata independent. |
| P22b | `pkg/parser/ast/{stream,stream_symbols,stream_join,stream_rename,stream_rewrite}.go` | Invalidate views on accepted mutations. Clear published handles and preserve old snapshot reads. |
| P22b | `pkg/parser/ast/{equal,equal_reflect,equal_test}.go` | Ignore operational handles in equality. Add handle parity cases. |
| P22b | `pkg/parser/ast/node_edit_test.go` | Include correspondence, revision, range replacement, unsupported-node, and atomic failure tests. |
| P22b | `pkg/parser/ast/node_edit_snapshot_test.go` | Verify independent reads and retained source revisions after all stream mutations. |
| P22b | `pkg/parser/ast/node_edit_validation_test.go` | Verify invalid nodes and relocations cannot be published by `Commit` or `Replace`. |
| P22b | `pkg/parser/ast/node_edit_data_bench_test.go` | Retain `TestNodeEditComplexDataPublicationKeepsOwnership`, even if benchmarks are omitted. |
| P22b | `pkg/parser/ast/stream_rewrite_retained_test.go` | Add snapshot, handle-clearing, and native commit ownership cases deferred from P22a. |
| P22b | `pkg/parser/ast/{node_test,instruction_copy_metadata_test,instruction_modifier_ownership_test}.go` | Add handle propagation and native-read modifier ownership assertions deferred from P05/P12. |
| P22b | `pkg/codec/{stream_rewrite_test,stream_equivalence_test}.go` | Add default 6502 native edits and the complete transformation comparison. |

The codec rename/rewrite tests share fixtures from `stream_equivalence_test.go`.
When P21 first needs them, extract the 6502 fixture and its required helpers.
Defer the full equivalence test until P22b. Remove helper constructors and
imports for excluded CPUs; selecting one table row is not sufficient.

The source now avoids some copies of retained internal nodes. Include this
only with independent public reads, independent duplicates, old snapshot
reads, and atomic publication. A simpler owned-copy implementation is valid
if these contracts pass. Record omitted allocation changes as deferred;
do not omit their ownership regression tests.

Use two successive PRs if the combined patch is difficult to review:

1. Extract `EntryEdit`, `Rewrite`, removed-source records, metadata remapping,
   and `stream_rewrite_test.go`. Add the 6502 codec rewrite cases.
2. Extract `NodeEdit`, source handles in base and comment nodes, handle
   propagation through every `Copy`, revision invalidation, and node-edit tests.
   Update equality to ignore operational handles and add those parity cases.

The first step must pass its gates before the second step starts. Keep symbols
and segment assignments rebuilt after edits. Clear obsolete resolved addresses.
Retain metadata only when it is still valid for the replacement instruction.
Keep removed source records separate from emitted code.

**Focused check:** `go test ./pkg/parser/ast/... ./pkg/codec/...`.
**Exit condition:** Moves, duplication, insertion, removal, and replacement retain
correct source correspondence. Conflicting comments, stale views, and foreign
handles fail atomically. New nodes have no original source entry. A full 6502
round-trip test compares bytes, symbols, forms, metadata, and diagnostics after
copy, join, rename, and rewrite.

Adapt `stream_equivalence_test.go` to its 6502 case only. Its helper functions
currently depend on the excluded CPUs' codec constructors even though the
file is in the shared package.

Source references: `41d0738`, `f766c2e`, and `8ce3906`.

Later source references: `042e562`, `9968043`, `2d731ad`, `dfba5ee`, and
`f29b126`. These patches cross P05, P19, and P22. Do not cherry-pick them as
independent merge parts without adaptation.

## Phase E: Documentation and final audit

### P23 — Finish user documentation and the remaining-difference audit

| File | Target action |
|---|---|
| `README.md` | Publish verified 6502 commands and links to the merged compatibility guides. Exclude the source's added CPU/system claims. |
| `docs/library-usage.md` | Complete the default-output, data, package, identity, codec, and stream-edit examples for merged APIs. |
| `examples/ast-first/main.go`, `pkg/retroasm/{doc,example_test}.go`, `pkg/codec/doc.go` | Check that examples and package docs use the merged API and supported CPU. |
| `docs/{asm6-compatibility,ca65-compatibility,nesasm-compatibility,compatibility-mode-plan,x816-compatibility-plan}.md` | Retain verified syntax and limits. Remove obsolete implementation checklists from the material copied to target user docs. |
| `docs/work-branch-changes.md`, `CHANGES_SUMMARY.md` | Update source-branch tracking only. Record each remaining included hunk as merged, superseded, or deferred. |
| `docs/README.md`, `.gitignore` | Adapt the user-guide index to target features. Keep the `!docs/README.md` rule if needed. Remove links to source-only plans and excluded CPU guides from the target index. Leave Z80 fixture rules deferred. |

For each deferred hunk, record its file, symbol or test name, reason, and
follow-up part. An unassigned hunk is an audit failure. New CPU files and the
local module replacement remain excluded regardless of the final diff size.

Update README examples and library usage for APIs already merged. Keep the
support matrix at the target's implemented CPUs. Do not copy the source README
support expansion or Z80 profile claims. Review the x816 reference against the
existing target implementation; omit historical implementation checklists from
user documentation.

Publish compatibility limits with each compatibility part. Publish API
migration notes with P03, P06, P11, P12, and any mandatory byte-order change.
Do not leave those notes until the final part.

Keep `CHANGES_SUMMARY.md` and this extraction plan as branch tracking records.
They are not product documentation to copy to `main`. Do not delete unrelated
files or source-branch work as cleanup.

**Final code candidate checks:** Common gates with the pinned module and no
replacement. Run one configured 6502 assembly in each documented mode. Compare
expected bytes and symbols. Verify CLI help and library examples.

**Exit condition:** Every in-scope hunk is merged, intentionally superseded,
or explicitly deferred with a reason. The remaining difference can still be
large because new architectures remain excluded. Do not use an empty full
branch diff as the completion criterion.

## Mixed-file ownership

| Paths | Parts and extraction constraints |
|---|---|
| `pkg/arch/arch.go` | P12 identity; P15 byte order; P17 relocation; P18 registrations. Defer unused target-state extensions. |
| `pkg/arch/cpu6502/**`, old `m6502/**` | P11 rename; P12 identity; P13 typed default codec; P17 relocations; P18 registration. Defer variant options and added variant modes. |
| `pkg/arch/byte_order_test.go` | P15: retain 6502 coverage and use a test adapter for generic behavior. |
| `pkg/arch/instruction_registration_test.go` | P18: retain only 6502 cases and their helpers. |
| `pkg/arch/relocation_test.go` | P17: shared recorder test; no new CPU required. |
| `pkg/assembler/address_assigning_step.go` | P02 segment start and bounds; P02a reservation extents; P17 relocation capture. |
| `pkg/assembler/address_assigning_step_test.go` | P02a reservation checks; P11 CPU imports; P17 recorder tests without packed fields. |
| `pkg/assembler/assembler.go` | P15 byte-order use, if required; P17 source indices and relocation output. |
| `pkg/assembler/nodes.go` | P06 data values and offsets; P12 ID type; P17 source indices. |
| `pkg/assembler/parse_ast_nodes.go` | P02a variable conversion and offset-counter rejection; P06 data/address parsing; P15 byte-order use, if required. |
| `pkg/assembler/expression_evaluation_step.go`, `generate_opcode_step.go` | P06 independent data and references; P15 byte order; P17 encoder metadata. |
| `pkg/assembler/memory*.go`, `write_output_step.go`, `banked_output_test.go` | P02 only. Adapt imports until P11 lands. |
| `pkg/assembler/process_macros_step.go` | P09 positional macro expansion. |
| Assembler asm6/ca65/x816 tests and `parse_ast_nodes_test.go` | P02 range interactions; P06 data migration; P07/P08 dialect cases; P11 imports. |
| `pkg/parser/parser.go` | P07-P09 labels and macros; P12 identity; P15 stream output. Preserve existing x816 behavior. |
| `pkg/parser/parser*_test.go` | Same behavior split as parser code; P11 import migration. |
| `pkg/parser/opcode_identity_test.go` | P12: keep shared isolation behavior; replace excluded CPU fixtures. |
| `pkg/parser/directives/data.go` | P06 data expressions; P09 NESASM `ds`. Existing align behavior is already on main. |
| `pkg/parser/directives/{addr,base,hex}.go` | P06 data migration. |
| `pkg/parser/directives/directives.go` | P07-P09 mode overlays. Preserve independent maps and existing x816 overlay. |
| `pkg/parser/directives/{ca65,macro}.go` | P08; adapt data handlers to P06. |
| `pkg/parser/directives/nesasm.go` | P07 NES 2.0 parser additions. Existing NESASM handlers stay intact. |
| `pkg/parser/directives/helper.go` | Include only if the retained program-counter path needs it; otherwise omit the redundant wrapper. |
| `pkg/parser/directives/x816.go` | Preserve current target semantics. Handler relocation and prose churn are not a required part. |
| Directive tests | P06-P09 according to the behavior tested. |
| `pkg/parser/ast/node.go`, scalar and composite node files | P05 ownership; P13/P14 used helpers; P22 source handles. |
| `pkg/parser/ast/configuration.go` | P05 nil-safe copy; P07 NES 2.0 item names. |
| `pkg/parser/ast/data.go` | P05 copying; P06 field type; P15 validation used by codec. |
| `pkg/parser/ast/instruction.go` | P05 copy ownership; P12 identity; P13 used operand helpers. |
| `pkg/parser/ast/instruction_copy_metadata_test.go` | P05 copy isolation; P12 scoped IDs; P22 handle propagation. |
| `pkg/parser/ast/instruction_modifier_ownership_test.go` | P05 modifier operator copying; P22 native read ownership. |
| `pkg/parser/ast/node_copy_bench_test.go` | P05 optional benchmark and shared benchmark variable. |
| `pkg/parser/ast/instruction_argument.go` | P05 copy contract; P14 references; defer unused target-only form/state contracts. |
| `pkg/parser/ast/value_format*.go`, `symbol_reference*.go` | P13 value formatting; P06 symbol/addend parsing. |
| `pkg/parser/ast/stream*.go` | P14 model/symbols; P20 join; P21 rename; P22 rewrite and revision handling. |
| `pkg/parser/ast/symbol_rewrite.go` | P21. |
| `pkg/parser/ast/node_edit*.go` | P22. |
| `pkg/parser/ast/equal*.go` | P19; operational-handle additions in P22. |
| AST tests | Same behavior split as AST code. Preserve test helpers required by each extracted test. |
| `pkg/codec/{codec,doc,metadata}.go`, `codec_test.go` | P15 basic operations; P16 formatting; P17 metadata reconciliation. Split shared test imports and helpers. |
| `pkg/codec/cpu6502_test.go` | P15 default codec cases; P17 relocations. Defer variant-specific cases. |
| `pkg/codec/directive_format.go` | P16. |
| `pkg/codec/reservation_test.go` | P02a: adapt regression assertions to assembler tests. P15: direct codec checks. P16: text/format comparisons. Exclude new CPU imports and test invocations. |
| `pkg/codec/metadata_completion_test.go` | P20: metadata completion after insertion, with P17 as a prerequisite. |
| `pkg/codec/stream_equivalence_test.go` | P21: extract required 6502 helpers; P22: complete 6502 comparison. |
| `pkg/codec/stream_rename_test.go`, `stream_rewrite_test.go` | P21 and P22. |
| CLI files and `main_test.go` | P10 mode selection; P11 imports. Exclude new CPU registrations, fixture helpers, and profiles. |
| `pkg/retroasm/default.go` | P03 default fill; P11 imports. Preserve the target's public dispatcher. |
| `pkg/retroasm/{assembler_test,example_test,doc}.go` | P03: add output-length coverage on the target; P11: extract import/API docs. |
| `pkg/lexer/**`, `pkg/number/**` | P01 literals; P04 width three; P15 byte order if used. |
| `README.md`, `docs/library-usage.md`, `examples/ast-first/main.go` | P11 migration examples; P23 verified usage. |
| Compatibility docs | P07-P10 tested reference material; P23 final audit. |
| `go.mod` | Omit the absolute local replacement. |
| `CHANGES_SUMMARY.md`, `docs/work-branch-changes.md` | Branch tracking only. |
| `docs/README.md`, `.gitignore` | P23: target user-guide index and its ignore exception. Exclude Z80 fixture rules and source-only links. |
| Excluded architecture docs/tests/examples | Deferred as listed below. |

## Deferred architecture proposals

These proposals are outside P00-P23. Each needs a separate scope decision and
candidate validation after the shared APIs it uses are merged. The order is
for review only. It is not a claim that one CPU package requires another.
Each file list below names all architecture-specific paths in the current
branch difference. Shared files need selected imports, cases, and helpers.

The following shared contracts were deferred from the 6502 queue. Add each
contract with its first CPU user and its focused regression tests. Later CPU
proposals must use the contract already present on the target.

| First user | Shared files | Changes to merge |
|---|---|---|
| T01: Chip-8 | `pkg/parser/ast/stream.go`, `pkg/parser/ast/stream_test.go`, `pkg/arch/arch.go`, `pkg/arch/relocation_test.go`, `pkg/assembler/address_assigning_step.go`, `pkg/assembler/address_assigning_step_test.go`, `pkg/codec/codec.go` | Add packed relocation fields, validation, recording, and reconciliation for bits inside an instruction word. |
| T02: CPU65816 | `pkg/arch/arch.go`, `pkg/parser/parser.go`, `pkg/parser/parser_test.go`, `pkg/parser/ast/instruction_argument.go`, `pkg/parser/ast/instruction_argument_test.go`, `pkg/parser/ast/stream.go`, `pkg/parser/ast/stream_test.go`, `pkg/codec/codec.go`, `pkg/codec/codec_test.go` | Add parser state methods, instruction state transitions, state-aware construction/parsing, and validation. Keep state private to each parser and stream. |
| T05: Z80 | `pkg/arch/arch.go`, `pkg/parser/parser.go`, registered adapter methods and parser mocks on the target | Pass the original mnemonic to `ParseIdentifier` for target forms that need it. Update all existing adapters and mocks in this candidate. The 6502 queue retains the old signature. |

Review these rows against the target before extraction. If an earlier CPU
proposal already needs a contract, move that row to its first user. Do not
copy unrelated fields or assertions from the same shared files.

| Proposal | Files and changes | Prerequisites and candidate checks |
|---|---|---|
| V01: 6502 variants | `pkg/arch/cpu6502/options.go`, `pkg/arch/cpu6502/cpu6502_test.go`; variant hunks in `cpu6502.go`, parser `operand.go`, `resolved.go`, `instruction.go`, and encoder `generate_opcode_step.go`; variant cases in encoder tests and `pkg/codec/cpu6502_test.go`. Add `WithVariant`, zero-page-relative, zero-page indirect, and absolute-X indirect modes. | P13/P17/P18. Check each registry, default behavior, operand validation, bytes, and relocation widths with `go test ./pkg/arch/cpu6502/... ./pkg/codec/...`. |

### T01 — Chip-8

Packed relocation fields, fixed-width instructions, direct CLI assembly, and examples.

Files to merge with this proposal:

- `pkg/arch/chip8/assembler/address_assigning_step.go`
- `pkg/arch/chip8/assembler/generate_opcode_step.go`
- `pkg/arch/chip8/assembler/generate_opcode_step_test.go`
- `pkg/arch/chip8/chip8.go`
- `pkg/arch/chip8/chip8_assemble_test.go`
- `pkg/arch/chip8/chip8_test.go`
- `pkg/arch/chip8/parser/codec.go`
- `pkg/arch/chip8/parser/codec_test.go`
- `pkg/arch/chip8/parser/instruction.go`
- `pkg/arch/chip8/parser/operand.go`
- `pkg/arch/chip8/parser/resolved.go`
- `pkg/arch/chip8/parser/resolved_copy_test.go`
- `pkg/arch/chip8/parser/symbol_rewrite.go`
- `pkg/codec/chip8_test.go`
- `examples/chip8/README.md`
- `examples/chip8/cube.asm`
- `examples/chip8/hello.asm`

Split shared changes in `pkg/arch/byte_order_test.go`,
`pkg/arch/instruction_registration_test.go`, `pkg/parser/opcode_identity_test.go`,
`pkg/codec/reservation_test.go`, and codec stream comparison/rename/rewrite
tests. Add only this CPU's constructors, imports, and cases. Extract CLI
registration, defaults, validation, help, and tests from
`cmd/retroasm/{architecture,assemble,main,main_test}.go` where this CPU needs
them. Publish checked README commands and guide links with the CPU phase.

**Prerequisites:** The used shared contracts from P05, P12-P18, and P20-P22.
Check exact calls and omit unused shared features.
**Focused check:** `go test ./pkg/arch/chip8/...` and `go test ./pkg/codec/... -run 'CHIP8|Chip8'`; run the common candidate gates after the complete phase.
**Exit condition:** Encoding, references, bounds, copy ownership, and all
public paths added by this proposal pass on its extracted candidate.

### T02 — CPU65816

Parser state, register widths, state transitions, long addresses, block moves, and forward RAM references.

Files to merge with this proposal:

- `pkg/arch/cpu65816/assembler/address_assigning_step.go`
- `pkg/arch/cpu65816/assembler/generate_opcode_step.go`
- `pkg/arch/cpu65816/assembler/generate_opcode_step_test.go`
- `pkg/arch/cpu65816/cpu65816.go`
- `pkg/arch/cpu65816/cpu65816_test.go`
- `pkg/arch/cpu65816/forward_storage_test.go`
- `pkg/arch/cpu65816/parser/addressing.go`
- `pkg/arch/cpu65816/parser/codec.go`
- `pkg/arch/cpu65816/parser/instruction.go`
- `pkg/arch/cpu65816/parser/operand.go`
- `pkg/arch/cpu65816/parser/resolved.go`
- `pkg/arch/cpu65816/parser/resolved_copy_test.go`
- `pkg/arch/cpu65816/parser/state.go`
- `pkg/arch/cpu65816/parser/symbol_rewrite.go`
- `pkg/codec/cpu65816_test.go`
- `docs/cpu65816-support-plan.md`

Split shared changes in `pkg/arch/byte_order_test.go`,
`pkg/arch/instruction_registration_test.go`, `pkg/parser/opcode_identity_test.go`,
`pkg/codec/reservation_test.go`, and codec stream comparison/rename/rewrite
tests. Add only this CPU's constructors, imports, and cases. Extract CLI
registration, defaults, validation, help, and tests from
`cmd/retroasm/{architecture,assemble,main,main_test}.go` where this CPU needs
them. Publish checked README commands and guide links with the CPU phase.

**Prerequisites:** The used shared contracts from P05, P12-P18, and P20-P22.
Check exact calls and omit unused shared features.
**Focused check:** `go test ./pkg/arch/cpu65816/...` and `go test ./pkg/codec/... -run CPU65816`; run the common candidate gates after the complete phase.
**Exit condition:** Encoding, references, bounds, copy ownership, and all
public paths added by this proposal pass on its extracted candidate.

### T03 — CPU68000

Big-endian data, effective addresses, size suffixes, register lists, and instruction encoders.

Files to merge with this proposal:

- `pkg/arch/cpu68000/assembler/address_assigning_step.go`
- `pkg/arch/cpu68000/assembler/address_assigning_step_test.go`
- `pkg/arch/cpu68000/assembler/coverage_test.go`
- `pkg/arch/cpu68000/assembler/encode.go`
- `pkg/arch/cpu68000/assembler/encode_alu.go`
- `pkg/arch/cpu68000/assembler/encode_misc.go`
- `pkg/arch/cpu68000/assembler/generate_opcode_step.go`
- `pkg/arch/cpu68000/assembler/generate_opcode_step_test.go`
- `pkg/arch/cpu68000/cpu68000.go`
- `pkg/arch/cpu68000/cpu68000_test.go`
- `pkg/arch/cpu68000/parser/codec.go`
- `pkg/arch/cpu68000/parser/codec_test.go`
- `pkg/arch/cpu68000/parser/condition.go`
- `pkg/arch/cpu68000/parser/condition_test.go`
- `pkg/arch/cpu68000/parser/effective_address.go`
- `pkg/arch/cpu68000/parser/instruction.go`
- `pkg/arch/cpu68000/parser/operand.go`
- `pkg/arch/cpu68000/parser/register.go`
- `pkg/arch/cpu68000/parser/register_list.go`
- `pkg/arch/cpu68000/parser/register_list_test.go`
- `pkg/arch/cpu68000/parser/resolved.go`
- `pkg/arch/cpu68000/parser/resolved_copy_test.go`
- `pkg/arch/cpu68000/parser/size.go`
- `pkg/arch/cpu68000/parser/size_test.go`
- `pkg/arch/cpu68000/parser/symbol_rewrite.go`
- `pkg/codec/cpu68000_test.go`
- `docs/cpu68000-support-plan.md`

Split shared changes in `pkg/arch/byte_order_test.go`,
`pkg/arch/instruction_registration_test.go`, `pkg/parser/opcode_identity_test.go`,
`pkg/codec/reservation_test.go`, and codec stream comparison/rename/rewrite
tests. Add only this CPU's constructors, imports, and cases. Extract CLI
registration, defaults, validation, help, and tests from
`cmd/retroasm/{architecture,assemble,main,main_test}.go` where this CPU needs
them. Publish checked README commands and guide links with the CPU phase.

**Prerequisites:** The used shared contracts from P05, P12-P18, and P20-P22.
Check exact calls and omit unused shared features.
**Focused check:** `go test ./pkg/arch/cpu68000/...` and `go test ./pkg/codec/... -run CPU68000`; run the common candidate gates after the complete phase.
**Exit condition:** Encoding, references, bounds, copy ownership, and all
public paths added by this proposal pass on its extracted candidate.

### T04 — SM83

Game Boy operands, LDH forms, typed instruction groups, and opcode generation.

Files to merge with this proposal:

- `pkg/arch/sm83/assembler/address_assigning_step.go`
- `pkg/arch/sm83/assembler/generate_opcode_step.go`
- `pkg/arch/sm83/assembler/generate_opcode_step_test.go`
- `pkg/arch/sm83/parser/codec.go`
- `pkg/arch/sm83/parser/instruction.go`
- `pkg/arch/sm83/parser/operand.go`
- `pkg/arch/sm83/parser/register.go`
- `pkg/arch/sm83/parser/resolved_copy_test.go`
- `pkg/arch/sm83/parser/symbol_rewrite.go`
- `pkg/arch/sm83/sm83.go`
- `pkg/arch/sm83/sm83_test.go`
- `pkg/codec/sm83_test.go`
- `docs/sm83-support-plan.md`

Split shared changes in `pkg/arch/byte_order_test.go`,
`pkg/arch/instruction_registration_test.go`, `pkg/parser/opcode_identity_test.go`,
`pkg/codec/reservation_test.go`, and codec stream comparison/rename/rewrite
tests. Add only this CPU's constructors, imports, and cases. Extract CLI
registration, defaults, validation, help, and tests from
`cmd/retroasm/{architecture,assemble,main,main_test}.go` where this CPU needs
them. Publish checked README commands and guide links with the CPU phase.

**Prerequisites:** The used shared contracts from P05, P12-P18, and P20-P22.
Check exact calls and omit unused shared features.
**Focused check:** `go test ./pkg/arch/sm83/...` and `go test ./pkg/codec/... -run SM83`; run the common candidate gates after the complete phase.
**Exit condition:** Encoding, references, bounds, copy ownership, and all
public paths added by this proposal pass on its extracted candidate.

### T05 — Z80

Operand resolution, indexed displacements, instruction profiles, CLI flags, and assembly fixtures.

Files to merge with this proposal:

- `pkg/arch/z80/assembler/address_assigning_step.go`
- `pkg/arch/z80/assembler/address_assigning_step_test.go`
- `pkg/arch/z80/assembler/coverage_test.go`
- `pkg/arch/z80/assembler/doc.go`
- `pkg/arch/z80/assembler/generate_opcode_step.go`
- `pkg/arch/z80/assembler/generate_opcode_step_test.go`
- `pkg/arch/z80/options.go`
- `pkg/arch/z80/parser/codec.go`
- `pkg/arch/z80/parser/doc.go`
- `pkg/arch/z80/parser/fuzz_test.go`
- `pkg/arch/z80/parser/instruction.go`
- `pkg/arch/z80/parser/instruction_test.go`
- `pkg/arch/z80/parser/mock_parser_test.go`
- `pkg/arch/z80/parser/operand.go`
- `pkg/arch/z80/parser/profile_test.go`
- `pkg/arch/z80/parser/register.go`
- `pkg/arch/z80/parser/register_test.go`
- `pkg/arch/z80/parser/resolved_copy_test.go`
- `pkg/arch/z80/parser/resolver.go`
- `pkg/arch/z80/parser/resolver_diagnostics.go`
- `pkg/arch/z80/parser/resolver_extended.go`
- `pkg/arch/z80/parser/resolver_indexed.go`
- `pkg/arch/z80/parser/resolver_indirect.go`
- `pkg/arch/z80/parser/resolver_port.go`
- `pkg/arch/z80/parser/resolver_single_operand.go`
- `pkg/arch/z80/parser/resolver_two_operand.go`
- `pkg/arch/z80/parser/resolver_value.go`
- `pkg/arch/z80/parser/symbol_rewrite.go`
- `pkg/arch/z80/profile/doc.go`
- `pkg/arch/z80/profile/profile.go`
- `pkg/arch/z80/profile/profile_test.go`
- `pkg/arch/z80/z80.go`
- `pkg/arch/z80/z80_test.go`
- `pkg/codec/z80_test.go`
- `docs/z80-support-plan.md`
- `docs/z80-branch-changes.md`
- `cmd/retroasm/z80_fixture_test.go`
- `tests/z80/basic.asm`
- `tests/z80/branches.asm`
- `tests/z80/branches_overflow.asm`
- `tests/z80/compatibility.asm`
- `tests/z80/expressions.asm`
- `tests/z80/indexed.asm`
- `tests/z80/indexed_boundaries.asm`
- `tests/z80/io_extended.asm`
- `tests/z80/offsets.asm`
- `tests/z80/offsets_chained.asm`
- `tests/z80/profile_gameboy_subset.asm`
- `tests/z80/profile_gameboy_subset_rejects.asm`
- `tests/z80/profile_strict_documented.asm`
- `tests/z80/profile_strict_documented_rejects.asm`

Split shared changes in `pkg/arch/byte_order_test.go`,
`pkg/arch/instruction_registration_test.go`, `pkg/parser/opcode_identity_test.go`,
`pkg/codec/reservation_test.go`, and codec stream comparison/rename/rewrite
tests. Add only this CPU's constructors, imports, and cases. Extract CLI
registration, defaults, validation, help, and tests from
`cmd/retroasm/{architecture,assemble,main,main_test}.go` where this CPU needs
them. Publish checked README commands and guide links with the CPU phase.

Extract the Z80 fixture exceptions from `.gitignore` in this proposal.
Run `go test ./cmd/retroasm/...` to check the CLI fixture runner, profiles,
defaults, and error cases with the Z80 fixtures present.

**Prerequisites:** The used shared contracts from P05, P12-P18, and P20-P22.
Check exact calls and omit unused shared features.
**Focused check:** `go test ./pkg/arch/z80/...` and `go test ./pkg/codec/... -run Z80`; run the common candidate gates after the complete phase.
**Exit condition:** Encoding, references, bounds, copy ownership, and all
public paths added by this proposal pass on its extracted candidate.

### T06 — x86 library

Register, immediate, and direct operand encoding. Keep library-only scope unless a separate CLI change is approved.

Files to merge with this proposal:

- `pkg/arch/x86/assembler/address_assigning_step.go`
- `pkg/arch/x86/assembler/generate_opcode_step.go`
- `pkg/arch/x86/instruction.go`
- `pkg/arch/x86/parser/instruction.go`
- `pkg/arch/x86/types.go`
- `pkg/arch/x86/x86.go`
- `pkg/arch/x86/x86_test.go`

Add the x86 byte-order case in `pkg/arch/byte_order_test.go`. Do not infer
a complete typed codec contract from the presence of the architecture adapter.

**Prerequisites:** The used shared contracts from P05, P12-P18, and P20-P22.
Check exact calls and omit unused shared features.
**Focused check:** `go test ./pkg/arch/x86/...`; run the common candidate gates after the complete phase.
**Exit condition:** Encoding, references, bounds, copy ownership, and all
public paths added by this proposal pass on its extracted candidate.

Use `docs/README.md`, `README.md`, and `docs/library-usage.md` only for features
present on the target after each proposal. Keep `docs/z80-branch-changes.md`
as historical evidence unless it has a separate target documentation purpose.
The absolute replacement in `go.mod` is excluded from every proposal.

## Progress record

All parts remain proposed in this refreshed plan. No part has been extracted
or merged by this task. No candidate build boundary has been validated here.
No code tests or linters were run for this documentation change. Verification
for this plan consists of local branch/base checks, source and diff review,
path ownership review, and `git diff --check`.

For each future part, record:

```text
Part:
Target base:
Candidate commit:
PR:
Included behavior and paths:
Excluded mixed hunks:
Public behavior/API migration:
Dependency and workspace configuration:
Focused checks and results:
Common gates and results:
Target merge commit and CI result:
Remaining issues or deferred hunks:
```
