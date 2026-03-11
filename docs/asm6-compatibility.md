# asm6 / asm6f Assembler Compatibility

## Overview

asm6 (v1.6) by loopy is a popular 6502 assembler for NES development. asm6f is a community fork adding undocumented opcode support, iNES headers, and symbol file export. Both share identical syntax.

See [Compatibility Mode Infrastructure](compatibility-mode-plan.md) for shared features (anonymous labels, colon-optional labels, number formats).

## Current Support Status

Based on `docs/support.md`, retroasm already supports most asm6 directives:

| Directive | Status | Notes |
|---|---|---|
| `EQU` | Supported | Literal string replacement |
| `=` | Supported | Numeric assignment (re-assignable) |
| `INCLUDE` / `INCSRC` | **Not yet** | Source include |
| `INCBIN` / `BIN` | Supported | Binary include (with offset/size) |
| `DB` / `DW` | Supported | Byte/word data |
| `DL` / `DH` | Supported | Low/high byte data |
| `HEX` | Supported | Compact hex table |
| `DSB` / `DSW` | Supported | Storage with fill |
| `PAD` | Supported | Pad to address |
| `ORG` | Supported | Origin / pad |
| `ALIGN` | Supported | Alignment |
| `FILLVALUE` | Supported | Default fill byte |
| `BASE` | Supported | Relocatable address |
| `IF` / `ELSEIF` / `ELSE` / `ENDIF` | Supported | Conditionals |
| `IFDEF` / `IFNDEF` | Supported | Symbol conditionals |
| `MACRO` / `ENDM` | **Not yet** | Macro definition |
| `REPT` / `ENDR` | **Not yet** | Repeat block |
| `ENUM` / `ENDE` | **Not yet** | RAM variable definition |
| `ERROR` | **Not yet** | Assembly error |

## asm6-Specific Features

### Labels

asm6 labels are case-sensitive. The colon after a label is optional (shared feature, see infrastructure doc).

**Local labels** begin with `@` and are scoped between non-local labels:

```asm
label1:
  @tmp1:     ; local to label1
  @tmp2:
label2:
  @tmp1:     ; different from label1's @tmp1
  @tmp2:
```

**Implementation:** Local labels prefixed with `@` are already partially supported via ca65 cheap local label syntax. In asm6 mode, `@` labels reset scope at each non-local label definition.

**Nameless labels** (`+`/`-`) are covered in the shared infrastructure doc. asm6 additionally allows trailing text on nameless labels (e.g., `+here`) to create more unique forward targets. asm6f clarifies that `+`/`-` labels do NOT break `@local` scope.

### `$` as Program Counter

asm6 uses `$` as the current program counter in expressions:

```asm
here = $
PAD $FFFA     ; equivalent to DSB $FFFA-$
$=9999        ; direct PC assignment (same as BASE)
```

This is already the default behavior in retroasm (`expression.ProgramCounterReference`).

### EQU vs `=`

- `EQU` performs **literal text substitution** (like C `#define`). The value is not evaluated at definition time.
- `=` evaluates to a **number** and the symbol can be reassigned.

Current retroasm likely treats both as numeric assignment. True `EQU` text substitution would require storing the token sequence and replaying it at each use site.

**Priority:** Medium. Most real-world usage of `EQU` is for numeric constants where the distinction doesn't matter.

### INCBIN with Offset/Size

```asm
INCBIN foo.bin, $400           ; read from $400 to EOF
INCBIN foo.bin, $200, $2000    ; read $2000 bytes starting from $200
```

**Status:** Check if the `Include` directive handler supports the optional offset and size parameters.

### MACRO / ENDM

```asm
MACRO setAXY x,y,z
    LDA #x
    LDX #y
    LDY #z
ENDM

setAXY $12,$34,$56
```

- Arguments are comma-separated in the macro definition
- Labels inside macros are local (scoped to macro expansion)
- Note: asm6 uses `MACRO name args` syntax (name comes after MACRO keyword)

**Current state:** retroasm has a `Macro` handler in the directives map. Verify it supports asm6's syntax variant.

### REPT / ENDR

```asm
i=0
REPT 256
    DB i
    i=i+1
ENDR
```

- Repeat a block N times
- Labels inside REPT are local
- Requires re-assignable `=` symbols to be useful

### ENUM / ENDE

```asm
ENUM $200
foo:    db 0
foo2:   db 0
ENDE
```

Temporarily reassigns PC and suppresses output. Used for defining RAM variable addresses. Equivalent to x816's `.table`/`.end` or ca65's unnamed `.segment "BSS"`.

### ERROR

```asm
IF x>100
    ERROR "X is out of range :("
ENDIF
```

Stop assembly with a user message. Useful in conditional blocks.

### Expression Operators

asm6 supports C-style operators with standard precedence:

| Precedence | Operators |
|---|---|
| Highest | `( )` |
| Unary | `+ - ~ ! < >` |
| Multiplicative | `* / %` |
| Additive | `+ -` |
| Shift | `<< >>` |
| Relational | `< > <= >=` |
| Equality | `= == != <>` |
| Bitwise AND | `&` |
| Bitwise XOR | `^` |
| Bitwise OR | `\|` |
| Logical AND | `&&` |
| Logical OR | `\|\|` |

Note: `=` in expression context is equality (like C's `==`), and `<>` is inequality (like C's `!=`).

**Unary `<` and `>`** give low/high byte of a 16-bit word (already supported).

### String Arithmetic

```asm
DB "ABCDE"+1        ; equivalent to DB "BCDEF"
DB "ABCDE"-"A"+32   ; equivalent to DB 32,33,34,35,36
```

asm6 allows adding/subtracting values from strings, shifting all character values. This is a data directive feature.

## asm6f Extensions

### Undocumented Opcodes

asm6f adds support for undocumented 6502 opcodes in three tiers:

| Tier | Directive | Opcodes |
|---|---|---|
| Normal | (always available) | `slo`, `rla`, `sre`, `rra`, `sax`, `lax`, `dcp`, `isc`, `anc`, `alr`, `arr`, `axs`, `las` |
| Unstable | `UNSTABLE` | `ahx`, `shy`, `shx`, `tas` |
| Highly Unstable | `HUNSTABLE` | `xaa` |

**Implementation:** Add `UNSTABLE` and `HUNSTABLE` as no-op directives (they just enable the opcodes). The opcodes themselves need to be registered in retrogolib's M6502 instruction set.

### iNES Header Directives (asm6f)

| Directive | Purpose |
|---|---|
| `INESPRG` | PRG ROM banks |
| `INESCHR` | CHR ROM banks |
| `INESMAP` | Mapper number |
| `INESMIR` | Mirroring mode |
| `NES2CHRRAM` | CHR RAM size (NES 2.0) |
| `NES2PRGRAM` | PRG RAM size (NES 2.0) |
| `NES2SUB` | Submapper (NES 2.0) |
| `NES2TV` | TV mode (NES 2.0) |
| `NES2VS` | Vs. Unisystem (NES 2.0) |
| `NES2BRAM` | Battery PRG RAM (NES 2.0) |
| `NES2CHRBRAM` | Battery CHR RAM (NES 2.0) |

Most of the basic iNES directives (`INESPRG`, `INESCHR`, `INESMAP`, `INESMIR`) are already supported. The NES 2.0 directives need to be added.

### Other asm6f Additions

- `IGNORENL` / `ENDINL` — suppress labels in symbol file export (no-op for retroasm)
- `a:` prefix to force absolute addressing for zero-page addresses
- Generic `+`/`-` labels do not break `@local` scope

### Absolute Addressing Prefix (`a:`)

```asm
lda a:$00    ; force absolute addressing even though $00 is zero-page
```

asm6f (and ca65) support `a:` prefix to force absolute (16-bit) addressing for addresses that would otherwise use zero-page (8-bit) mode.

**Implementation:** In the operand parser, detect `a:` prefix and set a flag to force absolute addressing mode.

## Implementation Order

| Step | Feature | Effort |
|---|---|---|
| 1 | Verify existing directive support matches asm6 semantics | Small |
| 2 | `@` local label scoping (reset at non-local labels) | Medium |
| 3 | `INCLUDE` / `INCSRC` source file inclusion | Medium |
| 4 | `ENUM` / `ENDE` RAM variable blocks | Medium |
| 5 | `MACRO` / `ENDM` (verify asm6 syntax variant) | Medium |
| 6 | `REPT` / `ENDR` repeat blocks | Medium |
| 7 | `ERROR` directive | Small |
| 8 | String arithmetic in data directives | Small |
| 9 | `a:` absolute addressing prefix | Small |
| 10 | Undocumented opcode support (asm6f) | Large |
| 11 | NES 2.0 header directives (asm6f) | Medium |
