# Assembler Compatibility Modes

This reference describes the `work2` source branch reviewed on 2026-10-02.
It is not a claim that every listed feature is on `main`. See the
[merge plan](work-branch-changes.md) for P07-P10 and their required checks.

## Select a mode

```sh
retroasm -compat ca65 -c memory.cfg -o program.bin program.asm
retroasm -m x816 -c memory.cfg -o program.bin program.asm
```

Without a mode flag, the CLI uses `CompatDefault`. Invalid names return an
error. Mode selection changes parser behavior; it does not select a CPU or
create a cartridge layout.

For library use, set `cfg.CompatibilityMode` before passing the architecture
config to the parser, assembler, codec, or high-level adapter:

```go
cfg := cpu6502.New()
cfg.CompatibilityMode = config.CompatCa65
```

Import `pkg/arch/cpu6502` and `pkg/assembler/config` for this fragment.
`retroasm.TextInput.Format` does not select the mode in the current dispatcher.

## Syntax features

`pkg/assembler/config/compatibility.go` defines these mode constants in order:
`CompatDefault`, `CompatAsm6`, `CompatCa65`, `CompatNesasm`, `CompatX816`.
Use names rather than stored numeric values.

| Feature method | asm6 | ca65 | NESASM | x816 |
|---|---|---|---|---|
| `AnonymousLabels()` | Yes | No | No | Yes |
| `AsteriskProgramCounter()` | No | Yes | Yes | Yes |
| `BankByteOperator()` | No | Yes | No | Yes |
| `ColonOptionalLabels()` | Yes | No | No | Yes |
| `DotLocalLabels()` | No | No | Yes | No |
| `LocalLabelScoping()` | Yes | Yes | No | No |
| `NesasmMacroSyntax()` | No | No | Yes | No |
| `UnnamedLabels()` | No | Yes | No | No |

These switches enable parser paths. They do not establish full dialect
compatibility. In particular, anonymous label definitions and references
are separate behaviors.

- asm6 and x816 accept colon-optional labels in column one.
- asm6 and ca65 scope `@local` labels under the last non-local label.
- NESASM scopes dot-local labels and accepts `name .macro`.
- ca65 accepts unnamed `:` definitions and `:+` / `:-` references.
- ca65, NESASM, and x816 accept `* = expression` for address assignment.
- x816 and ca65 enable bank-byte extraction in supported operand paths.

## Directive maps and limits

`directives.BuildHandlers(mode)` creates a fresh base map and applies one
mode overlay. Parser instances do not share a mutable handler map. Use this
function instead of the deprecated `directives.Handlers` variable.

| Guide | Added behavior | Important limits |
|---|---|---|
| [asm6 / asm6f](asm6-compatibility.md) | Local labels, asm6f directives, NES 2.0 configuration nodes | Header nodes do not generate a header; opcode tier controls are ignored. |
| [ca65](ca65-compatibility.md) | Scopes, unnamed labels, string/address data, macro termination | Imports, exports, and assertions are ignored. `.warning` creates an error; `.out` prints nothing. |
| [NESASM](nesasm-compatibility.md) | Dot-local labels, positional macros, storage aliases | `.bank` does not select an output bank. Section directives are ignored. |
| [x816](x816-compatibility-plan.md) | Data widths, source includes, comment blocks | Width/output directives listed as no-ops do not change CPU state or cartridge layout. |

A no-op handler consumes syntax without implementing its original effect.
Do not use parser acceptance as proof of equivalent output.

## Numbers and expressions

The shared number parser accepts decimal, `$FF` and `0xFF` hexadecimal,
`%1010` and `1010b` binary, and an optional immediate `#` prefix.
These number forms are not selected by compatibility mode.
Trailing `h` hexadecimal and `@` octal are not supported by that parser.

Data items use separate expressions. For example, `DB 1+2, 3*4, (5+1)`
has three values. Address-byte operators and address-size prefixes depend
on their parser context and CPU. NESASM `LOW()` and `HIGH()` function syntax
is not implemented; use supported prefix forms where applicable.

## Source and validation

| Source | Responsibility |
|---|---|
| `pkg/assembler/config/compatibility.go` | Mode names and feature switches |
| `pkg/parser/parser.go` | Labels, aliases, macros, and mode-dependent parsing |
| `pkg/parser/directives/directives.go` | Base handlers and overlays |
| `pkg/assembler/process_macros_step.go` | Macro expansion |
| `cmd/retroasm/{main,assemble,architecture}.go` | Flags, mode parsing, and config propagation |

Parser tests are in `pkg/parser/parser_*_test.go`. Assembly regressions are
in `pkg/assembler/assembler_*_test.go`. Candidate checks must compare bytes
and addresses, not only AST nodes. No code tests were run for this documentation
review. P07-P10 define the checks for extraction to `main`.
