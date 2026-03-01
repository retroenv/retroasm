# Branch `z80_support` vs `main` — Detailed Change Summary

50 files changed, ~7100 insertions, ~160 deletions across three categories:
1. **Shared infrastructure** — changes to existing packages needed by any architecture
2. **Z80 implementation** — new packages under `pkg/arch/z80/`
3. **Tests and fixtures** — new test files and assembly fixture sources

---

## 1. Shared Infrastructure Changes

These changes are architecture-agnostic and could be merged to main independently of the Z80 implementation.

### `.gitignore`

- Changed `tests/` (ignore all) to `tests/*` (ignore direct children only).
- Added `!tests/z80/` and `!tests/z80/*.asm` to allow the Z80 fixture directory and all `.asm` files within it.

### `pkg/parser/ast/` — New AST node types

Three new files, zero changes to existing AST nodes.

#### `pkg/parser/ast/instruction_argument.go` *(new)*

Two new node types for architecture-specific operand payloads:

```go
// InstructionArgument stores an architecture-specific typed instruction argument value.
// Value is any — used to carry a ResolvedInstruction struct from the architecture parser
// through the shared pipeline without the pipeline needing to know its type.
type InstructionArgument struct {
    *node
    Value any
}

// InstructionArguments stores multiple instruction operands in source order.
// Used for multi-operand architectures (Z80 uses 2 operands; 6502 uses 1).
type InstructionArguments struct {
    *node
    Values []Node
}
```

Both implement `Copy()`. `InstructionArgument.Copy()` is a shallow copy (safe because the `Value` payload is treated as immutable after creation). `InstructionArguments.Copy()` deep-copies each child via `value.Copy()`.

#### `pkg/parser/ast/expression.go` *(new)*

A new node type wrapping `*expression.Expression` for expression-backed operand values:

```go
type Expression struct {
    *node
    Value *expression.Expression
}
```

Allows operands like `target+delta` or `(ix+disp)` to carry a full expression AST rather than a pre-evaluated literal. `Copy()` nil-guards the `Value` pointer.

#### `pkg/parser/ast/instruction_argument_test.go` *(new)*

Unit tests for `InstructionArgument.Copy()` and `InstructionArguments.Copy()` covering value preservation and type identity after copy.

#### `pkg/parser/ast/node_test.go` *(modified)*

Added `TestExpression_Copy` verifying that `Expression.Copy()` produces a distinct `*expression.Expression` with the same token count.

---

### `pkg/assembler/parse_ast_nodes.go` — New argument types in instruction parsing

**Key change**: `parseInstruction` extracted its argument-conversion logic into a standalone `convertInstructionArgument` function, then extended that function to handle the two new AST node types.

Before: `parseInstruction` contained an inline `switch` on `ast.Number`, `ast.Label`, `ast.Identifier`.

After: `convertInstructionArgument` handles those three cases plus:

```go
case ast.InstructionArgument:
    // passes arg.Value through directly; modifiers are not allowed
    if len(modifiers) > 0 {
        return nil, errors.New("modifiers are not supported for typed instruction arguments")
    }
    return arg.Value, nil

case ast.InstructionArguments:
    // converts each operand recursively into a []any
    return convertInstructionArgumentList(arg, modifiers)
```

New helper `convertInstructionArgumentList` iterates the operand list, recursively calling `convertInstructionArgument` for each, producing `[]any`.

Added sentinel error `errNilInstructionArgument`.

**6502 impact**: None. The existing `ast.Number`, `ast.Label`, `ast.Identifier` paths are unchanged.

#### `pkg/assembler/parse_ast_nodes_test.go` *(modified)*

Added four test cases to `TestParseInstruction`:
- typed argument without modifiers → value passed through
- typed argument with modifiers → error
- multi-operand argument list → `[]any` with correct types
- multi-operand with modifiers → error

---

### `pkg/assembler/address_assigning_step.go` — New argument value types

Extended `ArgumentValue` to handle the new AST node types and added overflow-safe offset arithmetic.

**New cases in `ArgumentValue`**:

```go
case ast.Number:
    return arg.Value, nil  // direct uint64 return

case ast.Label:
    return aa.ArgumentValue(reference{name: arg.Name})  // delegate to reference path

case ast.Identifier:
    return aa.ArgumentValue(reference{name: arg.Name})  // same

case ast.Expression:
    return aa.argumentExpressionValue(arg)  // evaluate via expression engine
```

**New method `argumentExpressionValue`**: evaluates an `ast.Expression` through the scope/expression subsystem. Handles both normal expressions and `$`-relative (program counter) expressions via `IsEvaluatedAtAddressAssign()`.

**New method `addressWidth()`**: returns `aa.arch.AddressWidth()` with nil-safe fallback to 16. Used during expression evaluation to provide correct word-width context.

**Refactored offset arithmetic**: The existing `int64`/`uint64` addition inside the `reference` resolution branch was extracted into:

```go
func applyInt64Offset(base, offset int64) (int64, error)   // checks MaxInt64/MinInt64 overflow
func applyUint64Offset(base uint64, offset int64) (uint64, error) // checks underflow/overflow
```

These replace the previous unchecked `v + uint64(offset)` arithmetic.

#### `pkg/assembler/address_assigning_step_test.go` *(modified)*

Added `TestAddressAssign_ArgumentValueExpression` with two sub-tests:
- arithmetic expression `1 + 2` → `uint64(3)`
- program-counter expression `$ + 1` at PC=`0x200` → `uint64(0x201)`

---

### `pkg/assembler/generate_opcode_step.go` — Context propagation fix

Small but important: when constructing the `addressAssign` used during opcode generation, added `arch` and `programCounter` fields:

```go
// Before:
assigner := &addressAssign[T]{
    currentScope: currentScope,
}

// After:
assigner := &addressAssign[T]{
    arch:           arch,
    currentScope:   currentScope,
    programCounter: n.Address(),
}
```

This ensures expression operands that reference `$` (program counter) or call `addressWidth()` during opcode generation have the correct context.

---

### `pkg/retroasm/default.go` — Architecture-generic assembly dispatch

Previously hard-coded to m6502. Now dispatches based on registered architecture.

**Key structural changes**:

1. Added `configAny() any` method to `ArchitectureAdapter[T]` — returns the underlying `*config.Config[T]` as `any` for type-switch dispatch.

2. New `resolveArchitectureConfig()` method on `defaultAssembler`:
   - 0 registered architectures → fall back to m6502 (backward compatible)
   - 1 registered architecture → use it via `adapterConfig()`
   - 2+ registered architectures → prefer `"6502"` if present, else `errAmbiguousArchitecture`

3. New `assembleASTWithArchitecture` and `assembleTextWithArchitecture` methods type-switch on the config:
   ```go
   case *config.Config[*cpum6502.Instruction]:  // 6502 path
   case *config.Config[*archz80.InstructionGroup]: // Z80 path
   ```
   Both dispatch to the same generic `assembleASTWithConfig[T]` / `assembleTextWithConfig[T]`.

4. Extracted config loading into `readAssemblerConfig[T]` and base-address application into `applyBaseAddress[T]` — these are now generic and work for any `config.Config[T]`.

5. New error sentinels: `errAmbiguousArchitecture`, `errArchitectureAdapterMismatch`, `errArchitectureNotRegistered`, `errUnsupportedArchitectureConfig`.

---

### `cmd/retroasm/main.go` — CLI multi-architecture support

**Removed**: single `supportedCPU = "6502"` and `supportedSystem = "nes"` constants.

**Added**: full CPU/system matrix:

```go
const (
    cpu6502 = string(arch.M6502)  // "6502"
    cpuZ80  = string(arch.Z80)    // "z80"

    systemNES        = string(arch.NES)
    systemGeneric    = string(arch.Generic)
    systemGameBoy    = string(arch.GameBoy)
    systemZXSpectrum = string(arch.ZXSpectrum)
)

var supportedSystemsByCPU = map[string]map[string]struct{}{
    cpu6502: {systemNES: {}, systemGeneric: {}},
    cpuZ80:  {systemGeneric: {}, systemGameBoy: {}, systemZXSpectrum: {}},
}

var defaultSystemByCPU = map[string]string{
    cpu6502: systemNES,
    cpuZ80:  systemGeneric,
}

var defaultCPUBySystem = map[string]string{
    systemNES:        cpu6502,
    systemGeneric:    cpuZ80,  // note: generic defaults to Z80
    systemGameBoy:    cpuZ80,
    systemZXSpectrum: cpuZ80,
}
```

**Refactored `validateAndProcessArchitecture`** into a pipeline:
1. `normalizeArchitectureOptions` — trims/lowercases all three flag values
2. `setDefaultArchitecture` — if no flags given, default to `6502/nes`
3. `validateSystem` — validates system name is known; normalises via `arch.SystemFromString`
4. `validateCPU` — validates CPU name is known; normalises via `arch.FromString`
5. `applyDerivedArchitectureDefaults` — fills missing CPU from system or vice versa
6. `validateArchitectureCompatibility` — cross-checks CPU+system against `supportedSystemsByCPU`
7. `validateZ80Profile` — parses and normalises the `z80-profile` flag; rejects non-default profile with non-Z80 CPU

**New CLI flag**: `-z80-profile` (values: `default`, `strict-documented`, `gameboy-z80-subset`).

**Replaced** hard-coded m6502 registration in `assembleFile` with `registerArchitectureForCPU(asm, cpu, z80Profile)` which switches on CPU name to register the correct architecture.

**Flag defaults changed**: `-cpu` and `-system` now default to `""` (resolved programmatically) instead of `"6502"` and `"nes"`.

---

## 2. Z80 Implementation (All New Files)

### `pkg/arch/z80/z80.go`

Architecture adapter implementing `arch.Architecture[*InstructionGroup]`.

```go
type InstructionGroup struct {
    Name     string
    Variants []*cpuz80.Instruction
}
```

`New(opts ...Option) *config.Config[*InstructionGroup]` builds instruction groups from four opcode tables:
- `cpuz80.Opcodes[256]`, `cpuz80.EDOpcodes[256]`, `cpuz80.DDOpcodes[256]`, `cpuz80.FDOpcodes[256]`
- Plus CB family (11 instruction pointers: `CBRlc`…`CBSet`)
- Plus indexed-bit families (`DdcbShift`, `DdcbBit`, `DdcbRes`, `DdcbSet`, `FdcbShift`…`FdcbSet`)

Deduplication by pointer identity using `slices.Contains`.

Implements:
- `AddressWidth() int` → `16`
- `Instruction(name string) (*InstructionGroup, bool)` — case-insensitive lookup
- `ParseIdentifier(...)` → delegates to `z80parser.ParseIdentifierWithProfile`
- `AssignInstructionAddress(...)` → delegates to `z80assembler.AssignInstructionAddress`
- `GenerateInstructionOpcode(...)` → delegates to `z80assembler.GenerateInstructionOpcode`

### `pkg/arch/z80/options.go`

Functional options pattern:

```go
type Option func(*options)

func WithProfile(kind profile.Kind) Option { ... }

func defaultOptions() options {
    return options{profile: profile.Default}
}
```

### `pkg/arch/z80/parser/register.go`

Table-driven classification of Z80 operand tokens:

| Table | Entries | Purpose |
|-------|---------|---------|
| `registerParamByName` | 16 | Named registers (A, B, C, D, E, H, L, IXH, IXL, IYH, IYL, AF, BC, DE, HL, SP) |
| `conditionParamByName` | 8 | Conditions (NZ, Z, NC, C, PO, PE, P, M) |
| `indirectRegisterParamsByName` | 7 | Indirect forms ((BC), (DE), (HL), (SP), (IX), (IY), (C)) |
| `registerParamsByNumber` | 10 | Numeric operands (RST vectors $00/$08/$10…$38 + IM modes 0/1/2) |

`C` appears in both `registerParamByName` and `conditionParamByName`; the parser returns both candidates and the resolver chooses based on mnemonic context.

### `pkg/arch/z80/parser/instruction.go`

Entry points:
- `ParseIdentifier(parser, mnemonic, variants)` — calls `ParseIdentifierWithProfile` with `Default`
- `ParseIdentifierWithProfile(parser, mnemonic, variants, profile)` — parses operands, resolves, validates profile

Operand parsing covers:
- Zero operands: `NOP`, `RET`, etc.
- Single scalar operand: registers, conditions, numbers, labels, identifiers
- Parenthesized indirect: `(HL)`, `(IX)`, `(IY)`
- Indexed displacement: `(IX+d)`, `(IY-d)`, compact form `iy-2` as single token
- Parenthesized value: `(nn)`, `(label)`, `(label+n)`, `($nn+n)`, `($nn+n-m)`
- Expression: `target+delta`, `label+3-1`, `($10+3-1)` (chained offsets)
- Expression displacement: `(ix+disp)` where `disp` is a symbolic expression

### `pkg/arch/z80/parser/resolver.go`

Resolves (mnemonic + parsed operands) → exact `*cpuz80.Instruction` variant + selected parameters.

**Single-operand pass** handles:
- Implied (zero operands)
- Register/condition single operand
- Numeric single operand (IM n, RST n)
- Value single operand (JP nn, CALL nn, JR e, LD r,n)

**Two-operand passes** (executed in order):
1. `resolveRegisterPairOperands` — reg-reg pairs (`LD A,B`)
2. `resolveExtendedRegisterMemoryOperands` — `LD r,(nn)` / `LD (nn),r`
3. `resolvePortRegisterOperands` — `IN r,(C)` / `OUT (C),r`
4. `resolvePortImmediateOperands` — `IN A,(n)` / `OUT (n),A`
5. `resolveRegisterIndexedOperands` — `LD r,(IX+d)` / `BIT n,(IX+d)`
6. `resolveIndexedRegisterOperands` — `LD (IX+d),r`
7. `resolveRegisterValueOperands` — `LD r,n`
8. `resolveValueRegisterOperands` — `BIT/RES/SET b,r`

Direction disambiguation for `LD` indexed variants uses opcode-bit pattern analysis (`matchesIndexedLoadDirection`, `matchesExtendedLoadDirection`).

Diagnostic errors for three high-confusion cases:
- `C` condition vs register
- Immediate vs parenthesized/indirect form
- Indexed load direction mismatch

### `pkg/arch/z80/assembler/address_assigning_step.go`

`AssignInstructionAddress(assigner, ins)`:
1. Extracts `ResolvedInstruction` from `ins.Argument()` via type assertion
2. Calls `opcodeInfoForResolvedInstruction` to get `OpcodeInfo` and `AddressingMode`
3. Sets `ins.Address`, `ins.Addressing`, `ins.Size`
4. Returns next PC = `pc + opcodeInfo.Size`

Opcode lookup priority:
- 1 register param → `RegisterOpcodes[param]`
- 2 register params → `RegisterPairOpcodes[[2]param]`
- Explicit addressing mode → `Addressing[mode]`
- Single-entry addressing map → use the sole entry

### `pkg/arch/z80/assembler/generate_opcode_step.go`

`GenerateInstructionOpcode(assigner, ins)`:
1. Resolves instruction and opcode info (same lookup as address assignment)
2. Dispatches to `buildOpcodeBytes`

Addressing-family byte emission:

| Family | Bytes emitted |
|--------|--------------|
| `Implied`, `Register`, `Bit` | prefix (if any) + opcode |
| `Immediate` | prefix+opcode + 1 or 2 operand bytes |
| `Extended` | prefix+opcode + 2-byte LE address |
| `Relative` | prefix+opcode + signed 1-byte offset (`target - (addr + size)`) |
| `RegisterIndirect`, `Port` | prefix+opcode + optional 1 operand byte |

Special cases:
- **CB bit family** (`BIT/RES/SET b,r`): `buildBitOpcode` → `[CB, base + (bit<<3) + regCode]`
- **Indexed bit family** (`BIT n,(IX/IY+d)`): `buildIndexedBitOpcode` → `[prefix, CB, displacement, base + (bit<<3) + regCode]`

### `pkg/arch/z80/profile/profile.go`

Three profile kinds:

| Kind | String | Behaviour |
|------|--------|-----------|
| `Default` | `"default"` | Allows all opcodes including undocumented |
| `StrictDocumented` | `"strict-documented"` | Rejects undocumented opcodes (SLL, IXH/IXL variants, specific ED/CB ranges) |
| `GameBoySubset` | `"gameboy-z80-subset"` | Rejects DD/ED/FD prefixes and DJNZ/EX/EXX/IN/OUT mnemonics |

Validation occurs at parse time (after resolution, before returning AST node) for immediate, actionable error messages.

Undocumented opcode detection uses:
- Instruction pointer identity (e.g. `cpuz80.CBSll`)
- Mnemonic name matching (`"sll"`, `"inf"`, `"outf"`)
- Opcode byte range checks (CB `0x30`–`0x37`)
- Prefix+opcode key lookup for ED-prefixed undocumented variants

---

## 3. Tests

### New test packages

| File | Package | What it tests |
|------|---------|---------------|
| `pkg/arch/z80/z80_test.go` | `z80` | Instruction lookup, case-insensitive keys, CB/indexed-bit variant presence |
| `pkg/arch/z80/assembler/address_assigning_step_test.go` | `assembler` | 1/2/3/4-byte instruction size assignment, error paths |
| `pkg/arch/z80/assembler/generate_opcode_step_test.go` | `assembler` | Core opcode emission matrix, boundary values (relative ±128/127, displacement 0x00/0xFF, port 0x00/0xFF, address 0x0000/0xFFFF), error paths |
| `pkg/arch/z80/assembler/coverage_test.go` | `assembler` | Exhaustive: synthesises a valid `ResolvedInstruction` per opcode variant from all tables; validates address assignment + opcode generation for every variant |
| `pkg/arch/z80/parser/instruction_test.go` | `parser` | 800+ lines; covers all operand forms, error cases, diagnostic message quality assertions |
| `pkg/arch/z80/parser/register_test.go` | `parser` | Register/condition/indirect/indexed classification table coverage |
| `pkg/arch/z80/parser/fuzz_test.go` | `parser` | Property-based determinism: same token stream → same success/error outcome |
| `pkg/arch/z80/parser/profile_test.go` | `parser` | Profile-gated instruction acceptance/rejection at parser level |
| `pkg/arch/z80/profile/profile_test.go` | `profile` | Parse/validation for all three profiles |
| `cmd/retroasm/z80_fixture_test.go` | `main` | End-to-end fixture assembly with byte-accurate assertions |

### Modified test files

| File | Changes |
|------|---------|
| `cmd/retroasm/main_test.go` | Extended CPU/system validation matrix, CPU-specific architecture registration tests, assembly with config file for both CPUs |
| `pkg/assembler/address_assigning_step_test.go` | Added `TestAddressAssign_ArgumentValueExpression` |
| `pkg/assembler/parse_ast_nodes_test.go` | Added 4 cases for typed and multi-operand arguments |
| `pkg/parser/ast/node_test.go` | Added `TestExpression_Copy` |

### Integration fixtures (`tests/z80/`)

| File | Purpose | Key instructions |
|------|---------|-----------------|
| `basic.asm` | Core smoke path | `NOP`, `LD BC,n`, `LD A,n`, `BIT 3,A`, `JR NZ,label` |
| `branches.asm` | Control flow | `JR NZ`, `JP NZ`, `CALL`, `RET` forward and backward |
| `branches_overflow.asm` | Error regression | 136 NOP padding + `JR` to out-of-range target; asserts error |
| `indexed.asm` | IX/IY prefix path | `LD IX,nn`, `LD A,(IX+5)`, `LD (IY-2),A`, `BIT 3,(IX+5)`, `JP (IX)`, `IM 1`, `RST $38` |
| `io_extended.asm` | Extended/port I/O | `LD A,(nn)`, `LD (nn),A`, `LD BC,(nn)`, `LD (nn),BC`, `IN A,(n)`, `OUT (n),A`, `IN B,(C)`, `OUT (C),E` |
| `offsets.asm` | Tokenized offsets | `JP target+1`, `LD A,(data+1)`, `IN A,($10+1)` |
| `offsets_chained.asm` | Chained offsets | `JP target+2-1`, `LD A,(data+3-1)`, `IN A,($10+3-1)` |
| `expressions.asm` | Symbolic expressions | `JP target+delta`, `LD A,(table+disp)`, `LD A,(IX+disp)` (symbol-backed displacements) |
| `compatibility.asm` | Mixed control flow | `JR NZ`, `JP`, `LD A,(table+1)`, `DJNZ`, `RET` |
| `indexed_boundaries.asm` | Displacement edge values | `LD A,(IX-128)`, `LD (IY+127),A`, `BIT 0,(IX+127)`, `RES 7,(IY-128)`, `SET 3,(IX-1)` |
| `profile_strict_documented.asm` | Positive: strict profile | `BIT 3,A`, relative jump |
| `profile_gameboy_subset.asm` | Positive: gameboy profile | `NOP`, `LD A,n`, `JR NZ,e` |
| `profile_strict_documented_rejects.asm` | Negative: strict profile | `SLL A` (undocumented) — asserts rejection |
| `profile_gameboy_subset_rejects.asm` | Negative: gameboy profile | `IN A,(n)` — asserts rejection |

---

## 4. Dependency Relationships

Changes can be separated into these layers (each depends only on layers below it):

```
Layer 4: CLI changes (cmd/retroasm/main.go)
            └─ uses Layer 3 Z80 packages + Layer 2 retroasm

Layer 3: Z80 implementation (pkg/arch/z80/)
            └─ uses Layer 1 AST types + retrogolib Z80 opcode data

Layer 2: retroasm library dispatch (pkg/retroasm/default.go)
            └─ uses Layer 1 shared assembler + Layer 3 Z80 config type

Layer 1: Shared assembler extensions (pkg/assembler/ + pkg/parser/ast/)
            └─ no new dependencies on Z80-specific code
```

**Layer 1 changes are fully extractable to main** without pulling in any Z80-specific code. They make the pipeline generic enough to support any multi-operand architecture.

---

## 5. Extraction Plan

### Goal

Merge only the architecture-agnostic improvements to `main` first, keeping the Z80 packages on the feature branch until they are ready to ship. This keeps `main` buildable and all tests green at every step.

### PR 1 — Shared AST extensions (Layer 1, part A)

**Files:**
- `pkg/parser/ast/expression.go` *(new)*
- `pkg/parser/ast/instruction_argument.go` *(new)*
- `pkg/parser/ast/instruction_argument_test.go` *(new)*
- `pkg/parser/ast/node_test.go` *(add `TestExpression_Copy`)*

**What it does:** Adds three new AST node types (`Expression`, `InstructionArgument`, `InstructionArguments`) with `Copy()` implementations and tests. No changes to any existing node or file.

**Risk:** None. Pure additions. No existing code is modified.

**Verification:** `go test ./pkg/parser/ast/...`

---

### PR 2 — Shared assembler extensions (Layer 1, part B)

Depends on: PR 1

**Files:**
- `pkg/assembler/parse_ast_nodes.go` *(extended argument conversion)*
- `pkg/assembler/parse_ast_nodes_test.go` *(new test cases)*
- `pkg/assembler/address_assigning_step.go` *(new argument value types + overflow-safe arithmetic)*
- `pkg/assembler/address_assigning_step_test.go` *(new expression tests)*
- `pkg/assembler/generate_opcode_step.go` *(context propagation fix)*

**What it does:**
- Extends `convertInstructionArgument` to handle `ast.InstructionArgument` and `ast.InstructionArguments`.
- Extends `ArgumentValue` to handle `ast.Number`, `ast.Label`, `ast.Identifier`, and `ast.Expression`.
- Adds `argumentExpressionValue` for expression evaluation with PC context.
- Adds `addressWidth()` helper.
- Replaces unchecked offset arithmetic with `applyInt64Offset`/`applyUint64Offset`.
- Propagates `arch` and `programCounter` into the `addressAssign` used during opcode generation.

**Risk:** Low. The 6502 path is exercised by the existing test suite. The new cases in the type switches are additive. The arithmetic refactor is guarded by the existing test coverage of reference-offset resolution.

**Verification:** `go test ./pkg/assembler/...`

---

### PR 3 — retroasm library dispatch refactor (Layer 2)

Depends on: PR 2

**Constraint:** `pkg/retroasm/default.go` currently imports `archz80` directly for the type-switch dispatch. This import must be removed before the file can land on `main` without the Z80 package.

**Required change:** Replace the two architecture-specific type-switch cases with a single interface-based dispatch. The `configAny() any` method on `ArchitectureAdapter[T]` is already in place; the assembly helpers `assembleASTWithConfig[T]` and `assembleTextWithConfig[T]` are already generic. What is needed is a way to call them without knowing the concrete type `T` at the call site.

**Approach:** Introduce a `configAssembler` interface with a single method and have `ArchitectureAdapter[T]` implement it:

```go
// assembleASTFunc and assembleTextFunc are closures captured at adapter creation.
type configAssembler interface {
    assembleAST(ctx context.Context, nodes []ast.Node, baseAddr uint64) ([]byte, error)
    assembleText(ctx context.Context, source io.Reader, configFile string) ([]byte, error)
}
```

`ArchitectureAdapter[T].CreateAssembler` (or a new unexported method) returns a `configAssembler` backed by its `*config.Config[T]`, calling the generic helpers directly. The `assembleASTWithArchitecture` and `assembleTextWithArchitecture` methods then use `adapterConfig` → cast to `configAssembler` instead of a type switch, eliminating the `archz80` import entirely.

**Files to modify for main:**
- `pkg/retroasm/default.go` *(remove archz80 import; add configAssembler interface)*
- `pkg/retroasm/assembler.go` *(no change required)*

**Files to update on the z80 branch after PR 3 merges:**
- Rebase `z80_support` onto the new `main`; update `ArchitectureAdapter` and `NewArchitectureAdapter` calls if the interface changed.

**Verification:** `go test ./pkg/retroasm/...` on both `main` and the rebased branch.

---

### PR 4 — Z80 architecture package (Layer 3)

Depends on: PR 3 merged and branch rebased

**Files (all new):**
- `pkg/arch/z80/z80.go`
- `pkg/arch/z80/z80_test.go`
- `pkg/arch/z80/options.go`
- `pkg/arch/z80/parser/doc.go`
- `pkg/arch/z80/parser/register.go`
- `pkg/arch/z80/parser/register_test.go`
- `pkg/arch/z80/parser/instruction.go`
- `pkg/arch/z80/parser/instruction_test.go`
- `pkg/arch/z80/parser/mock_parser_test.go`
- `pkg/arch/z80/parser/fuzz_test.go`
- `pkg/arch/z80/parser/profile_test.go`
- `pkg/arch/z80/parser/resolver.go`
- `pkg/arch/z80/assembler/doc.go`
- `pkg/arch/z80/assembler/address_assigning_step.go`
- `pkg/arch/z80/assembler/address_assigning_step_test.go`
- `pkg/arch/z80/assembler/generate_opcode_step.go`
- `pkg/arch/z80/assembler/generate_opcode_step_test.go`
- `pkg/arch/z80/assembler/coverage_test.go`
- `pkg/arch/z80/profile/doc.go`
- `pkg/arch/z80/profile/profile.go`
- `pkg/arch/z80/profile/profile_test.go`

**Risk:** Self-contained new package. Does not modify any existing file. All existing tests remain green.

**Verification:** `go test ./pkg/arch/z80/...`

---

### PR 5 — CLI and integration tests (Layer 4)

Depends on: PR 4

**Files:**
- `.gitignore` *(tests/* change + Z80 fixture allowlist)*
- `cmd/retroasm/main.go` *(multi-architecture validation, `-z80-profile` flag, `registerArchitectureForCPU`)*
- `cmd/retroasm/main_test.go` *(extended validation matrix)*
- `cmd/retroasm/z80_fixture_test.go` *(new)*
- `tests/z80/*.asm` *(14 fixture files)*

**Risk:** The CLI changes are additive (new flags, new CPU option). Existing `-cpu 6502 -system nes` behaviour is preserved by the `setDefaultArchitecture` fast-path (when no flags are given, defaults to 6502/nes). The only observable change to existing users is that `-cpu` and `-system` now default to `""` instead of `"6502"` and `"nes"` — both are functionally equivalent because `setDefaultArchitecture` fills them in before any validation.

**Verification:** `go test ./cmd/retroasm/...` including all 14 Z80 fixture tests and the extended main_test.go matrix.

---

### Summary Table

| PR | Branch from | Merges to | Files changed | Risk | Green gate |
|----|------------|-----------|--------------|------|-----------|
| 1 | `z80_support` | `main` | 4 | None | `./pkg/parser/ast/...` |
| 2 | `z80_support` | `main` | 5 | Low | `./pkg/assembler/...` |
| 3 | `z80_support` (after refactor) | `main` | 1–2 | Low | `./pkg/retroasm/...` |
| 4 | rebased `z80_support` | `main` | 21 | None | `./pkg/arch/z80/...` |
| 5 | rebased `z80_support` | `main` | 5 + 14 | Low | `./cmd/retroasm/...` |

Each PR leaves `main` fully buildable and all tests green.
