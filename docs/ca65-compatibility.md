# ca65 Assembler Compatibility

## Overview

ca65 is the assembler component of the cc65 C compiler toolchain. It targets 6502-family processors and is widely used for NES, SNES, C64, and other retro platform development.

See [Compatibility Mode Infrastructure](compatibility-mode-plan.md) for shared features.

## Implementation Status

The ca65 compatibility mode is selected via `--compat ca65`. The following sections describe what is supported and what remains unimplemented.

## Label Syntax

### Standard Labels [done]

```asm
label:          ; standard label with colon (required in ca65)
```

### Cheap Local Labels [done]

```asm
.proc main
    lda #0
@loop:              ; cheap local label (scoped between non-local labels)
    sta $200,x
    dex
    bne @loop
.endproc
```

- Prefixed with `@`
- Scoped between non-local labels via `LocalLabelScoping()` feature flag

### Unnamed Labels [done]

```asm
:       lda $2002   ; unnamed label definition (colon at start of line)
        bne :-      ; reference previous unnamed label
        beq :+      ; reference next unnamed label
        jmp :--     ; reference 2nd previous unnamed label
        jmp :++     ; reference 2nd next unnamed label
```

- Defined with `:` alone at start of line
- Referenced with `:-`/`:+` (nearest), `:--`/`:++` (2nd nearest), etc.
- Enabled via `UnnamedLabels()` feature flag

## Segments [done]

```asm
.segment "CODE"
    lda #0
```

The `.segment` directive switches to a named segment. Segment names are parsed and stored as AST nodes.

## Scoping

### `.proc` / `.endproc` [done]

```asm
.proc main
    jsr init
    jmp loop
.endproc
```

- Defines a named procedure
- The proc name becomes a label at the start address

### `.scope` / `.endscope` [done]

```asm
.scope MyScope
    foo = 42
.endscope
```

- Like `.proc` but does not create an entry-point label
- Supports both named and anonymous scopes

## Program Counter [done]

```asm
* = $8000           ; program counter assignment via AsteriskProgramCounter()
```

Enabled via the `AsteriskProgramCounter()` feature flag.

## Bank Byte Operator [done]

The `^` operator extracts the bank byte (bits 16-23), enabled via `BankByteOperator()`.

## Directives

### Supported via Shared Handlers

These directives are available in all compatibility modes.

| Directive | Handler | Notes |
|---|---|---|
| `.addr` | `Addr` | 16-bit address |
| `.align` | `Align` | Alignment |
| `.byte` / `.db` | `Data` | Byte data |
| `.dh` / `.dl` | `AddrHigh` / `AddrLow` | High/low byte of address |
| `.endproc` | `EndProc` | End procedure |
| `.if` / `.else` / `.endif` | Conditional | Conditionals |
| `.ifdef` / `.ifndef` | Conditional | Symbol conditionals |
| `.incbin` | `Include` | Binary include |
| `.include` | `Include` | Source include |
| `.macro` / `.endmacro` | `Macro` | Block macros |
| `.org` | `Base` | Set PC |
| `.proc` | `Proc` | Named procedure |
| `.res` | `Res` | Reserve storage |
| `.segment` | `Segment` | Named segment switch |
| `.setcpu` | `SetCPU` | Set target CPU (skipped) |
| `.word` / `.dw` | `Data` | Word data |

### ca65-Specific Directives

These are added by `ca65Handlers()` on top of the shared set.

| Directive | Handler | Status | Notes |
|---|---|---|---|
| `.asciiz` | `Asciiz` | Done | Null-terminated string |
| `.assert` | `NoOp` | Stub | Compile-time assertion (ignored) |
| `.autoimport` | `NoOp` | Stub | Auto-import toggle (ignored) |
| `.bankbytes` | `BankBytes` | Done | Bank byte of addresses |
| `.charmap` | `NoOp` | Stub | Character remapping (ignored) |
| `.condes` | `NoOp` | Stub | Constructor/destructor tables (ignored) |
| `.debuginfo` | `NoOp` | Stub | Debug info toggle (ignored) |
| `.define` | `NoOp` | Stub | Text macro (ignored) |
| `.endrepeat` | `Endr` | Done | Alias for `.endr` |
| `.endscope` | `EndScope` | Done | End scope |
| `.export` / `.exportzp` | `NoOp` | Stub | Symbol export (ignored) |
| `.fatal` | `Error` | Done | Fatal error message |
| `.faraddr` | `FarAddr` | Done | 24-bit address data |
| `.feature` | `NoOp` | Stub | Parser feature flags (ignored) |
| `.global` / `.globalzp` | `NoOp` | Stub | Combined import/export (ignored) |
| `.hibytes` / `.lobytes` | `AddrHigh` / `AddrLow` | Done | High/low bytes of addresses |
| `.import` / `.importzp` | `NoOp` | Stub | Symbol import (ignored) |
| `.linecont` | `NoOp` | Stub | Line continuation (ignored) |
| `.list` / `.listbytes` | `NoOp` | Stub | Listing control (ignored) |
| `.local` | `NoOp` | Stub | Local symbol in macro (ignored) |
| `.out` | `Out` | Done | Print message during assembly |
| `.repeat` | `Rept` | Done | Alias for `.rept` |
| `.scope` | `Scope` | Done | Named/anonymous scope |
| `.undefine` | `NoOp` | Stub | Remove text macro (ignored) |
| `.warning` | `Warning` | Done | Warning message |
| `.error` | `Error` | Done | Error message (via shared handler) |

### Stub Directives

Directives marked "Stub" are recognized and consumed without error, but have no effect on output. This allows ca65 source files that use these directives to assemble without modification, as long as the directives are not essential to correctness.

Key stubs that may need full implementation for complex projects:
- `.define` / `.undefine` -- text macro substitution
- `.feature` -- runtime parser behavior flags
- `.charmap` -- character remapping

## Not Yet Implemented

The following ca65 features are not yet supported.

| Feature | Notes |
|---|---|
| `.enum` / `.endenum` | Enumeration constants (available in shared handlers but not ca65-specific) |
| `.struct` / `.endstruct` | Structure definitions |
| `.union` / `.endunion` | Union definitions |
| Keyword expression operators | `.shl`, `.shr`, `.and`, `.or`, `.not`, `.mod`, `.bitand`, `.bitor`, `.bitxor`, `.bitnot` |
| Scope resolution (`::`) | `MyScope::foo` symbol access |
| String functions | `.concat`, `.left`, `.right`, `.mid`, `.string`, `.sprintf`, `.ident` |
| Symbol predicates | `.blank`, `.const`, `.ref`, `.match`, `.xmatch` |
| Size operators | `.sizeof`, `.loword`, `.hiword` |
| Addressing mode overrides | `a:`, `z:`, `f:` prefixes |

## Notes

- ca65 normally requires an external linker (ld65). retroasm handles segment layout internally.
- Full ca65 compatibility is a large effort. The current implementation targets the subset needed for common NES projects.
