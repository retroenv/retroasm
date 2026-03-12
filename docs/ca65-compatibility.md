# ca65 Assembler Compatibility

## Overview

ca65 is the assembler component of the cc65 C compiler toolchain. It targets 6502-family processors and is widely used for NES, SNES, C64, and other retro platform development. ca65 has the most extensive feature set of any 6502 assembler.

See [Compatibility Mode Infrastructure](compatibility-mode-plan.md) for shared features.

## Key Differences from retroasm's Current Behavior

ca65's design differs significantly from simpler assemblers:
- **Segment-based** memory model (`.segment "CODE"`) vs. flat `.org` model
- **Scoped symbols** with `.proc`/`.scope` blocks
- **Import/export** system for multi-file linking (via ld65 linker)
- **Rich macro system** with `.define` text macros and `.macro`/`.endmacro` block macros
- **Feature flags** (`.feature`) that alter parser behavior

## Label Syntax

### Standard Labels

```asm
label:          ; standard label with colon (required in ca65)
```

Unlike asm6/x816, ca65 **requires** the trailing colon on labels.

### Cheap Local Labels

```asm
.proc main
    lda #0
@loop:              ; cheap local label (scoped to enclosing .proc/.scope)
    sta $200,x
    dex
    bne @loop
.endproc
```

- Prefixed with `@`
- Scoped to the enclosing `.proc`, `.scope`, or between two non-local labels
- Same concept as asm6's `@` local labels but scoped differently

### Unnamed Labels

```asm
:       lda $2002   ; unnamed label definition (colon at start of line)
        bne :-      ; reference previous unnamed label
        beq :+      ; reference next unnamed label
        jmp :--     ; reference 2nd previous unnamed label
        jmp :++     ; reference 2nd next unnamed label
```

- Defined with `:` alone at start of line
- Referenced with `:-`/`:+` (nearest), `:--`/`:++` (2nd nearest), etc.
- Different syntax from x816/asm6 `+`/`-` labels but same concept

## Segments

ca65 uses a segment model where code and data are assigned to named segments:

```asm
.segment "HEADER"
    .byte "NES", $1A

.segment "CODE"
    lda #0

.segment "VECTORS"
    .word nmi, reset, irq

.segment "CHARS"
    .incbin "tiles.chr"
```

Segments are defined in a linker configuration file (`.cfg`) that maps them to memory regions.

**Implementation approach:** For NES development without a linker, map common segment names to addresses:
- `"HEADER"` → iNES header (offset 0)
- `"CODE"` / `"PRG"` → PRG ROM (typically `$8000` or `$C000`)
- `"VECTORS"` → `$FFFA`
- `"CHARS"` / `"CHR"` → CHR ROM

Alternatively, support `.segment` as a labeled `.org` with predefined mappings.

**Priority:** High — this is fundamental to ca65 source compatibility.

## Scoping

### `.proc` / `.endproc`

```asm
.proc main
    jsr init
    jmp loop
.endproc
```

- Defines a named scope
- The proc name becomes a label at the start address
- Local labels and cheap locals inside are not visible outside
- Can be nested

### `.scope` / `.endscope`

```asm
.scope MyScope
    foo = 42
.endscope

lda #MyScope::foo    ; access scoped symbol with ::
```

- Like `.proc` but doesn't create an entry-point label
- Symbols accessed via `::` scope resolution operator

## Directives

### Already Supported (via shared handlers)

| ca65 Directive | retroasm Handler | Notes |
|---|---|---|
| `.org` | `Base` | Set PC |
| `.byte` / `.db` | `Data` | Byte data |
| `.word` / `.dw` | `Data` | Word data |
| `.res` | `DataStorage` | Reserve storage |
| `.incbin` | `Include` | Binary include |
| `.include` | `Include` | Source include |
| `.if` / `.else` / `.endif` | Conditional | Conditionals |
| `.ifdef` / `.ifndef` | Conditional | Symbol conditionals |
| `.macro` / `.endmacro` | `Macro` | Block macros |

### New Directives to Add

| Directive | Behavior | Priority |
|---|---|---|
| `.segment` | Switch to named segment | High |
| `.proc` / `.endproc` | Named scope with entry label | High |
| `.scope` / `.endscope` | Named scope without entry label | Medium |
| `.import` / `.export` | Symbol import/export (no-op without linker) | Medium |
| `.importzp` / `.exportzp` | Zero-page symbol import/export (no-op) | Medium |
| `.global` / `.globalzp` | Combined import+export (no-op) | Low |
| `.define` | Text macro (token-level substitution) | Medium |
| `.undefine` | Remove text macro | Low |
| `.enum` / `.endenum` | Enumeration (auto-incrementing constants) | Medium |
| `.struct` / `.endstruct` | Structure definition | Low |
| `.union` / `.endunion` | Union definition | Low |
| `.charmap` | Character remapping | Low |
| `.feature` | Enable parser features | Medium |
| `.setcpu` | Set target CPU | Supported |
| `.repeat` / `.endrepeat` | Repeat block (alias for `.rept`) | Small |
| `.local` | Declare local symbol in macro | Small |
| `.addr` | Emit 16-bit address | Supported |
| `.faraddr` | Emit 24-bit address | Small |
| `.bankbytes` | Emit bank byte of addresses | Small |
| `.hibytes` / `.lobytes` | Emit high/low bytes of addresses | Supported (`.dh`/`.dl`) |
| `.asciiz` | Null-terminated string | Small |
| `.align` | Alignment | Supported |
| `.assert` | Compile-time assertion | Medium |
| `.warning` / `.error` / `.fatal` | Diagnostic messages | Small |
| `.out` | Print message during assembly | Small |
| `.linecont` | Enable line continuation with `\` | Low |
| `.condes` | Constructor/destructor tables | Low |

### `.feature` Flags

ca65's `.feature` directive enables optional parser behaviors:

```asm
.feature at_in_identifiers     ; allow @ in identifiers
.feature dollar_in_identifiers ; allow $ in identifiers
.feature labels_without_colons ; asm6-style colon-optional labels
.feature pc_assignment         ; allow * = $8000 for PC assignment
.feature string_escapes        ; enable \n, \t, etc. in strings
.feature org_per_seg           ; allow .org within segments
.feature c_comments            ; enable /* */ block comments
.feature force_range           ; force range checking
.feature underline_in_numbers  ; allow 1_000_000 style numbers
.feature addrsize              ; address size operators
.feature bracket_as_indirect   ; use [] for indirect addressing
```

**Implementation:** `.feature` acts as a runtime parser configuration switch. Each feature flag maps to a boolean in the parser state.

**Priority:** Medium. The most commonly used features are `labels_without_colons`, `pc_assignment`, and `string_escapes`.

### `.define` Text Macros

```asm
.define PPUCTRL $2000
.define PPUMASK $2001
.define SCREEN_W 256

lda PPUCTRL          ; expands to: lda $2000
```

Unlike `.macro`, `.define` performs token-level text substitution (like C `#define`). This is the same concept as asm6's `EQU`.

**Implementation:** Store the token sequence at definition time and replay at each use site during lexing/parsing.

## Expression Operators

ca65 supports both symbolic and keyword operators:

| Operator | Keyword | Meaning |
|---|---|---|
| `+` `-` `*` `/` | | Standard arithmetic |
| `.mod` | | Modulo |
| `.bitand` | `&` | Bitwise AND |
| `.bitor` | `\|` | Bitwise OR |
| `.bitxor` | `^` | Bitwise XOR |
| `.bitnot` | `~` | Bitwise NOT |
| `.shl` | `<<` | Shift left |
| `.shr` | `>>` | Shift right |
| `.and` | `&&` | Logical AND |
| `.or` | `\|\|` | Logical OR |
| `.not` | `!` | Logical NOT |
| `.lobyte` | `<` | Low byte |
| `.hibyte` | `>` | High byte |
| `.bankbyte` | `^` | Bank byte |
| `.loword` | | Low word |
| `.hiword` | | High word |
| `.xmatch` / `.match` | | Token matching (in macros) |
| `.blank` / `.const` / `.ref` | | Symbol predicates |
| `.ident` / `.concat` / `.left` / `.right` / `.mid` / `.string` / `.sprintf` | | String functions |
| `.sizeof` | | Size of scope/struct |

**Priority:** The keyword operators (`.shl`, `.and`, `.or`, etc.) are Medium priority. String functions and predicates are Low.

## Number Formats

| Format | Example | Notes |
|---|---|---|
| `$xx` | `$FF` | Hexadecimal |
| `%xxxx` | `%10101010` | Binary |
| `0xNN` | `0xFF` | C-style hexadecimal |
| Decimal | `255` | Standard decimal |
| `'c'` | `'A'` | Character constant |

ca65 does NOT support trailing `h` or `b` number formats.

## Addressing Mode Overrides

```asm
lda a:$00        ; force absolute addressing (16-bit)
lda z:$1234      ; force zero-page addressing (8-bit, may truncate)
lda f:$1234      ; force far (24-bit) addressing
```

The `a:`, `z:`, and `f:` prefixes override the default addressing mode selection.

## Implementation Order

| Step | Feature | Effort |
|---|---|---|
| 1 | `.segment` with predefined NES mappings | Large |
| 2 | `.proc` / `.endproc` scoping | Medium |
| 3 | Cheap local labels (`@`) with `.proc` scope | Medium |
| 4 | Unnamed labels (`:` / `:-` / `:+`) | Medium |
| 5 | `.define` text macros | Large |
| 6 | `.feature` flag system | Medium |
| 7 | `.scope` / `.endscope` | Medium |
| 8 | `.import` / `.export` (no-op stubs) | Small |
| 9 | `.enum` / `.endenum` | Small |
| 10 | Keyword expression operators (`.shl`, `.and`, etc.) | Medium |
| 11 | `.asciiz`, `.faraddr`, `.bankbytes` | Small |
| 12 | `.charmap` character remapping | Medium |
| 13 | Addressing mode overrides (`a:`, `z:`, `f:`) | Small |
| 14 | `.struct` / `.union` | Large |
| 15 | String functions and predicates | Large |

## Notes

- ca65 requires an external linker (ld65) for final binary output. retroasm would need to either embed a minimal linker or handle segment layout internally.
- ca65 is the most feature-rich assembler and full compatibility is a large effort. Focus on the subset needed for common NES projects first.
- The `.feature labels_without_colons` flag means ca65 can optionally behave like asm6 for label syntax.
