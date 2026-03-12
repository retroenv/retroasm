# M68000 Architecture Support Plan

## Current Status
- **Status:** COMPLETE
- **Last Updated:** 2026-03-06
- **Summary:** M68000 fully implemented with all 62 instruction mnemonics, 14 EA modes, and comprehensive unit + integration tests.

## Metrics

| Metric | Baseline | Current |
|--------|----------|---------|
| M68000 test packages | 0 | 3 (m68000, assembler, parser) |
| Integration test cases | 0 | 14 |
| Parser unit test cases | 0 | 40+ |
| Assembler unit test cases | 0 | 30+ |

## Targets

### Target 1: Scaffold + Architecture Adapter (COMPLETED ✓)
**Solution:** `pkg/arch/m68000/m68000.go` — `New()` returns `*config.Config[*m68000.Instruction]`; `architecture` struct implements `AddressWidth() int` returning 24; `Instruction(name)` handles exact lookup + condition code variants + size-stripped base names; `lastMnemonic` bridges `Instruction()` → `ParseIdentifier()`.

### Target 2: Register + Size + Condition Parsing (COMPLETED ✓)
**Solution:** `parser/register.go` (D0-D7, A0-A7, SP, SR, CCR, PC), `parser/size.go` (`ParseSizeSuffix`, `parseSizeToken`), `parser/condition.go` (`ParseConditionCode` — 16 conditions for Bcc/DBcc/Scc).

### Target 3: Effective Address Parsing (COMPLETED ✓)
**Solution:** `parser/effective_address.go` handles all 14 M68000 addressing modes; `parser/register_list.go` handles MOVEM register lists (D0-D3/A0-A2 → 16-bit bitmask).

### Target 4: Instruction Parser Dispatcher (COMPLETED ✓)
**Solution:** `parser/instruction.go` — `ParseIdentifier` dispatches by instruction name to specialized parsers (no-operand, branch, DBcc, MOVEM, MOVEQ, TRAP, STOP, LINK, UNLK, SWAP/EXT, EXG, ADDQ/SUBQ, generic 1/2-EA). Token stream size suffix parsed via dot + B/W/L tokens.

### Target 5: Address Assignment (COMPLETED ✓)
**Solution:** `assembler/address_assigning_step.go` — `instructionSize` computes instruction byte count from resolved EA modes; `eaExtensionSize` returns per-EA extension word sizes.

### Target 6: Opcode Encoding (COMPLETED ✓)
**Solution:** `assembler/generate_opcode_step.go` — reverse of retrogolib's hierarchical line decoder; `assembler/encode.go` — EA field encoding helpers and big-endian extension word generation.

### Target 7: CLI Integration (COMPLETED ✓)
**Solution:** `cmd/retroasm/main.go` — `cpuM68000` constant, `supportedSystemsByCPU`, `registerArchitectureForCPU`; `pkg/retroasm/default.go` — type switch cases for `*cpum68000.Instruction`.

### Target 8: Unit Tests + AddressWidth Fix (COMPLETED ✓)
**Solution:**
- `pkg/arch/m68000/parser/condition_test.go` — 29 cases covering all Bcc/DBcc/Scc conditions and case insensitivity
- `pkg/arch/m68000/parser/size_test.go` — 17 cases for `ParseSizeSuffix` and `parseSizeToken`
- `pkg/arch/m68000/parser/register_list_test.go` — 15 cases including ranges, slash separators, and error cases
- `pkg/arch/m68000/assembler/address_assigning_step_test.go` — 14 size calculation cases + error test; mock infrastructure
- `pkg/arch/m68000/assembler/generate_opcode_step_test.go` — 16 opcode encoding cases covering no-operand, MOVEQ, MOVE, CLR, SWAP, EXT, UNLK, TRAP, ADDQ, branch
- `pkg/retroasm/default.go` — `ArchitectureAdapter.AddressWidth()` now uses type assertion so M68000 correctly returns 24 instead of hardcoded 16

## File Structure

```
pkg/arch/m68000/
  m68000.go                              - Architecture adapter
  m68000_test.go                         - Integration tests (14 test functions)
  parser/
    resolved.go                          - ResolvedInstruction + EffectiveAddress types
    register.go                          - Register name mapping (D0-D7, A0-A7, SP, SR, CCR, PC)
    size.go                              - Size suffix parsing (.B/.W/.L)
    condition.go                         - Condition code parsing (Bcc/DBcc/Scc)
    effective_address.go                 - EA mode parsing (14 addressing modes)
    register_list.go                     - MOVEM register list parsing
    instruction.go                       - Main parser dispatcher
    condition_test.go                    - Condition code unit tests
    size_test.go                         - Size suffix unit tests
    register_list_test.go               - Register list unit tests
  assembler/
    address_assigning_step.go            - Instruction size calculation
    generate_opcode_step.go              - Opcode encoding
    encode.go                            - EA encoding helpers
    address_assigning_step_test.go       - Size calculation unit tests + mock infra
    generate_opcode_step_test.go         - Opcode encoding unit tests
```

### Target 9: Assembler Coverage Test (COMPLETED ✓)
**Problem:** No test verifies that all 62 M68000 instruction names can be assembled without error. Individual targeted tests cover ~15 mnemonics; the remaining 47 have no encoder coverage. Z80 has `coverage_test.go` that exercises every instruction variant.
**Proposed fix:** Add `pkg/arch/m68000/assembler/coverage_test.go` — iterate all entries in `m68000.Instructions`, construct a minimal valid `ResolvedInstruction` for each, run `AssignInstructionAddress` + `GenerateInstructionOpcode`, verify no error and byte output length matches computed size.
**Expected outcome:** 62 instruction names covered; encoding bugs caught for untested instructions.
**Confidence:** HIGH

## Completed Work

| Date | What | Before | After |
|------|------|--------|-------|
| 2026-03-06 | Full M68000 implementation (Targets 1-7) | Not implemented | All 62 mnemonics, 14 EA modes, big-endian, CLI |
| 2026-03-06 | Parser + assembler unit tests, AddressWidth fix | 1 test pkg | 3 test pkgs, 70+ unit test cases |
| 2026-03-06 | Linter cleanup: dead code, staticcheck SA fixes, nolint directives, modernize | 15 issues | 0 issues |
| 2026-03-06 | Coverage test: all 62 M68000 instruction names exercised through full encode pipeline | 0 coverage subtests | 74 subtests pass |
