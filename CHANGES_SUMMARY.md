# Change Summary

## Overview

The clean `work2` branch adds typed assembly streams and codecs, expands CPU
and syntax support, and changes shared output and storage handling. This
summary covers local `main...HEAD` on 2026-10-06: source `1a3ab37`, target and
merge base `0e4317a`. The fixed snapshot has 277 changed files, including this
summary; the table below lists the other 276 files.

The [gradual merge plan](docs/work-branch-changes.md) defines the detailed
6502 merge queue. It lists files, changes, dependencies, candidate checks, and
exit conditions for each part. Additional CPUs and 6502 variants remain
separate follow-up work under the existing plan's scope.

## Changes

- **Typed stream and codec:** Owned AST data records positions, comments,
  symbols, relocations, target state, and formatting details. The APIs provide
  exact node comparison, stream joins, atomic symbol renames, explicit rewrites,
  and native edits with independent snapshots. The codec can parse, build,
  format, validate, and assemble typed streams.
- **Targets:** The branch moves `m6502` to `cpu6502` and adds Chip-8, CPU65816,
  CPU68000, SM83, Z80, and x86 packages. The CLI adds the first five targets;
  x86 remains a library package. Explicit 6502 variants add instruction modes.
- **Syntax and CLI:** Parser and directive changes add asm6, ca65, and NESASM
  forms. Existing x816 behavior must be retained during extraction. CLI flags
  select compatibility mode and a Z80 instruction profile.
- **Assembler and output:** Data uses the target byte order and declared width.
  Independent data items can contain forward references and symbol offsets.
  Output uses configured segment addresses, memory bounds, and bank fill.
  The default library configuration emits used bytes without full-bank fill.
- **Storage:** Variable nodes now reach address assignment. Reservations advance
  RAM addresses without load bytes. Negative sizes, integer overflow, CPU
  address overflow, and memory overflow return errors. Offset-counter
  reservations are rejected. CPU65816 retains a wider instruction form while
  a forward reference has no address.
- **Dependency:** Both branch tips use Go 1.25.0 and the same pinned
  `retrogolib` version. Only the absolute local replacement differs. Merge
  candidates must use the pinned module without that replacement.

## Files

| Slice | Status | File | Role |
| --- | --- | --- | --- |
| Typed stream | Added | `pkg/codec/chip8_test.go` | Checks typed codec chip8 behavior. |
| Typed stream | Added | `pkg/codec/codec.go` | Exposes typed parse, build, format, validate, and assemble operations. |
| Typed stream | Added | `pkg/codec/codec_test.go` | Checks parse, build, format, validate, and assemble operations. |
| Typed stream | Added | `pkg/codec/cpu6502_test.go` | Checks typed codec cpu6502 behavior. |
| Typed stream | Added | `pkg/codec/cpu65816_test.go` | Checks typed codec cpu65816 behavior. |
| Typed stream | Added | `pkg/codec/cpu68000_test.go` | Checks typed codec cpu68000 behavior. |
| Typed stream | Added | `pkg/codec/directive_format.go` | Formats typed data and control directives. |
| Typed stream | Added | `pkg/codec/doc.go` | Documents the typed codec package. |
| Typed stream | Added | `pkg/codec/metadata.go` | Completes stream symbols, relocations, and assembly metadata. |
| Typed stream | Added | `pkg/codec/metadata_completion_test.go` | Checks typed codec metadata completion behavior. |
| Typed stream | Added | `pkg/codec/sm83_test.go` | Checks typed codec sm83 behavior. |
| Typed stream | Added | `pkg/codec/stream_equivalence_test.go` | Checks typed codec stream equivalence behavior. |
| Typed stream | Added | `pkg/codec/stream_rename_test.go` | Checks typed codec stream rename behavior. |
| Typed stream | Added | `pkg/codec/stream_rewrite_test.go` | Checks typed codec stream rewrite behavior. |
| Typed stream | Added | `pkg/codec/z80_test.go` | Checks typed codec z80 behavior. |
| Typed stream | Modified | `pkg/parser/ast/alias.go` | Keeps alias node metadata in copies and edits. |
| Typed stream | Modified | `pkg/parser/ast/bank.go` | Keeps bank node metadata in copies and edits. |
| Typed stream | Modified | `pkg/parser/ast/base.go` | Keeps base address node metadata in copies and edits. |
| Typed stream | Modified | `pkg/parser/ast/comment.go` | Preserves comment text in copied entries. |
| Typed stream | Modified | `pkg/parser/ast/condition.go` | Keeps conditional node comments and edit handles. |
| Typed stream | Modified | `pkg/parser/ast/configuration.go` | Keeps configuration node comments and edit handles. |
| Typed stream | Modified | `pkg/parser/ast/data.go` | Stores typed data item lists and preserves comments and edit handles. |
| Typed stream | Modified | `pkg/parser/ast/enum.go` | Keeps enum node comments and edit handles. |
| Typed stream | Added | `pkg/parser/ast/equal.go` | Compares native AST nodes by exact value. |
| Typed stream | Added | `pkg/parser/ast/equal_bench_test.go` | Benchmarks exact AST node comparison. |
| Typed stream | Added | `pkg/parser/ast/equal_reflect.go` | Supports exact comparison of nested AST values. |
| Typed stream | Added | `pkg/parser/ast/equal_test.go` | Checks AST equal behavior. |
| Typed stream | Modified | `pkg/parser/ast/error.go` | Keeps error node metadata in copies and edits. |
| Typed stream | Modified | `pkg/parser/ast/expression.go` | Keeps expression node metadata in copies and edits. |
| Typed stream | Modified | `pkg/parser/ast/function.go` | Keeps function node metadata in copies and edits. |
| Typed stream | Modified | `pkg/parser/ast/identifier.go` | Keeps identifier node metadata in copies and edits. |
| Typed stream | Modified | `pkg/parser/ast/include.go` | Keeps include node metadata in copies and edits. |
| Typed stream | Modified | `pkg/parser/ast/instruction.go` | Stores target-scoped opcode identity and typed instruction metadata. |
| Typed stream | Modified | `pkg/parser/ast/instruction_argument.go` | Copies and validates typed instruction operands and references. |
| Typed stream | Modified | `pkg/parser/ast/instruction_argument_test.go` | Checks AST instruction argument behavior. |
| Typed stream | Added | `pkg/parser/ast/instruction_copy_metadata_test.go` | Checks independent instruction and operand metadata, comments, and nil metadata. |
| Typed stream | Added | `pkg/parser/ast/instruction_modifier_ownership_test.go` | Checks modifier operator copies and independent native edit reads. |
| Typed stream | Modified | `pkg/parser/ast/label.go` | Keeps label comments and edit handles. |
| Typed stream | Modified | `pkg/parser/ast/macro.go` | Keeps macro comments and edit handles. |
| Typed stream | Modified | `pkg/parser/ast/node.go` | Defines node copy, comment, and source entry handle operations. |
| Typed stream | Added | `pkg/parser/ast/node_copy_bench_test.go` | Measures instruction copy allocation and retains a shared benchmark result variable. |
| Typed stream | Added | `pkg/parser/ast/node_edit.go` | Commits native node edits through source entry handles. |
| Typed stream | Added | `pkg/parser/ast/node_edit_data_bench_test.go` | Checks complex data ownership after publication and measures data edits. |
| Typed stream | Added | `pkg/parser/ast/node_edit_snapshot_test.go` | Checks independent snapshot reads, source revisions, operands, and handles. |
| Typed stream | Added | `pkg/parser/ast/node_edit_test.go` | Checks AST node edit behavior. |
| Typed stream | Added | `pkg/parser/ast/node_edit_validation_test.go` | Checks atomic rejection of invalid nodes and relocations during native edits. |
| Typed stream | Modified | `pkg/parser/ast/node_test.go` | Checks AST node behavior. |
| Typed stream | Modified | `pkg/parser/ast/number.go` | Keeps number node metadata in copies and edits. |
| Typed stream | Modified | `pkg/parser/ast/offset_counter.go` | Keeps offset node metadata in copies and edits. |
| Typed stream | Modified | `pkg/parser/ast/operator.go` | Keeps operator node metadata in copies and edits. |
| Typed stream | Modified | `pkg/parser/ast/register.go` | Keeps register node metadata in copies and edits. |
| Typed stream | Modified | `pkg/parser/ast/rept.go` | Keeps repeat node metadata in copies and edits. |
| Typed stream | Modified | `pkg/parser/ast/scope.go` | Keeps scope node metadata in copies and edits. |
| Typed stream | Modified | `pkg/parser/ast/segment.go` | Keeps segment node metadata in copies and edits. |
| Typed stream | Added | `pkg/parser/ast/stream.go` | Defines typed stream entries, symbols, relocations, state, and validation. |
| Typed stream | Added | `pkg/parser/ast/stream_join.go` | Joins streams and rebases entry-indexed metadata. |
| Typed stream | Added | `pkg/parser/ast/stream_join_test.go` | Checks AST stream join behavior. |
| Typed stream | Added | `pkg/parser/ast/stream_rename.go` | Renames symbol definitions and references in one atomic edit. |
| Typed stream | Added | `pkg/parser/ast/stream_rename_test.go` | Checks AST stream rename behavior. |
| Typed stream | Added | `pkg/parser/ast/stream_rewrite.go` | Rewrites entries while retaining source metadata and tracking removed entries. |
| Typed stream | Added | `pkg/parser/ast/stream_rewrite_retained_test.go` | Checks retained snapshots, independent duplicates, cleared handles, and copied relocations. |
| Typed stream | Added | `pkg/parser/ast/stream_rewrite_test.go` | Checks AST stream rewrite behavior. |
| Typed stream | Added | `pkg/parser/ast/stream_symbols.go` | Rebuilds symbol metadata from stream entries. |
| Typed stream | Added | `pkg/parser/ast/stream_test.go` | Checks AST stream behavior. |
| Typed stream | Added | `pkg/parser/ast/symbol_reference.go` | Finds symbol references in expressions. |
| Typed stream | Added | `pkg/parser/ast/symbol_reference_test.go` | Checks AST symbol reference behavior. |
| Typed stream | Added | `pkg/parser/ast/symbol_rewrite.go` | Rewrites symbol names in AST nodes and expressions. |
| Typed stream | Added | `pkg/parser/ast/value_format.go` | Stores formatting intent for typed values and instructions. |
| Typed stream | Added | `pkg/parser/ast/value_format_test.go` | Checks AST value format behavior. |
| Typed stream | Modified | `pkg/parser/ast/variable.go` | Keeps variable node metadata in copies and edits. |
| Shared targets | Modified | `pkg/arch/arch.go` | Extends architecture, parser, registration, byte order, opcode identity, and relocation contracts. |
| Shared targets | Added | `pkg/arch/byte_order_test.go` | Checks native byte order reports for architecture adapters. |
| Shared targets | Added | `pkg/arch/instruction_registration_test.go` | Checks registered instruction selectors and opcode identities. |
| Shared targets | Added | `pkg/arch/relocation_test.go` | Checks architecture relocation recording contracts. |
| Assembler | Modified | `pkg/assembler/address_assigning_step.go` | Uses configured segment addresses, checks reservation bounds, and records instruction relocations. |
| Assembler | Modified | `pkg/assembler/address_assigning_step_test.go` | Checks segment address and reference assignment behavior. |
| Assembler | Modified | `pkg/assembler/assembler.go` | Selects target byte order, tracks source entries, and exposes instruction relocations. |
| Assembler | Modified | `pkg/assembler/assembler_asm6_test.go` | Checks asm6 assembly behavior. |
| Assembler | Modified | `pkg/assembler/assembler_ca65_test.go` | Checks ca65 assembly behavior. |
| Assembler | Modified | `pkg/assembler/assembler_x816_test.go` | Checks x816 assembly behavior. |
| Assembler | Added | `pkg/assembler/banked_output_test.go` | Checks output padding and placement across banks. |
| Assembler | Modified | `pkg/assembler/expression_evaluation_step.go` | Evaluates data item lists with target byte order and defers forward references. |
| Assembler | Modified | `pkg/assembler/generate_opcode_step.go` | Encodes deferred data and references and collects instruction relocations. |
| Assembler | Modified | `pkg/assembler/memory.go` | Writes by absolute memory address and rejects out-of-range writes. |
| Assembler | Added | `pkg/assembler/memory_test.go` | Checks memory bounds and addressed writes. |
| Assembler | Modified | `pkg/assembler/nodes.go` | Carries data item lists, typed opcode IDs, and source entry indices. |
| Assembler | Modified | `pkg/assembler/parse_ast_nodes.go` | Converts typed data and instructions, retains reservation nodes, and rejects offset-counter reservations. |
| Assembler | Modified | `pkg/assembler/parse_ast_nodes_test.go` | Checks AST conversion for data and instruction nodes. |
| Assembler | Modified | `pkg/assembler/process_macros_step.go` | Expands NESASM positional macro parameters. |
| Assembler | Modified | `pkg/assembler/write_output_step.go` | Writes memory buffers with bank padding and address checks. |
| Assembler | Added | `pkg/codec/reservation_test.go` | Checks RAM reservations, capacity, address limits, code gaps, and offset-counter rejection through the codec. |
| Assembler | Modified | `pkg/number/number.go` | Adds three-byte numbers and byte order selection for numeric output. |
| Assembler | Modified | `pkg/number/number_test.go` | Checks numeric encoding under both byte orders. |
| Syntax | Modified | `pkg/lexer/lexer.go` | Accepts a hexadecimal marker after a configured decimal prefix and zero. |
| Syntax | Modified | `pkg/lexer/lexer_test.go` | Checks prefixed hexadecimal number tokenization. |
| Syntax | Modified | `pkg/parser/directives/addr.go` | Parses address extraction directives. |
| Syntax | Modified | `pkg/parser/directives/base.go` | Parses base address directives. |
| Syntax | Added | `pkg/parser/directives/ca65.go` | Adds ca65-specific directive handlers. |
| Syntax | Modified | `pkg/parser/directives/data.go` | Parses data item lists and storage directives. |
| Syntax | Modified | `pkg/parser/directives/directives.go` | Selects directive handlers by compatibility mode. |
| Syntax | Modified | `pkg/parser/directives/directives_test.go` | Checks directive handler selection by syntax mode. |
| Syntax | Modified | `pkg/parser/directives/helper.go` | Provides shared directive parsing helpers. |
| Syntax | Modified | `pkg/parser/directives/hex.go` | Parses hexadecimal data directives. |
| Syntax | Modified | `pkg/parser/directives/macro.go` | Accepts macro terminators for compatibility modes. |
| Syntax | Modified | `pkg/parser/directives/nesasm.go` | Adds NESASM-specific directive handlers. |
| Syntax | Modified | `pkg/parser/directives/noop_test.go` | Checks accepted directives that produce no output. |
| Syntax | Modified | `pkg/parser/directives/x816.go` | Parses x816 data and compatibility directives. |
| Syntax | Added | `pkg/parser/opcode_identity_test.go` | Checks that parsed instructions retain target opcode identities. |
| Syntax | Modified | `pkg/parser/parser.go` | Parses syntax mode labels and macros and emits source-positioned typed streams. |
| Syntax | Modified | `pkg/parser/parser_asm6_test.go` | Checks asm6 local and anonymous labels. |
| Syntax | Added | `pkg/parser/parser_ca65_test.go` | Checks ca65 unnamed and local labels and directives. |
| Syntax | Added | `pkg/parser/parser_nesasm_test.go` | Checks NESASM dot labels and macro syntax. |
| Syntax | Modified | `pkg/parser/parser_test.go` | Checks shared parser behavior after typed stream conversion. |
| Syntax | Modified | `pkg/parser/parser_x816_test.go` | Checks x816 syntax after parser changes. |
| CPU6502 | Renamed | `pkg/arch/cpu6502/assembler/address_assigning_step.go` | Moves from `pkg/arch/m6502/assembler/address_assigning_step.go`. CPU6502: Assigns instruction sizes and addresses. |
| CPU6502 | Renamed | `pkg/arch/cpu6502/assembler/generate_opcode_step.go` | Moves from `pkg/arch/m6502/assembler/generate_opcode_step.go`. CPU6502: Encodes instructions and records operand relocations. |
| CPU6502 | Added | `pkg/arch/cpu6502/assembler/generate_opcode_step_test.go` | Checks CPU6502 generate opcode step behavior. |
| CPU6502 | Renamed | `pkg/arch/cpu6502/assembler/instruction_size.go` | Moves from `pkg/arch/m6502/assembler/instruction_size.go`. CPU6502: Calculates instruction size from selected addressing. |
| CPU6502 | Added | `pkg/arch/cpu6502/cpu6502.go` | Registers the CPU6502 adapter, instruction forms, and target settings. |
| CPU6502 | Added | `pkg/arch/cpu6502/cpu6502_test.go` | Checks CPU6502 adapter behavior. |
| CPU6502 | Added | `pkg/arch/cpu6502/options.go` | CPU6502: Defines target-specific adapter options. |
| CPU6502 | Renamed | `pkg/arch/cpu6502/parser/addressing.go` | Moves from `pkg/arch/m6502/parser/addressing.go`. CPU6502: Defines and selects target addressing forms. |
| CPU6502 | Added | `pkg/arch/cpu6502/parser/codec.go` | CPU6502: Builds, validates, and formats typed target operands. |
| CPU6502 | Renamed | `pkg/arch/cpu6502/parser/instruction.go` | Moves from `pkg/arch/m6502/parser/instruction.go`. CPU6502: Parses target instruction syntax and selects forms. |
| CPU6502 | Renamed | `pkg/arch/cpu6502/parser/instruction_test.go` | Moves from `pkg/arch/m6502/parser/instruction_test.go`. Checks CPU6502 instruction behavior. |
| CPU6502 | Added | `pkg/arch/cpu6502/parser/operand.go` | CPU6502: Defines and classifies typed instruction operands. |
| CPU6502 | Added | `pkg/arch/cpu6502/parser/resolved.go` | CPU6502: Stores resolved instruction forms and copy behavior. |
| CPU6502 | Added | `pkg/arch/cpu6502/parser/resolved_test.go` | Checks CPU6502 resolved behavior. |
| CPU6502 | Deleted | `pkg/arch/m6502/assembler/generate_opcode_step_test.go` | Replaces old encoder tests with cpu6502 tests. |
| CPU6502 | Deleted | `pkg/arch/m6502/m6502.go` | Replaces the old adapter with the cpu6502 adapter. |
| Chip-8 | Added | `pkg/arch/chip8/assembler/address_assigning_step.go` | Chip-8: Assigns instruction sizes and addresses. |
| Chip-8 | Added | `pkg/arch/chip8/assembler/generate_opcode_step.go` | Chip-8: Encodes instructions and records operand relocations. |
| Chip-8 | Added | `pkg/arch/chip8/assembler/generate_opcode_step_test.go` | Checks Chip-8 generate opcode step behavior. |
| Chip-8 | Added | `pkg/arch/chip8/chip8.go` | Registers the Chip-8 adapter, instruction forms, and target settings. |
| Chip-8 | Added | `pkg/arch/chip8/chip8_assemble_test.go` | Checks Chip-8 chip8 assemble behavior. |
| Chip-8 | Added | `pkg/arch/chip8/chip8_test.go` | Checks the Chip-8 adapter. |
| Chip-8 | Added | `pkg/arch/chip8/parser/codec.go` | Chip-8: Builds, validates, and formats typed target operands. |
| Chip-8 | Added | `pkg/arch/chip8/parser/codec_test.go` | Checks Chip-8 codec behavior. |
| Chip-8 | Added | `pkg/arch/chip8/parser/instruction.go` | Chip-8: Parses target instruction syntax and selects forms. |
| Chip-8 | Added | `pkg/arch/chip8/parser/operand.go` | Chip-8: Defines and classifies typed instruction operands. |
| Chip-8 | Added | `pkg/arch/chip8/parser/resolved.go` | Chip-8: Stores resolved instruction forms and copy behavior. |
| Chip-8 | Added | `pkg/arch/chip8/parser/resolved_copy_test.go` | Checks Chip-8 resolved copy behavior. |
| Chip-8 | Added | `pkg/arch/chip8/parser/symbol_rewrite.go` | Chip-8: Renames symbols inside typed instruction operands. |
| CPU65816 | Added | `pkg/arch/cpu65816/assembler/address_assigning_step.go` | CPU65816: Assigns instruction sizes and addresses. |
| CPU65816 | Added | `pkg/arch/cpu65816/assembler/generate_opcode_step.go` | CPU65816: Encodes instructions and records operand relocations. |
| CPU65816 | Added | `pkg/arch/cpu65816/assembler/generate_opcode_step_test.go` | Checks CPU65816 generate opcode step behavior. |
| CPU65816 | Added | `pkg/arch/cpu65816/cpu65816.go` | Registers the CPU65816 adapter, instruction forms, and target settings. |
| CPU65816 | Added | `pkg/arch/cpu65816/cpu65816_test.go` | Checks the CPU65816 adapter. |
| CPU65816 | Added | `pkg/arch/cpu65816/forward_storage_test.go` | Checks forward RAM references through text and typed assembly without RAM load bytes. |
| CPU65816 | Added | `pkg/arch/cpu65816/parser/addressing.go` | CPU65816: Defines and selects target addressing forms. |
| CPU65816 | Added | `pkg/arch/cpu65816/parser/codec.go` | CPU65816: Builds, validates, and formats typed target operands. |
| CPU65816 | Added | `pkg/arch/cpu65816/parser/instruction.go` | CPU65816: Parses target instruction syntax and selects forms. |
| CPU65816 | Added | `pkg/arch/cpu65816/parser/operand.go` | CPU65816: Defines and classifies typed instruction operands. |
| CPU65816 | Added | `pkg/arch/cpu65816/parser/resolved.go` | CPU65816: Stores resolved instruction forms and copy behavior. |
| CPU65816 | Added | `pkg/arch/cpu65816/parser/resolved_copy_test.go` | Checks CPU65816 resolved copy behavior. |
| CPU65816 | Added | `pkg/arch/cpu65816/parser/state.go` | CPU65816: Tracks CPU mode state across parsed instructions. |
| CPU65816 | Added | `pkg/arch/cpu65816/parser/symbol_rewrite.go` | CPU65816: Renames symbols inside typed instruction operands. |
| CPU68000 | Added | `pkg/arch/cpu68000/assembler/address_assigning_step.go` | CPU68000: Assigns instruction sizes and addresses. |
| CPU68000 | Added | `pkg/arch/cpu68000/assembler/address_assigning_step_test.go` | Checks CPU68000 address assigning step behavior. |
| CPU68000 | Added | `pkg/arch/cpu68000/assembler/coverage_test.go` | Checks CPU68000 coverage behavior. |
| CPU68000 | Added | `pkg/arch/cpu68000/assembler/encode.go` | CPU68000: Encodes common instruction forms and effective addresses. |
| CPU68000 | Added | `pkg/arch/cpu68000/assembler/encode_alu.go` | CPU68000: Encodes arithmetic and logic instructions. |
| CPU68000 | Added | `pkg/arch/cpu68000/assembler/encode_misc.go` | CPU68000: Encodes branch, control, and other instruction forms. |
| CPU68000 | Added | `pkg/arch/cpu68000/assembler/generate_opcode_step.go` | CPU68000: Encodes instructions and records operand relocations. |
| CPU68000 | Added | `pkg/arch/cpu68000/assembler/generate_opcode_step_test.go` | Checks CPU68000 generate opcode step behavior. |
| CPU68000 | Added | `pkg/arch/cpu68000/cpu68000.go` | Registers the CPU68000 adapter, instruction forms, and target settings. |
| CPU68000 | Added | `pkg/arch/cpu68000/cpu68000_test.go` | Checks the CPU68000 adapter. |
| CPU68000 | Added | `pkg/arch/cpu68000/parser/codec.go` | CPU68000: Builds, validates, and formats typed target operands. |
| CPU68000 | Added | `pkg/arch/cpu68000/parser/codec_test.go` | Checks CPU68000 codec behavior. |
| CPU68000 | Added | `pkg/arch/cpu68000/parser/condition.go` | CPU68000: Defines branch condition selectors. |
| CPU68000 | Added | `pkg/arch/cpu68000/parser/condition_test.go` | Checks CPU68000 condition behavior. |
| CPU68000 | Added | `pkg/arch/cpu68000/parser/effective_address.go` | CPU68000: Parses and formats effective addresses. |
| CPU68000 | Added | `pkg/arch/cpu68000/parser/instruction.go` | CPU68000: Parses target instruction syntax and selects forms. |
| CPU68000 | Added | `pkg/arch/cpu68000/parser/operand.go` | CPU68000: Defines and classifies typed instruction operands. |
| CPU68000 | Added | `pkg/arch/cpu68000/parser/register.go` | CPU68000: Defines target register operands. |
| CPU68000 | Added | `pkg/arch/cpu68000/parser/register_list.go` | CPU68000: Parses register lists. |
| CPU68000 | Added | `pkg/arch/cpu68000/parser/register_list_test.go` | Checks CPU68000 register list behavior. |
| CPU68000 | Added | `pkg/arch/cpu68000/parser/resolved.go` | CPU68000: Stores resolved instruction forms and copy behavior. |
| CPU68000 | Added | `pkg/arch/cpu68000/parser/resolved_copy_test.go` | Checks CPU68000 resolved copy behavior. |
| CPU68000 | Added | `pkg/arch/cpu68000/parser/size.go` | CPU68000: Parses instruction size suffixes. |
| CPU68000 | Added | `pkg/arch/cpu68000/parser/size_test.go` | Checks CPU68000 size behavior. |
| CPU68000 | Added | `pkg/arch/cpu68000/parser/symbol_rewrite.go` | CPU68000: Renames symbols inside typed instruction operands. |
| SM83 | Added | `pkg/arch/sm83/assembler/address_assigning_step.go` | SM83: Assigns instruction sizes and addresses. |
| SM83 | Added | `pkg/arch/sm83/assembler/generate_opcode_step.go` | SM83: Encodes instructions and records operand relocations. |
| SM83 | Added | `pkg/arch/sm83/assembler/generate_opcode_step_test.go` | Checks SM83 generate opcode step behavior. |
| SM83 | Added | `pkg/arch/sm83/parser/codec.go` | SM83: Builds, validates, and formats typed target operands. |
| SM83 | Added | `pkg/arch/sm83/parser/instruction.go` | SM83: Parses target instruction syntax and selects forms. |
| SM83 | Added | `pkg/arch/sm83/parser/operand.go` | SM83: Defines and classifies typed instruction operands. |
| SM83 | Added | `pkg/arch/sm83/parser/register.go` | SM83: Defines target register operands. |
| SM83 | Added | `pkg/arch/sm83/parser/resolved_copy_test.go` | Checks SM83 resolved copy behavior. |
| SM83 | Added | `pkg/arch/sm83/parser/symbol_rewrite.go` | SM83: Renames symbols inside typed instruction operands. |
| SM83 | Added | `pkg/arch/sm83/sm83.go` | Registers the SM83 adapter, instruction forms, and target settings. |
| SM83 | Added | `pkg/arch/sm83/sm83_test.go` | Checks the SM83 adapter. |
| Z80 | Added | `pkg/arch/z80/assembler/address_assigning_step.go` | Z80: Assigns instruction sizes and addresses. |
| Z80 | Added | `pkg/arch/z80/assembler/address_assigning_step_test.go` | Checks Z80 address assigning step behavior. |
| Z80 | Added | `pkg/arch/z80/assembler/coverage_test.go` | Checks Z80 coverage behavior. |
| Z80 | Added | `pkg/arch/z80/assembler/doc.go` | Z80: Documents this target package. |
| Z80 | Added | `pkg/arch/z80/assembler/generate_opcode_step.go` | Z80: Encodes instructions and records operand relocations. |
| Z80 | Added | `pkg/arch/z80/assembler/generate_opcode_step_test.go` | Checks Z80 generate opcode step behavior. |
| Z80 | Added | `pkg/arch/z80/options.go` | Z80: Defines target-specific adapter options. |
| Z80 | Added | `pkg/arch/z80/parser/codec.go` | Z80: Builds, validates, and formats typed target operands. |
| Z80 | Added | `pkg/arch/z80/parser/doc.go` | Z80: Documents this target package. |
| Z80 | Added | `pkg/arch/z80/parser/fuzz_test.go` | Checks Z80 fuzz behavior. |
| Z80 | Added | `pkg/arch/z80/parser/instruction.go` | Z80: Parses target instruction syntax and selects forms. |
| Z80 | Added | `pkg/arch/z80/parser/instruction_test.go` | Checks Z80 instruction behavior. |
| Z80 | Added | `pkg/arch/z80/parser/mock_parser_test.go` | Supplies token and parser state to Z80 parser tests. |
| Z80 | Added | `pkg/arch/z80/parser/operand.go` | Z80: Defines and classifies typed instruction operands. |
| Z80 | Added | `pkg/arch/z80/parser/profile_test.go` | Checks Z80 profile behavior. |
| Z80 | Added | `pkg/arch/z80/parser/register.go` | Z80: Defines target register operands. |
| Z80 | Added | `pkg/arch/z80/parser/register_test.go` | Checks Z80 register behavior. |
| Z80 | Added | `pkg/arch/z80/parser/resolved_copy_test.go` | Checks Z80 resolved copy behavior. |
| Z80 | Added | `pkg/arch/z80/parser/resolver.go` | Z80: Selects Z80 instruction forms from typed operands. |
| Z80 | Added | `pkg/arch/z80/parser/resolver_diagnostics.go` | Z80: Reports invalid or ambiguous Z80 operand forms. |
| Z80 | Added | `pkg/arch/z80/parser/resolver_extended.go` | Z80: Resolves Z80 extended memory instruction forms. |
| Z80 | Added | `pkg/arch/z80/parser/resolver_indexed.go` | Z80: Resolves Z80 indexed instruction forms. |
| Z80 | Added | `pkg/arch/z80/parser/resolver_indirect.go` | Z80: Resolves Z80 indirect instruction forms. |
| Z80 | Added | `pkg/arch/z80/parser/resolver_port.go` | Z80: Resolves Z80 port I/O instruction forms. |
| Z80 | Added | `pkg/arch/z80/parser/resolver_single_operand.go` | Z80: Resolves one-operand Z80 instructions. |
| Z80 | Added | `pkg/arch/z80/parser/resolver_two_operand.go` | Z80: Resolves two-operand Z80 instructions. |
| Z80 | Added | `pkg/arch/z80/parser/resolver_value.go` | Z80: Resolves immediate and bit value forms. |
| Z80 | Added | `pkg/arch/z80/parser/symbol_rewrite.go` | Z80: Renames symbols inside typed instruction operands. |
| Z80 | Added | `pkg/arch/z80/profile/doc.go` | Z80: Documents this target package. |
| Z80 | Added | `pkg/arch/z80/profile/profile.go` | Z80: Filters Z80 instruction forms by selected profile. |
| Z80 | Added | `pkg/arch/z80/profile/profile_test.go` | Checks Z80 profile behavior. |
| Z80 | Added | `pkg/arch/z80/z80.go` | Registers the Z80 adapter, instruction forms, and target settings. |
| Z80 | Added | `pkg/arch/z80/z80_test.go` | Checks the Z80 adapter. |
| Z80 | Added | `tests/z80/basic.asm` | Covers basic Z80 opcodes. |
| Z80 | Added | `tests/z80/branches.asm` | Covers Z80 branch encoding. |
| Z80 | Added | `tests/z80/branches_overflow.asm` | Covers Z80 branch displacement errors. |
| Z80 | Added | `tests/z80/compatibility.asm` | Covers Z80 syntax compatibility. |
| Z80 | Added | `tests/z80/expressions.asm` | Covers Z80 expression operands. |
| Z80 | Added | `tests/z80/indexed.asm` | Covers indexed Z80 operands. |
| Z80 | Added | `tests/z80/indexed_boundaries.asm` | Covers indexed displacement limits. |
| Z80 | Added | `tests/z80/io_extended.asm` | Covers Z80 I/O and extended opcodes. |
| Z80 | Added | `tests/z80/offsets.asm` | Covers offset expressions. |
| Z80 | Added | `tests/z80/offsets_chained.asm` | Covers chained offsets. |
| Z80 | Added | `tests/z80/profile_gameboy_subset.asm` | Covers accepted Game Boy subset forms. |
| Z80 | Added | `tests/z80/profile_gameboy_subset_rejects.asm` | Covers rejected Game Boy subset forms. |
| Z80 | Added | `tests/z80/profile_strict_documented.asm` | Covers accepted documented forms. |
| Z80 | Added | `tests/z80/profile_strict_documented_rejects.asm` | Covers rejected undocumented forms. |
| x86 | Added | `pkg/arch/x86/assembler/address_assigning_step.go` | x86: Assigns instruction sizes and addresses. |
| x86 | Added | `pkg/arch/x86/assembler/generate_opcode_step.go` | Encodes x86 register, immediate, and direct operands. |
| x86 | Added | `pkg/arch/x86/instruction.go` | Defines x86 registers and instruction opcode tables. |
| x86 | Added | `pkg/arch/x86/parser/instruction.go` | x86: Parses target instruction syntax and selects forms. |
| x86 | Added | `pkg/arch/x86/types.go` | x86: Defines x86 instruction and operand types. |
| x86 | Added | `pkg/arch/x86/x86.go` | Registers the x86 adapter, instruction forms, and target settings. |
| x86 | Added | `pkg/arch/x86/x86_test.go` | Checks the x86 adapter. |
| CLI | Modified | `.gitignore` | Keeps the documentation index and Z80 assembly fixtures visible to Git. |
| CLI | Modified | `cmd/retroasm/architecture.go` | Validates CPU, system, Z80 profile, and compatibility choices; registers target adapters. |
| CLI | Modified | `cmd/retroasm/assemble.go` | Routes Chip-8 through the direct assembler and passes syntax mode and Z80 profile to other adapters. |
| CLI | Modified | `cmd/retroasm/main.go` | Adds compatibility and Z80 profile flags and related log fields. |
| CLI | Modified | `cmd/retroasm/main_test.go` | Checks CLI architecture defaults, validation, and flags. |
| CLI | Added | `cmd/retroasm/z80_fixture_test.go` | Assembles Z80 fixtures and checks bytes, errors, and profile behavior. |
| Library API | Modified | `pkg/retroasm/assembler_test.go` | Checks high-level assembly with registered architecture adapters. |
| Library API | Modified | `pkg/retroasm/default.go` | Uses cpu6502 defaults and emits only used bytes unless fill is configured. |
| Library API | Modified | `pkg/retroasm/doc.go` | Updates public library documentation for registered targets. |
| Library API | Modified | `pkg/retroasm/example_test.go` | Updates library examples for cpu6502. |
| Dependencies | Modified | `go.mod` | Adds an absolute local retrogolib replacement. The Go version and pinned module version already match main. |
| Docs/examples | Modified | `README.md` | Lists supported CPUs, systems, syntax modes, CLI flags, and examples. |
| Docs/examples | Added | `docs/README.md` | Indexes user guides and the 6502 merge plan. States source-branch scope. |
| Docs/examples | Added | `docs/asm6-compatibility.md` | Documents asm6 syntax coverage and limits. |
| Docs/examples | Added | `docs/ca65-compatibility.md` | Documents ca65 syntax coverage and limits. |
| Docs/examples | Added | `docs/compatibility-mode-plan.md` | Records the compatibility mode implementation plan. |
| Docs/examples | Added | `docs/cpu65816-support-plan.md` | Records CPU65816 support and remaining limits. |
| Docs/examples | Added | `docs/cpu68000-support-plan.md` | Records CPU68000 support and remaining limits. |
| Docs/examples | Modified | `docs/library-usage.md` | Documents the high-level API, dispatcher limits, owned codec streams, and public API migrations. |
| Docs/examples | Added | `docs/nesasm-compatibility.md` | Documents NESASM syntax coverage and limits. |
| Docs/examples | Added | `docs/sm83-support-plan.md` | Records SM83 support and remaining limits. |
| Docs/examples | Added | `docs/work-branch-changes.md` | Defines the 6502 merge phases, file changes, dependencies, exclusions, and candidate checks. |
| Docs/examples | Added | `docs/x816-compatibility-plan.md` | Documents x816 syntax and planned coverage. |
| Docs/examples | Added | `docs/z80-branch-changes.md` | Records Z80 implementation history. |
| Docs/examples | Added | `docs/z80-support-plan.md` | Records Z80 support and remaining limits. |
| Docs/examples | Modified | `examples/ast-first/main.go` | Updates AST assembly imports and selectors for cpu6502. |
| Docs/examples | Added | `examples/chip8/README.md` | Explains how to assemble and use the Chip-8 examples. |
| Docs/examples | Added | `examples/chip8/cube.asm` | Provides a Chip-8 cube program example. |
| Docs/examples | Added | `examples/chip8/hello.asm` | Provides a small Chip-8 program example. |

## Merge phases

| Phase | Parts | Changes to merge | Dependency and file rules |
| --- | --- | --- | --- |
| A: Correctness and ownership | P00-P06, including P02a | Baseline inventory, prefixed hex, memory layout, reservation checks, default output length, three-byte numbers, AST copy ownership, and independent data expressions. | P02a follows P02. Keep output-size and public data-field changes separate. See each part's exact file table in the plan. |
| B: Compatibility | P07-P10 | asm6/asm6f, ca65, NESASM labels/macros, and CLI mode selection. | Requires P06. Split parser, directive, CLI, and shared test changes by behavior. |
| C: 6502 and codec APIs | P11-P18 | Package migration, scoped opcode IDs, default 6502 typed operands, owned streams, codec operations, formatting, relocations, and registrations. | Uses Phase A/B APIs. P15/P16 add codec reservation checks after P02a. Adapt tests to 6502. |
| D: Stream changes | P19-P22b | Exact equality, joins, atomic rename, explicit rewrites, and native edits. | Merge P22a before P22b. Keep view invalidation, handle copying, and publication checks together. |
| E: User documentation and audit | P23 | Final usage examples and review of all remaining included changes. | Publish migration notes with the API change that needs them. Keep branch tracking documents on the source branch. |
| Deferred CPU work | Separate proposals | Chip-8, CPU65816, CPU68000, SM83, Z80, x86, and 6502 variants. | Do not include their imports or helpers in the 6502 queue. The plan lists follow-up file groups. |

These are candidate boundaries. No extracted phase has a verified build or
test result from this task. Use the plan's mixed-file ownership table to split
files that contain changes for several parts. Historical commits can combine
features; inspect their complete patches before selecting a commit.

## Verification

- Reviewed: clean initial worktree, local branch refs, merge base, changed-file
  status, source differences, current plan, and newer storage changes.
- Not run: build, lint, and code tests. This task changes documentation only.
  Test files describe intended coverage; they do not prove a passing run.
- Passed: inventory comparison found all 276 scoped files exactly once.
  Local document links and whitespace checks passed.
- Passed: `git diff --check -- CHANGES_SUMMARY.md docs/work-branch-changes.md`.

## Notes

- Local `main` is the merge base. No remote refs were fetched.
- `Makefile`, `go.sum`, and `pkg/assembler/config/ca65_test.go` have no remaining
  difference from local `main`; their old summary entries were removed.
- Rename status is recomputed from the current diff. Source and destination
  paths for each detected rename are listed together.
- The inventory describes the fixed source commit before this refresh.
  Later extractions must use a new target comparison and a progress record.
- No source code or Git history was changed by this documentation task.
