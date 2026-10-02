# Documentation guide

These documents describe the `work2` source reviewed on 2026-10-02, unless a
section identifies historical evidence. Source features are not all present
on local `main`. Tests named in a document are coverage references, not a
record of a passing run from this documentation review.

## Use the assembler

| Document | Purpose |
|---|---|
| [Library usage](library-usage.md) | High-level API, configuration limits, owned streams, and resolved output |
| [Compatibility modes](compatibility-mode-plan.md) | Mode selection and shared syntax limits |
| [asm6 / asm6f](asm6-compatibility.md) | Directive handlers and output limits |
| [ca65](ca65-compatibility.md) | Labels, scopes, data, and ignored linker directives |
| [NESASM](nesasm-compatibility.md) | Local labels, macros, and explicit bank layout requirements |
| [x816](x816-compatibility-plan.md) | Data widths, source syntax, and remaining gaps |

## Review branch integration

Start with the [gradual merge plan](work-branch-changes.md). It defines the
6502-only target scope, file changes, dependencies, exclusions, and candidate
checks for P00-P23. Do not copy all source-branch support claims to `main`.

| Architecture reference | Scope |
|---|---|
| [65816](cpu65816-support-plan.md) | Addressing and register-width state |
| [68000](cpu68000-support-plan.md) | Effective addresses, typed operands, and big-endian encoding |
| [SM83](sm83-support-plan.md) | Game Boy CPU implementation and integration files |
| [Z80](z80-support-plan.md) | Operand resolution, profiles, and fixtures |
| [Z80 history](z80-branch-changes.md) | Historical implementation record; not a current merge plan |

These architectures are excluded from the current gradual merge plan. Each
needs a separate extraction proposal and candidate validation. Filenames that
end in `-plan.md` are retained so existing links continue to work.
