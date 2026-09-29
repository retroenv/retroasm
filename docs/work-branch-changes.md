# Gradual Merge Plan for work2

## Scope and verified base

Move the shared assembler, compatibility, and 6502 changes to `main` in small
parts. Each part must build and pass its checks with only the existing 6502
architecture. Do not merge the complete `work2` branch.

| Item | Verified value |
|---|---|
| Review date | 2026-09-29 |
| Source branch | `work2` |
| Source commit | `586df3b3e73e7f5305cd4cb1b57676c4d701692d` |
| Local target commit | `0e4317aa7dc0fd94dfb33e93f26a7ef8e285d3b3` |
| Merge base | Same commit as local `main` |
| Working tree before this plan | Clean |
| Complete branch difference | 267 files; 38,827 added lines; 931 removed lines |

These values refer to local Git refs. The remote target was not fetched.
`main` is an ancestor of the source commit. At this base, the two-dot and
three-dot diffs describe the same changes. The old instruction to synchronize
`main` into `work2` before extraction is no longer necessary.

The previous plan described an August base. It did not include the later AST,
stream, codec, relocation, and memory-output changes. Use this plan in its
place. Old statements that CPU6502 changes are comments only, that AST helpers
have no production callers, and that `default.go` has only format changes are
no longer applicable.

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
support claims. The `.gitignore` change only enables Z80 fixtures. Exclude it.

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

The path-level inventory excludes 131 files, with 25,941 added lines and one
removed line. The other 136 files have 12,886 added lines and 930 removed lines.
The second group contains mixed files, branch reports, and the local module
replacement. It is an inventory to split, not a patch to merge in full.

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
| P15 | Parser stream output and basic codec operations | P13, P14 | High |
| P16 | Data and directive stream formatting | P07-P09, P15 | High |
| P17 | Instruction relocation output | P02, P06, P15 | High |
| P18 | Instruction registration inventory | P13 | Medium |
| P19 | Exact AST equality | P05, P12, P14 | Medium |
| P20 | Stream joins and metadata reindexing | P17 | High |
| P21 | Atomic symbol rename | P17 | High |
| P22 | Explicit stream rewrites and native node edits | P19-P21 | High |
| P23 | User docs and final scope audit | All included parts | Medium |

P01, P04, and P05 are independent of the output fixes. Keep the listed order
for a simple review queue. If urgent, move an independent part earlier without
changing its scope. P22 has two review steps described below.

## Phase A: Correctness and data ownership

### P00 — Establish the target baseline

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

Extract the `#0x3c` lexer handling and its regression case from
`pkg/lexer/{lexer.go,lexer_test.go}`.

**Focused check:** `go test ./pkg/lexer/...`.
**Exit condition:** Prefixed and plain hexadecimal forms produce the expected
number token. Existing decimal, binary, and hexadecimal token cases still pass.

### P02 — Preserve memory-relative output and bank fill

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

### P03 — Change the default library output length separately

Extract the removal of `fill = yes` from the private `defaultConfig` in
`pkg/retroasm/default.go`, plus the related expectations in library tests.
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

Extract width-three range checks and little-endian output from
`pkg/number/{number.go,number_test.go}`. Preserve `WriteToBytes` behavior for
all existing widths. Keep this part small; byte-order selection can wait for
P15/P17.

**Focused check:** `go test ./pkg/number/...`.
**Exit condition:** `$ffffff` fits in three bytes; `$1000000` fails the width
check; all existing number cases pass. This supports ca65 data, with no new CPU.

### P05 — Make AST copies own mutable data

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

Extract `InstructionRegistration`, its provider, sorted default 6502 output,
and the 6502 case from `instruction_registration_test.go`.

**Focused check:** `go test ./pkg/arch/...`.
**Exit condition:** Names and addressing selectors are unique, stable, sorted,
and use valid 6502 IDs. Do not include the 65816 combined-addressing test or
imports of excluded packages. Add form providers only if an included caller
requires them.

## Phase D: Safe stream transformations

### P19 — Add exact AST equality

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

Extract `AppendStream`, associated insertion/reindexing logic, and tests.
Use generic test state for join compatibility. Retain positions, annotations,
comments, symbols, relocations, and segment changes at their new indices.

**Focused check:** `go test ./pkg/parser/ast/... -run 'Stream.*(Append|Replace|Join)'`.
**Exit condition:** Empty streams, self-append, incompatible state, and invalid
metadata have tested behavior. Rejected joins leave the destination unchanged.
Returned metadata cannot mutate either input.

Source reference: `b9b1294`.

### P21 — Rename symbols atomically

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

## Phase E: Documentation and final audit

### P23 — Finish user documentation and the remaining-difference audit

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
| `pkg/assembler/address_assigning_step.go` | P02 segment start and bounds; P17 relocation capture. |
| `pkg/assembler/assembler.go` | P15 byte-order use, if required; P17 source indices and relocation output. |
| `pkg/assembler/nodes.go` | P06 data values and offsets; P12 ID type; P17 source indices. |
| `pkg/assembler/parse_ast_nodes.go` | P06 data/address parsing; P15 byte-order use, if required. |
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
| `pkg/codec/metadata_completion_test.go` | P17/P20: add once metadata and insertion paths exist. |
| `pkg/codec/stream_equivalence_test.go` | P22: 6502 case and its helper closure only. |
| `pkg/codec/stream_rename_test.go`, `stream_rewrite_test.go` | P21 and P22. |
| CLI files and `main_test.go` | P10 mode selection; P11 imports. Exclude new CPU registrations, fixture helpers, and profiles. |
| `pkg/retroasm/default.go` | P03 default fill; P11 imports. Preserve the target's public dispatcher. |
| `pkg/retroasm/{assembler_test,example_test,doc}.go` | P03 output assertions; P11 import/API docs. |
| `pkg/lexer/**`, `pkg/number/**` | P01 literals; P04 width three; P15 byte order if used. |
| `README.md`, `docs/library-usage.md`, `examples/ast-first/main.go` | P11 migration examples; P23 verified usage. |
| Compatibility docs | P07-P10 tested reference material; P23 final audit. |
| `go.mod` | Omit the absolute local replacement. |
| `CHANGES_SUMMARY.md`, `docs/work-branch-changes.md` | Branch tracking only. |
| `.gitignore`, excluded architecture docs/tests/examples | Excluded. |

## Progress record

All parts are planned. No part has been extracted or merged by this task.
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
