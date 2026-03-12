# M68000 Architecture Support

## Status

**COMPLETE** -- All 74 instruction mnemonics, 14 addressing modes, and comprehensive tests implemented.

## Metrics

| Metric | Count |
|--------|-------|
| Instruction mnemonics | 74 |
| Addressing modes | 14 (12 standard + StatusReg + QuickImmediate) |
| Condition codes | 16 (+ 2 aliases: HS=CC, LO=CS) |
| Test packages | 3 (m68000, assembler, parser) |
| Integration test functions | 14 |
| Parser unit test cases | 40+ |
| Assembler unit test cases | 30+ |
| Coverage subtests | 74 (one per mnemonic) |

## Architecture Overview

- `*m68000.Instruction` as generic type T (like M6502, not grouped like Z80)
- 24-bit address width
- Big-endian opcode output (unlike M6502/Z80 which are little-endian)
- `lastMnemonic` field bridges `Instruction()` to `ParseIdentifier()` for condition code and size suffix resolution

### Instruction Lookup

`Instruction(name)` resolves mnemonics in three steps:

1. Exact match against `m68000.Instructions` map
2. Condition code extraction (BEQ -> Bcc, DBNE -> DBcc, SHI -> Scc)
3. Size suffix stripping (MOVE.L -> MOVE), then retry steps 1-2

### Parser Dispatch

`ParseIdentifier` dispatches by instruction name to specialized parsers:

| Parser | Instructions |
|--------|-------------|
| No-operand | ILLEGAL, NOP, RESET, RTE, RTR, RTS, TRAPV |
| Branch | Bcc, BRA, BSR |
| DBcc | DBcc |
| Data-reg-only | EXT, SWAP |
| EXG | EXG |
| Generic 1/2-EA | All remaining instructions |
| LINK | LINK |
| MOVEM | MOVEM |
| MOVEQ | MOVEQ |
| Quick | ADDQ, SUBQ |
| STOP | STOP |
| TRAP | TRAP |
| UNLK | UNLK |

### Addressing Modes

| Mode | Syntax | EA Field |
|------|--------|----------|
| AbsLong | (xxx).L | 7/1 |
| AbsShort | (xxx).W | 7/0 |
| AddrRegDirect | An | 1/reg |
| AddrRegIndirect | (An) | 2/reg |
| DataRegDirect | Dn | 0/reg |
| Displacement | d16(An) | 5/reg |
| Immediate | #imm | 7/4 |
| Indexed | d8(An,Xn) | 6/reg |
| PCDisplacement | d16(PC) | 7/2 |
| PCIndexed | d8(PC,Xn) | 7/3 |
| PostIncrement | (An)+ | 3/reg |
| PreDecrement | -(An) | 4/reg |
| QuickImmediate | (internal) | in opcode |
| StatusReg | SR/CCR | (internal) |

### Registers

D0-D7, A0-A7, SP (alias for A7), SR, CCR, USP, PC

### Opcode Encoding

Encoding is split across three files by category:

| File | Encoders |
|------|----------|
| `encode.go` | EA field encoding, extension words, size bits, register list reversal |
| `encode_alu.go` | ADD/SUB, ADDA/SUBA, ADDX/SUBX, AND/OR, CMP/CMPA/CMPM, EOR, immediate ALU, quick, unary, MOVE |
| `encode_misc.go` | Branch, Bcc/DBcc/Scc, BCD, bit ops, CHK, EXG, EXT, JMP/JSR, LEA/PEA, LINK/UNLK, MOVEM/MOVEP/MOVEQ, mul/div, shift/rotate, STOP, SWAP, TRAP |

## File Structure

```
pkg/arch/m68000/
  m68000.go                              - Architecture adapter (New, Instruction, ParseIdentifier, delegates)
  m68000_test.go                         - Integration tests (14 test functions)
  assembler/
    address_assigning_step.go            - Instruction size calculation
    address_assigning_step_test.go       - Size calculation unit tests + mock infra
    coverage_test.go                     - Full mnemonic coverage test (74 subtests)
    encode.go                            - EA encoding helpers, extension words, size bits
    encode_alu.go                        - ALU instruction encoders (MOVE, ADD/SUB, AND/OR/EOR, CMP, immediate, quick, unary)
    encode_misc.go                       - Non-ALU encoders (branch, bit, BCD, shift, control flow, MOVEM, etc.)
    generate_opcode_step.go              - Opcode encoding dispatcher
    generate_opcode_step_test.go         - Opcode encoding unit tests
  parser/
    condition.go                         - Condition code parsing (Bcc/DBcc/Scc, 16 codes + 2 aliases)
    condition_test.go                    - Condition code unit tests
    effective_address.go                 - EA mode parsing (all 14 addressing modes)
    instruction.go                       - Main parser dispatcher
    register.go                          - Register name mapping (D0-D7, A0-A7, SP, SR, CCR, USP, PC)
    register_list.go                     - MOVEM register list parsing (D0-D3/A0-A2 -> 16-bit bitmask)
    register_list_test.go                - Register list unit tests
    resolved.go                          - ResolvedInstruction + EffectiveAddress types
    size.go                              - Size suffix parsing (.B/.W/.L)
    size_test.go                         - Size suffix unit tests
```

## Completed Work

| Date | Change |
|------|--------|
| 2026-03-06 | Full M68000 implementation: 74 mnemonics, 14 EA modes, big-endian output, CLI integration |
| 2026-03-06 | Parser + assembler unit tests, AddressWidth fix (type assertion for 24-bit) |
| 2026-03-06 | Linter cleanup: dead code, staticcheck SA fixes, nolint directives, modernize |
| 2026-03-06 | Coverage test: all 74 M68000 instruction names exercised through full encode pipeline |
