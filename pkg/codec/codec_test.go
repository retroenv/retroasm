package codec_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	asmchip8 "github.com/retroenv/retroasm/pkg/arch/chip8"
	asmcpu6502 "github.com/retroenv/retroasm/pkg/arch/cpu6502"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/arch/cpu/cpu6502"
	"github.com/retroenv/retrogolib/assert"
)

func TestNew_RejectsIncompleteConfiguration(t *testing.T) {
	t.Parallel()

	_, err := codec.New[*cpu6502.Instruction](nil)
	assert.ErrorIs(t, err, codec.ErrNilConfiguration)

	_, err = codec.New(&config.Config[*cpu6502.Instruction]{})
	assert.ErrorIs(t, err, codec.ErrNilArchitecture)
}

func TestCodec_ParseTypedStream(t *testing.T) {
	t.Parallel()

	c := newCPU6502Codec(t)
	stream, err := c.ParseStream(
		t.Context(),
		"input.asm",
		strings.NewReader("start:\n; before load\nlda #1 ; inline"),
	)
	assert.NoError(t, err)
	assert.Equal(t, 3, stream.Len())
	assert.Equal(t, ast.SourcePosition{Source: "input.asm", Line: 1, Column: 1}, stream.At(0).Position)
	assert.Equal(t, ast.SourcePosition{Source: "input.asm", Line: 2, Column: 1}, stream.At(1).Position)
	assert.Equal(t, ast.SourcePosition{Source: "input.asm", Line: 3, Column: 1}, stream.At(2).Position)
	nodes := stream.Nodes()
	assert.Len(t, nodes, 3)

	label, ok := nodes[0].(ast.Label)
	assert.True(t, ok)
	assert.Equal(t, "start", label.Name)
	_, ok = nodes[1].(*ast.Comment)
	assert.True(t, ok)
	instruction, ok := ast.InstructionFromNode(nodes[2])
	assert.True(t, ok)
	assert.True(t, instruction.OpcodeID.ValidFor(arch.CPU6502))
	assert.Equal(t, uint16(cpu6502.Lda), instruction.OpcodeID.Value)
}

func TestCodec_ParseInstruction(t *testing.T) {
	t.Parallel()

	c := newCPU6502Codec(t)
	instruction, err := c.ParseInstruction(t.Context(), strings.NewReader("lda #1"))
	assert.NoError(t, err)
	assert.Equal(t, cpu6502.LdaName, instruction.Name)
	assert.True(t, instruction.OpcodeID.ValidFor(arch.CPU6502))

	_, err = c.ParseInstruction(t.Context(), strings.NewReader("label:"))
	assert.ErrorIs(t, err, codec.ErrExpectedInstruction)
	_, err = c.ParseInstruction(t.Context(), strings.NewReader("lda #1\nsta $20"))
	assert.ErrorIs(t, err, codec.ErrExpectedInstruction)
	_, err = c.ParseInstruction(t.Context(), nil)
	assert.ErrorIs(t, err, codec.ErrNilSource)
}

func TestCodec_OpcodeID(t *testing.T) {
	t.Parallel()

	c := newCPU6502Codec(t)
	identity, err := c.OpcodeID(" LDA ")
	assert.NoError(t, err)
	assert.True(t, identity.ValidFor(arch.CPU6502))
	assert.Equal(t, uint16(cpu6502.Lda), identity.Value)

	_, err = c.OpcodeID("missing")
	assert.ErrorIs(t, err, codec.ErrUnknownInstruction)
}

func TestCodec_AssembleTypedStreamWithoutReparsing(t *testing.T) {
	t.Parallel()

	c, err := codec.New(asmchip8.New())
	assert.NoError(t, err)
	stream, err := c.ParseStream(t.Context(), "input.asm", strings.NewReader("entry:\ncls"))
	assert.NoError(t, err)

	result, err := c.AssembleStream(t.Context(), stream)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x00, 0xe0}, result.Binary)
	assert.Equal(t, uint64(0x200), result.Symbols["entry"])

	_, err = c.AssembleStream(t.Context(), nil)
	assert.ErrorIs(t, err, codec.ErrNilStream)
}

func TestCodec_ValidateStreamReportsInstructionPosition(t *testing.T) {
	t.Parallel()

	c := newCPU6502Codec(t)
	stream, err := c.ParseStream(t.Context(), "input.asm", strings.NewReader("entry:\nlda #1"))
	assert.NoError(t, err)
	assert.NoError(t, c.ValidateStream(stream))

	stream.Replace(1, 2, []ast.Entry{ast.NewEntry(
		ast.NewInstruction("missing", 0, nil, nil),
		ast.SourcePosition{Source: "input.asm", Line: 2, Column: 1},
	)})
	err = c.ValidateStream(stream)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "input.asm:2:1")
	assert.ErrorContains(t, err, "unknown CPU6502 instruction")

	err = c.ValidateStream(nil)
	assert.ErrorIs(t, err, codec.ErrNilStream)
}

func TestCodec_ValidateStreamReportsDataPosition(t *testing.T) {
	t.Parallel()

	c := newCPU6502Codec(t)
	data := ast.NewData(ast.DataType, 1)
	stream := ast.NewStream(ast.NewEntry(
		data,
		ast.SourcePosition{Source: "input.asm", Line: 6, Column: 3},
	))

	err := c.ValidateStream(stream)
	assert.ErrorIs(t, err, ast.ErrInvalidData)
	assert.ErrorContains(t, err, "input.asm:6:3")
}

func TestCodec_FormatStreamRoundTripsLabelsCommentsAndInstructions(t *testing.T) {
	t.Parallel()

	c := newCPU6502AssemblyCodec(t)
	stream, err := c.ParseStream(
		t.Context(),
		"input.asm",
		strings.NewReader("; prologue\nentry: ; target\nlda #1 ; load"),
	)
	assert.NoError(t, err)

	formatted, err := c.FormatStream(stream)
	assert.NoError(t, err)
	assert.Equal(t, "; prologue\nentry: ; target\nlda #$01 ; load", formatted)

	roundTripped, err := c.ParseStream(t.Context(), "formatted.asm", strings.NewReader(formatted))
	assert.NoError(t, err)
	assert.NoError(t, c.ValidateStream(roundTripped))
	assert.Equal(t, stream.Len(), roundTripped.Len())
	assert.Equal(t, "prologue", roundTripped.At(0).Node.(*ast.Comment).Message)
	assert.Equal(t, "target", ast.InlineComment(roundTripped.At(1).Node))
	assert.Equal(t, "load", ast.InlineComment(roundTripped.At(2).Node))

	originalAssembly, err := c.AssembleStream(t.Context(), stream)
	assert.NoError(t, err)
	roundTripAssembly, err := c.AssembleStream(t.Context(), roundTripped)
	assert.NoError(t, err)
	assert.Equal(t, originalAssembly.Binary, roundTripAssembly.Binary)
	assert.Equal(t, originalAssembly.Symbols, roundTripAssembly.Symbols)
}

func TestCodec_FormatStreamRoundTripsData(t *testing.T) {
	t.Parallel()

	c := newCPU6502AssemblyCodec(t)
	stream, err := c.ParseStream(
		t.Context(),
		"input.asm",
		strings.NewReader(".byte 1+2,3*4,\"A\" ; values\n.dsb 5,1,2\n.addr target\ntarget:\n.byte 0"),
	)
	assert.NoError(t, err)

	formatted, err := c.FormatStream(stream)
	assert.NoError(t, err)
	assert.Equal(t, ".byte 0x1+0x2, 0x3*0x4, \"A\" ; values\n.dsb 0x5, 0x1, 0x2\n.addr target\ntarget:\n.byte 0x0", formatted)

	roundTripped, err := c.ParseStream(t.Context(), "formatted.asm", strings.NewReader(formatted))
	assert.NoError(t, err)
	assert.NoError(t, c.ValidateStream(roundTripped))
	assert.Equal(t, stream.Len(), roundTripped.Len())
	assert.Equal(t, "values", ast.InlineComment(roundTripped.At(0).Node))

	for _, index := range []int{0, 1, 2, 4} {
		originalData, ok := stream.At(index).Node.(ast.Data)
		assert.True(t, ok)
		roundTripData, ok := roundTripped.At(index).Node.(ast.Data)
		assert.True(t, ok)
		assert.Equal(t, originalData.Type, roundTripData.Type)
		assert.Equal(t, originalData.Width, roundTripData.Width)
		assert.Equal(t, originalData.ReferenceType, roundTripData.ReferenceType)
		assert.Equal(t, originalData.Fill, roundTripData.Fill)
		assert.Len(t, roundTripData.Values, len(originalData.Values))
	}

	originalAssembly, err := c.AssembleStream(t.Context(), stream)
	assert.NoError(t, err)
	roundTripAssembly, err := c.AssembleStream(t.Context(), roundTripped)
	assert.NoError(t, err)
	assert.Equal(t, []byte{3, 12, 'A', 1, 2, 1, 2, 1, 10, 0, 0}, originalAssembly.Binary)
	assert.Equal(t, originalAssembly.Binary, roundTripAssembly.Binary)
	assert.Equal(t, originalAssembly.Symbols, roundTripAssembly.Symbols)
}

func TestCodec_FormatStreamRoundTripsCa65AddressData(t *testing.T) {
	t.Parallel()

	configuration := asmcpu6502.New()
	configuration.CompatibilityMode = config.CompatCa65
	c := newCPU6502AssemblyCodecWithConfig(t, configuration)
	stream, err := c.ParseStream(
		t.Context(),
		"input.asm",
		strings.NewReader(".faraddr target\n.lobytes target\n.hibytes target\n.bankbytes target\ntarget:\n.byte 0"),
	)
	assert.NoError(t, err)

	formatted, err := c.FormatStream(stream)
	assert.NoError(t, err)
	assert.Equal(t, ".faraddr target\n.dl target\n.dh target\n.bankbytes target\ntarget:\n.byte 0x0", formatted)

	roundTripped, err := c.ParseStream(t.Context(), "formatted.asm", strings.NewReader(formatted))
	assert.NoError(t, err)
	originalAssembly, err := c.AssembleStream(t.Context(), stream)
	assert.NoError(t, err)
	roundTripAssembly, err := c.AssembleStream(t.Context(), roundTripped)
	assert.NoError(t, err)
	assert.Equal(t, []byte{6, 0, 0, 6, 0, 0, 0}, originalAssembly.Binary)
	assert.Equal(t, originalAssembly.Binary, roundTripAssembly.Binary)
	assert.Equal(t, originalAssembly.Symbols, roundTripAssembly.Symbols)
}

func TestCodec_FormatStreamRoundTripsSymbolAndLayoutDirectives(t *testing.T) {
	t.Parallel()

	c := newCPU6502AssemblyCodec(t)
	stream, err := c.ParseStream(
		t.Context(),
		"input.asm",
		strings.NewReader("reusable = 1+2 ; alias\ndynamic EQU reusable+1\n.org $8000\n.bank 2\n.segment code\n.rsset 16\nscratch .rs 4\n.res 3\n.inesmap 4\n.inessubmap 2\n.inesprg 2\n.ineschr 3\n.inesbat 1\n.inesmir 0\n.fillvalue $aa"),
	)
	assert.NoError(t, err)

	formatted, err := c.FormatStream(stream)
	assert.NoError(t, err)
	assert.Equal(
		t,
		"reusable = 0x1+0x2 ; alias\ndynamic EQU reusable+0x1\n.org 0x8000\n.bank 2\n.segment code\n.rsset 16\nscratch .rs 4\n.res 3\n.inesmap 4\n.inessubmap 2\n.inesprg 2\n.ineschr 3\n.inesbat 1\n.inesmir 0\n.fillvalue 0xAA",
		formatted,
	)

	roundTripped, err := c.ParseStream(t.Context(), "formatted.asm", strings.NewReader(formatted))
	assert.NoError(t, err)
	assert.Equal(t, stream.Len(), roundTripped.Len())

	for index := range stream.Len() {
		assert.Equal(t, layoutNodeSignature(t, stream.At(index).Node), layoutNodeSignature(t, roundTripped.At(index).Node))
	}
}

func TestCodec_FormatStreamRoundTripsAsm6ConfigurationDirectives(t *testing.T) {
	t.Parallel()

	configuration := asmcpu6502.New()
	configuration.CompatibilityMode = config.CompatAsm6
	c := newCPU6502AssemblyCodecWithConfig(t, configuration)
	stream, err := c.ParseStream(
		t.Context(),
		"input.asm",
		strings.NewReader(".nes2chrram 1\n.nes2prgram 2\n.nes2sub 3\n.nes2tv 4\n.nes2vs 5\n.nes2bram 6\n.nes2chrbram 7"),
	)
	assert.NoError(t, err)

	formatted, err := c.FormatStream(stream)
	assert.NoError(t, err)
	roundTripped, err := c.ParseStream(t.Context(), "formatted.asm", strings.NewReader(formatted))
	assert.NoError(t, err)
	assert.Equal(t, stream.Len(), roundTripped.Len())

	for index := range stream.Len() {
		assert.Equal(t, layoutNodeSignature(t, stream.At(index).Node), layoutNodeSignature(t, roundTripped.At(index).Node))
	}
}

func TestCodec_FormatStreamPreservesLayoutAssembly(t *testing.T) {
	t.Parallel()

	c := newCPU6502AssemblyCodec(t)
	stream, err := c.ParseStream(
		t.Context(),
		"input.asm",
		strings.NewReader("value = 1\n.segment code\n.org 2\nentry:\n.byte value"),
	)
	assert.NoError(t, err)
	formatted, err := c.FormatStream(stream)
	assert.NoError(t, err)
	roundTripped, err := c.ParseStream(t.Context(), "formatted.asm", strings.NewReader(formatted))
	assert.NoError(t, err)

	originalAssembly, err := c.AssembleStream(t.Context(), stream)
	assert.NoError(t, err)
	roundTripAssembly, err := c.AssembleStream(t.Context(), roundTripped)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0, 0, 1}, originalAssembly.Binary)
	assert.Equal(t, originalAssembly.Binary, roundTripAssembly.Binary)
	assert.Equal(t, originalAssembly.Symbols, roundTripAssembly.Symbols)
}

func TestCodec_FormatStreamRoundTripsStructuralDirectives(t *testing.T) {
	t.Parallel()

	configuration := asmcpu6502.New()
	configuration.CompatibilityMode = config.CompatCa65
	c := newCPU6502AssemblyCodecWithConfig(t, configuration)
	stream, err := c.ParseStream(
		t.Context(),
		"input.asm",
		strings.NewReader(".scope outer ; scope\n.proc work\n.enum 2+3\n.ende\n.rept 2\n.if 1\n.elseif 0\n.else\n.endif\n.ifdef feature\n.endif\n.ifndef feature\n.endif\n.endr\n.endproc\n.endscope\n.include \"defs.asm\"\n.incbin \"data.bin\", 2, 4\n.error \"stop now\""),
	)
	assert.NoError(t, err)

	formatted, err := c.FormatStream(stream)
	assert.NoError(t, err)
	roundTripped, err := c.ParseStream(t.Context(), "formatted.asm", strings.NewReader(formatted))
	assert.NoError(t, err)
	assert.Equal(t, stream.Len(), roundTripped.Len())

	for index := range stream.Len() {
		assert.Equal(t, directiveNodeSignature(t, stream.At(index).Node), directiveNodeSignature(t, roundTripped.At(index).Node))
	}

	formattedAgain, err := c.FormatStream(roundTripped)
	assert.NoError(t, err)
	assert.Equal(t, formatted, formattedAgain)
}

func TestCodec_FormatStreamRoundTripsMacros(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mode   config.CompatibilityMode
		source string
	}{
		{
			name:   "named arguments",
			mode:   config.CompatDefault,
			source: ".macro transfer source, target\nlda source ; load\nsta target\n.endm ; definition",
		},
		{
			name:   "nesasm positional arguments",
			mode:   config.CompatNesasm,
			source: "add_val .macro\nclc\nadc \\1\n.endm",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			configuration := asmcpu6502.New()
			configuration.CompatibilityMode = test.mode
			c := newCPU6502AssemblyCodecWithConfig(t, configuration)
			stream, err := c.ParseStream(t.Context(), "input.asm", strings.NewReader(test.source))
			assert.NoError(t, err)

			formatted, err := c.FormatStream(stream)
			assert.NoError(t, err)
			roundTripped, err := c.ParseStream(t.Context(), "formatted.asm", strings.NewReader(formatted))
			assert.NoError(t, err)
			assert.Equal(t, stream.Len(), roundTripped.Len())
			assert.Equal(t, directiveNodeSignature(t, stream.At(0).Node), directiveNodeSignature(t, roundTripped.At(0).Node))
		})
	}
}

func TestCodec_FormatStreamRejectsUnrepresentableDirectiveNodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		node ast.Node
	}{
		{name: "function name", node: ast.NewFunction("")},
		{name: "condition identifier", node: ast.NewIfdef("not valid")},
		{name: "include name", node: ast.NewInclude("missing_extension", false, 0, 0)},
		{name: "include range", node: ast.NewInclude("\"data.bin\"", true, -1, 0)},
		{name: "error message", node: ast.NewError("two\nlines")},
		{name: "macro boundaries", node: ast.NewMacro("empty")},
	}

	c := newCPU6502Codec(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := c.FormatStream(ast.NewStreamFromNodes(test.node))
			assert.ErrorIs(t, err, codec.ErrFormattingUnsupported)
		})
	}
}

func TestCodec_FormatStreamRejectsUnrepresentableLayoutNodes(t *testing.T) {
	t.Parallel()

	alias := ast.NewAlias("value")
	alias.SymbolReusable = true
	invalidVariable := ast.NewVariable("", 1)
	invalidVariable.UseOffsetCounter = true
	invalidConfiguration := ast.NewConfiguration(ast.ConfigPrg)
	invalidConfiguration.Value = 1
	asm6Configuration := ast.NewConfiguration(ast.ConfigNes2ChrRAM)

	tests := []struct {
		name string
		node ast.Node
	}{
		{name: "alias policy", node: alias},
		{name: "negative bank", node: ast.NewBank(-1)},
		{name: "named reservation", node: ast.NewVariable("named", 1)},
		{name: "anonymous offset variable", node: invalidVariable},
		{name: "configuration value", node: invalidConfiguration},
		{name: "mode-specific configuration", node: asm6Configuration},
		{name: "invalid configuration item", node: ast.NewConfiguration(ast.ConfigInvalid)},
	}

	c := newCPU6502Codec(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := c.FormatStream(ast.NewStreamFromNodes(test.node))
			assert.ErrorIs(t, err, codec.ErrFormattingUnsupported)
		})
	}
}

func TestCodec_FormatStreamRejectsUnsupportedNodes(t *testing.T) {
	t.Parallel()

	c := newCPU6502Codec(t)
	stream := ast.NewStream(ast.NewEntry(
		ast.NewScope("inner"),
		ast.SourcePosition{Source: "input.asm", Line: 4, Column: 1},
	))

	_, err := c.FormatStream(stream)
	assert.ErrorIs(t, err, codec.ErrFormattingUnsupported)
	assert.ErrorContains(t, err, "input.asm:4:1")

	_, err = c.FormatStream(nil)
	assert.ErrorIs(t, err, codec.ErrNilStream)
}

func TestCodec_ParseHonorsCancellation(t *testing.T) {
	t.Parallel()

	c := newCPU6502Codec(t)
	ctx := t.Context()
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	_, err := c.Parse(cancelled, strings.NewReader("lda #1"))
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func newCPU6502Codec(t *testing.T) *codec.Codec[*cpu6502.Instruction] {
	t.Helper()
	c, err := codec.New(asmcpu6502.New())
	assert.NoError(t, err)
	return c
}

func layoutNodeSignature(t *testing.T, node ast.Node) string {
	t.Helper()

	comment := ast.InlineComment(node)
	switch typed := node.(type) {
	case ast.Alias:
		value, err := ast.FormatExpression(typed.Expression)
		assert.NoError(t, err)
		return fmt.Sprintf("alias:%s:%t:%t:%s:%s", typed.Name, typed.SymbolReusable, typed.Expression.IsEvaluatedOnce(), value, comment)
	case ast.Base:
		value, err := ast.FormatExpression(typed.Address)
		assert.NoError(t, err)
		return fmt.Sprintf("base:%s:%s", value, comment)
	case ast.Bank:
		return fmt.Sprintf("bank:%d:%s", typed.Number, comment)
	case ast.Segment:
		return fmt.Sprintf("segment:%s:%s", typed.Name, comment)
	case ast.OffsetCounter:
		return fmt.Sprintf("offset:%d:%s", typed.Number, comment)
	case ast.Variable:
		return fmt.Sprintf("variable:%s:%d:%t:%s", typed.Name, typed.Size, typed.UseOffsetCounter, comment)
	case ast.Configuration:
		value := ""
		if typed.Expression != nil {
			var err error
			value, err = ast.FormatExpression(typed.Expression)
			assert.NoError(t, err)
		}
		return fmt.Sprintf("configuration:%d:%d:%s:%s", typed.Item, typed.Value, value, comment)
	default:
		return fmt.Sprintf("%T:%s", node, comment)
	}
}

func directiveNodeSignature(t *testing.T, node ast.Node) string {
	t.Helper()

	comment := ast.InlineComment(node)
	if signature, ok := blockNodeSignature(t, node, comment); ok {
		return signature
	}
	if signature, ok := conditionNodeSignature(t, node, comment); ok {
		return signature
	}
	return sourceNodeSignature(t, node, comment)
}

func blockNodeSignature(t *testing.T, node ast.Node, comment string) (string, bool) {
	t.Helper()

	switch typed := node.(type) {
	case ast.Scope:
		return fmt.Sprintf("scope:%s:%s", typed.Name, comment), true
	case ast.ScopeEnd:
		return "scope-end:" + comment, true
	case ast.Function:
		return fmt.Sprintf("function:%s:%s", typed.Name, comment), true
	case ast.FunctionEnd:
		return "function-end:" + comment, true
	case ast.Enum:
		return expressionNodeSignature(t, "enum", typed.Address, comment), true
	case ast.EnumEnd:
		return "enum-end:" + comment, true
	case ast.Rept:
		return expressionNodeSignature(t, "rept", typed.Count, comment), true
	case ast.Endr:
		return "rept-end:" + comment, true
	default:
		return "", false
	}
}

func conditionNodeSignature(t *testing.T, node ast.Node, comment string) (string, bool) {
	t.Helper()

	switch typed := node.(type) {
	case ast.If:
		return expressionNodeSignature(t, "if", typed.Condition, comment), true
	case ast.Ifdef:
		return fmt.Sprintf("ifdef:%s:%s", typed.Identifier, comment), true
	case ast.Ifndef:
		return fmt.Sprintf("ifndef:%s:%s", typed.Identifier, comment), true
	case ast.Else:
		return "else:" + comment, true
	case ast.ElseIf:
		return expressionNodeSignature(t, "elseif", typed.Condition, comment), true
	case ast.Endif:
		return "endif:" + comment, true
	default:
		return "", false
	}
}

func sourceNodeSignature(t *testing.T, node ast.Node, comment string) string {
	t.Helper()

	switch typed := node.(type) {
	case ast.Include:
		return fmt.Sprintf("include:%s:%t:%d:%d:%s", typed.Name, typed.Binary, typed.Start, typed.Size, comment)
	case ast.Macro:
		tokens := make([]string, len(typed.Token))
		for index, bodyToken := range typed.Token {
			tokens[index] = fmt.Sprintf("%d:%s", bodyToken.Type, bodyToken.Value)
		}
		return fmt.Sprintf("macro:%s:%v:%s:%s", typed.Name, typed.Arguments, strings.Join(tokens, ","), comment)
	case ast.Error:
		return fmt.Sprintf("error:%s:%s", typed.Message, comment)
	default:
		return layoutNodeSignature(t, node)
	}
}

func expressionNodeSignature(t *testing.T, kind string, value *expression.Expression, comment string) string {
	t.Helper()
	formatted, err := ast.FormatExpression(value)
	assert.NoError(t, err)
	return fmt.Sprintf("%s:%s:%s", kind, formatted, comment)
}
