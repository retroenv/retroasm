# Change Summary

## Overview

The clean `work2` branch adds typed assembly streams and codecs, expands target and compatibility support, and changes shared assembly output handling. This summary covers 271 files in `main...HEAD` at `a7b8387`.

## Changes

- **Typed stream and codec:** The AST records source positions, comments, symbols, relocations, target state, and formatting details. It supports exact node comparison, stream joins, symbol renames, and edits that retain source entry metadata. The codec can parse, build, format, validate, and assemble typed streams.
- **Targets:** The branch moves `m6502` to `cpu6502` and adds Chip-8, CPU65816, CPU68000, SM83, Z80, and an x86 library package. The CLI registers the first five added targets; the x86 package is library-only.
- **Syntax and CLI:** Parser and directive changes add asm6, ca65, and NESASM forms alongside existing x816 support. CLI flags select compatibility mode and a Z80 instruction profile.
- **Assembler:** Target byte order controls data output. The assembler records instruction relocations, resolves data lists and forward references, and places bytes at configured segment addresses with bounds checks.
- **Dependency:** `go.mod` requests Go 1.25 and a newer `retrogolib`, but replaces that module with an absolute local path. This checkout depends on that local directory until the replacement is removed.

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
| Typed stream | Modified | `pkg/parser/ast/label.go` | Keeps label comments and edit handles. |
| Typed stream | Modified | `pkg/parser/ast/macro.go` | Keeps macro comments and edit handles. |
| Typed stream | Modified | `pkg/parser/ast/node.go` | Defines node copy, comment, and source entry handle operations. |
| Typed stream | Added | `pkg/parser/ast/node_edit.go` | Commits native node edits through source entry handles. |
| Typed stream | Added | `pkg/parser/ast/node_edit_test.go` | Checks AST node edit behavior. |
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
| Assembler | Modified | `pkg/assembler/address_assigning_step.go` | Starts segments at configured addresses and records instruction relocations. |
| Assembler | Modified | `pkg/assembler/address_assigning_step_test.go` | Checks segment address and reference assignment behavior. |
| Assembler | Modified | `pkg/assembler/assembler.go` | Selects target byte order, tracks source entries, and exposes instruction relocations. |
| Assembler | Modified | `pkg/assembler/assembler_asm6_test.go` | Checks asm6 assembly behavior. |
| Assembler | Modified | `pkg/assembler/assembler_ca65_test.go` | Checks ca65 assembly behavior. |
| Assembler | Modified | `pkg/assembler/assembler_x816_test.go` | Checks x816 assembly behavior. |
| Assembler | Added | `pkg/assembler/banked_output_test.go` | Checks output padding and placement across banks. |
| Assembler | Modified | `pkg/assembler/config/ca65_test.go` | Checks ca65 configuration parsing used by output layout. |
| Assembler | Modified | `pkg/assembler/expression_evaluation_step.go` | Evaluates data item lists with target byte order and defers forward references. |
| Assembler | Modified | `pkg/assembler/generate_opcode_step.go` | Encodes deferred data and references and collects instruction relocations. |
| Assembler | Modified | `pkg/assembler/memory.go` | Writes by absolute memory address and rejects out-of-range writes. |
| Assembler | Added | `pkg/assembler/memory_test.go` | Checks memory bounds and addressed writes. |
| Assembler | Modified | `pkg/assembler/nodes.go` | Carries data item lists, typed opcode IDs, and source entry indices. |
| Assembler | Modified | `pkg/assembler/parse_ast_nodes.go` | Converts typed data and instruction AST nodes for target byte order. |
| Assembler | Modified | `pkg/assembler/parse_ast_nodes_test.go` | Checks AST conversion for data and instruction nodes. |
| Assembler | Modified | `pkg/assembler/process_macros_step.go` | Expands NESASM positional macro parameters. |
| Assembler | Modified | `pkg/assembler/write_output_step.go` | Writes memory buffers with bank padding and address checks. |
| Assembler | Modified | `pkg/number/number.go` | Adds byte order selection for numeric encoding. |
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
| CPU6502 | Added | `pkg/arch/cpu6502/assembler/generate_opcode_step.go` | CPU6502: Encodes instructions and records operand relocations. |
| CPU6502 | Added | `pkg/arch/cpu6502/assembler/generate_opcode_step_test.go` | Checks CPU6502 generate opcode step behavior. |
| CPU6502 | Added | `pkg/arch/cpu6502/assembler/instruction_size.go` | CPU6502: Calculates instruction size from selected addressing. |
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
| CPU6502 | Deleted | `pkg/arch/m6502/assembler/generate_opcode_step.go` | Replaces old opcode generation with the cpu6502 encoder. |
| CPU6502 | Deleted | `pkg/arch/m6502/assembler/generate_opcode_step_test.go` | Replaces old encoder tests with cpu6502 tests. |
| CPU6502 | Deleted | `pkg/arch/m6502/assembler/instruction_size.go` | Moves instruction sizing into the cpu6502 package. |
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
| CLI | Modified | `.gitignore` | Keeps Z80 assembly fixtures in Git while other generated test files stay ignored. |
| CLI | Modified | `cmd/retroasm/architecture.go` | Validates CPU, system, Z80 profile, and compatibility choices; registers target adapters. |
| CLI | Modified | `cmd/retroasm/assemble.go` | Routes Chip-8 through the direct assembler and passes syntax mode and Z80 profile to other adapters. |
| CLI | Modified | `cmd/retroasm/main.go` | Adds compatibility and Z80 profile flags and related log fields. |
| CLI | Modified | `cmd/retroasm/main_test.go` | Checks CLI architecture defaults, validation, and flags. |
| CLI | Added | `cmd/retroasm/z80_fixture_test.go` | Assembles Z80 fixtures and checks bytes, errors, and profile behavior. |
| Library API | Modified | `pkg/retroasm/assembler_test.go` | Checks high-level assembly with registered architecture adapters. |
| Library API | Modified | `pkg/retroasm/default.go` | Uses cpu6502 defaults and emits only used bytes unless fill is configured. |
| Library API | Modified | `pkg/retroasm/doc.go` | Updates public library documentation for registered targets. |
| Library API | Modified | `pkg/retroasm/example_test.go` | Updates library examples for cpu6502. |
| Dependencies | Modified | `Makefile` | Updates the golangci-lint version. |
| Dependencies | Modified | `go.mod` | Moves to Go 1.25 and a newer retrogolib version; adds a local replacement path. |
| Dependencies | Modified | `go.sum` | Removes checksums for earlier retrogolib versions. |
| Docs/examples | Modified | `README.md` | Lists supported CPUs, systems, syntax modes, CLI flags, and examples. |
| Docs/examples | Added | `docs/asm6-compatibility.md` | Documents asm6 syntax coverage and limits. |
| Docs/examples | Added | `docs/ca65-compatibility.md` | Documents ca65 syntax coverage and limits. |
| Docs/examples | Added | `docs/compatibility-mode-plan.md` | Records the compatibility mode implementation plan. |
| Docs/examples | Added | `docs/cpu65816-support-plan.md` | Records CPU65816 support and remaining limits. |
| Docs/examples | Added | `docs/cpu68000-support-plan.md` | Records CPU68000 support and remaining limits. |
| Docs/examples | Modified | `docs/library-usage.md` | Changes library examples to the cpu6502 package and identifiers. |
| Docs/examples | Added | `docs/nesasm-compatibility.md` | Documents NESASM syntax coverage and limits. |
| Docs/examples | Added | `docs/sm83-support-plan.md` | Records SM83 support and remaining limits. |
| Docs/examples | Added | `docs/work-branch-changes.md` | Records an older branch extraction plan; its snapshot is dated. |
| Docs/examples | Added | `docs/x816-compatibility-plan.md` | Documents x816 syntax and planned coverage. |
| Docs/examples | Added | `docs/z80-branch-changes.md` | Records Z80 implementation history. |
| Docs/examples | Added | `docs/z80-support-plan.md` | Records Z80 support and remaining limits. |
| Docs/examples | Modified | `examples/ast-first/main.go` | Updates AST assembly imports and selectors for cpu6502. |
| Docs/examples | Added | `examples/chip8/README.md` | Explains how to assemble and use the Chip-8 examples. |
| Docs/examples | Added | `examples/chip8/cube.asm` | Provides a Chip-8 cube program example. |
| Docs/examples | Added | `examples/chip8/hello.asm` | Provides a small Chip-8 program example. |

## Possible commit slices

- **Typed stream foundation:** AST node ownership, metadata, equality, stream editing, symbol rewrites, and tests. The codec and parser stream entry points depend on these types. The `stream-symbol-renames`, `source-entry-output`, and `native-entry-handles` branches provide commit context for the later stream changes; all are already ancestors of this branch.
- **Codec and relocation contract:** Target operand builders and formatters, shared codec behavior, architecture registration, and relocation recording. This depends on the typed stream foundation. `pkg/arch/arch.go`, `pkg/assembler/assembler.go`, and the target parser and encoder files cross this boundary and need hunk-level review.
- **Architecture packages:** CPU6502 migration, Chip-8, CPU65816, CPU68000, SM83, Z80, and x86 can be reviewed by target with each target's tests. Shared parser, assembler, and codec files cross target boundaries and need hunk-level separation. The x86 package has no CLI registration.
- **Syntax modes and CLI:** asm6, ca65, and NESASM parser and directive changes can be reviewed by dialect. `pkg/parser/parser.go`, `pkg/parser/directives/directives.go`, and the CLI files need hunk-level separation. The CLI target registrations depend on the target packages.
- **Output layout and documentation:** Segment placement and bank padding can be reviewed with assembler tests. Update README and user guides only after their public paths work in the destination branch. The older `docs/work-branch-changes.md` is branch tracking material.
- **Dependency prerequisite:** Replace the absolute `retrogolib` path with a portable dependency before a merge candidate. Each proposed boundary needs its own build and test results before it can be treated as independent.

## Verification

- Passed: `git diff --stat main...HEAD`, `git diff --name-status main...HEAD`, and component diff inspection — confirmed the scoped file set and behavior claims.
- Not run: build, lint, and tests — this documentation task does not change build inputs; no current result is claimed for the branch.

## Notes

- The worktree had no staged, unstaged, or untracked files before this summary. The comparison base is local `main` at `bf78324`.
- The branch references above give provenance. They do not add files outside `main...HEAD` to this summary.
- `docs/work-branch-changes.md` contains an August snapshot with older counts and prerequisites. Use the current diff for extraction decisions.
