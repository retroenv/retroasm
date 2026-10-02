# SM83 Architecture Support

## Branch status

SM83 implementation is present on `work2`, reviewed on 2026-10-02.
It is absent from the local `main` at `0e4317a` and excluded from the
[current merge plan](work-branch-changes.md). This document replaces the
old instructions to create files that now exist.

## Architecture

The adapter uses `*InstructionGroup` to group instruction variants by mnemonic.
Addresses are 16-bit and data uses little-endian byte order. SM83 has its own
adapter; it is not the Z80 Game Boy subset profile.

The parser supports register, immediate, indirect, relative, extended, and
bit operands. SM83-specific paths include `LDH`, `STOP`, `SWAP`,
`(HL+)`, `(HL-)`, C-indirect high memory, and signed SP offsets.
There is no IX/IY or Z80 profile selection in this adapter.

## Implementation files

| File | Role |
|---|---|
| `pkg/arch/sm83/sm83.go` | Config, instruction groups, scoped IDs, registrations, byte order, and codec delegation |
| `pkg/arch/sm83/parser/instruction.go` | Operand parsing and instruction resolution |
| `pkg/arch/sm83/parser/register.go` | Register, condition, and indirect-register lookup |
| `pkg/arch/sm83/parser/operand.go` | Owned typed operands and resolved forms |
| `pkg/arch/sm83/parser/codec.go` | Typed construction, validation, and formatting |
| `pkg/arch/sm83/parser/symbol_rewrite.go` | Symbol rewriting for typed operands |
| `pkg/arch/sm83/assembler/address_assigning_step.go` | Selected instruction sizes and addresses |
| `pkg/arch/sm83/assembler/generate_opcode_step.go` | Opcode bytes and instruction relocation fields |
| `cmd/retroasm/architecture.go` | SM83 registration and Game Boy/system defaults |
| `cmd/retroasm/main.go` | CPU flag help |

## CLI selection

```sh
retroasm -cpu sm83 -system gameboy -c memory.cfg -o game.gb program.asm
retroasm -cpu sm83 -system generic -c memory.cfg -o program.bin program.asm
```

`-system gameboy` without an explicit CPU selects SM83. Explicit Z80 selection
uses the separate Z80 implementation. A system flag does not generate a Game
Boy cartridge header or checksum. Supply the required memory layout and data.

## Test evidence and future extraction

| File | Coverage in source |
|---|---|
| `pkg/arch/sm83/sm83_test.go` | Config and instruction lookup |
| `pkg/arch/sm83/assembler/generate_opcode_step_test.go` | Instruction encoding cases |
| `pkg/arch/sm83/parser/resolved_copy_test.go` | Typed operand copy ownership |
| `pkg/codec/sm83_test.go` | Build, validate, format, assembly, and relocation cases |
| `cmd/retroasm/main_test.go` | CPU/system selection and registration |

No code tests were run for this documentation review. A future SM83 proposal
must extract its adapter, parser, encoders, tests, and CLI registration together,
after the required shared contracts exist. Validate it with:

```sh
go test ./pkg/arch/sm83/... ./pkg/codec/... ./cmd/retroasm/...
make build
make lint
make test
```

Use the pinned dependency without a local replacement. Record the candidate
commit and results. Existing source tests do not prove an extracted candidate
passes, and this document does not authorize a merge.
