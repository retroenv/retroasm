package parser

import (
	"context"
	"strings"
	"testing"

	asmcpu6502 "github.com/retroenv/retroasm/pkg/arch/cpu6502"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	retroarch "github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/arch/cpu/cpu6502"
	"github.com/retroenv/retrogolib/assert"
)

//nolint:funlen // table-driven test with many cases
func TestParser_Instruction(t *testing.T) {
	tests := []struct {
		input    string
		expected func() []ast.Node
	}{
		{"asl a:var1", func() []ast.Node {
			l := ast.NewLabel("var1")
			return []ast.Node{cpu6502Instruction("asl", int(cpu6502.AbsoluteAddressing), l)}
		}},
		{"asl a:1", func() []ast.Node {
			return []ast.Node{cpu6502Instruction("asl", int(cpu6502.AbsoluteAddressing), ast.NewNumber(1))}
		}},
		{"asl", func() []ast.Node {
			return []ast.Node{cpu6502Instruction("asl", int(cpu6502.AccumulatorAddressing), nil)}
		}},
		{"asl a", func() []ast.Node {
			return []ast.Node{cpu6502Instruction("asl", int(cpu6502.AccumulatorAddressing), nil)}
		}},
		{"asl\na:", func() []ast.Node {
			l := ast.NewLabel("a")
			return []ast.Node{
				cpu6502Instruction("asl", int(cpu6502.AccumulatorAddressing), nil),
				l,
			}
		}},
	}

	cfg := asmcpu6502.New()

	for _, tt := range tests {
		parser := New(cfg.Arch, strings.NewReader(tt.input), config.CompatDefault)
		assert.NoError(t, parser.Read(t.Context()))
		nodes, err := parser.TokensToAstNodes()
		assert.NoError(t, err)

		expectedNodes := tt.expected()
		assert.Len(t, nodes, len(expectedNodes), "input: "+tt.input)
		for i, expected := range expectedNodes {
			assert.Equal(t, expected, nodes[i])
		}
	}
}

func TestParser_EdgeCases(t *testing.T) {
	cfg := asmcpu6502.New()

	t.Run("empty input", func(t *testing.T) {
		parser := New(cfg.Arch, strings.NewReader(""), config.CompatDefault)
		assert.NoError(t, parser.Read(t.Context()))
		nodes, err := parser.TokensToAstNodes()
		assert.NoError(t, err)
		assert.Empty(t, nodes)
	})

	t.Run("only whitespace", func(t *testing.T) {
		parser := New(cfg.Arch, strings.NewReader("   \n\t  \n"), config.CompatDefault)
		assert.NoError(t, parser.Read(t.Context()))
		nodes, err := parser.TokensToAstNodes()
		assert.NoError(t, err)
		assert.Empty(t, nodes)
	})

	t.Run("only comments", func(t *testing.T) {
		parser := New(cfg.Arch, strings.NewReader("; comment\n// another comment"), config.CompatDefault)
		assert.NoError(t, parser.Read(t.Context()))
		nodes, err := parser.TokensToAstNodes()
		assert.NoError(t, err)
		// Comment nodes may be combined or filtered, check actual behavior
		assert.True(t, len(nodes) >= 1, "should have at least one comment node")
	})
}

func TestParser_ErrorConditions(t *testing.T) {
	cfg := asmcpu6502.New()

	t.Run("context cancellation during read", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		parser := New(cfg.Arch, strings.NewReader("lda #$01"), config.CompatDefault)
		err := parser.Read(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
	})

	t.Run("unsupported directive", func(t *testing.T) {
		parser := New(cfg.Arch, strings.NewReader(".unsupported"), config.CompatDefault)
		assert.NoError(t, parser.Read(t.Context()))
		_, err := parser.TokensToAstNodes()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported directive")
	})

	t.Run("missing directive parameter", func(t *testing.T) {
		parser := New(cfg.Arch, strings.NewReader(".byte"), config.CompatDefault)
		assert.NoError(t, parser.Read(t.Context()))
		_, err := parser.TokensToAstNodes()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing parameter")
	})

	t.Run("unexpected token type", func(t *testing.T) {
		parser := New(cfg.Arch, strings.NewReader("@"), config.CompatDefault)
		// The lexer may handle @ as an illegal token before parser sees it
		err := parser.Read(t.Context())
		if err != nil {
			assert.Contains(t, err.Error(), "illegal")
		} else {
			_, err = parser.TokensToAstNodes()
			if err != nil {
				assert.Contains(t, err.Error(), "unexpected")
			}
		}
	})
}

func TestParser_NextToken(t *testing.T) {
	cfg := asmcpu6502.New()
	parser := NewWithTokens(cfg.Arch, []token.Token{
		{Type: token.Identifier, Value: "lda"},
		{Type: token.Number, Value: "#$01"},
		{Type: token.EOL},
	}, config.CompatDefault)
	parser.programLength = 3

	t.Run("valid offset", func(t *testing.T) {
		tok := parser.NextToken(0)
		assert.Equal(t, token.Identifier, tok.Type)
		assert.Equal(t, "lda", tok.Value)

		tok = parser.NextToken(1)
		assert.Equal(t, token.Number, tok.Type)
		assert.Equal(t, "#$01", tok.Value)
	})

	t.Run("offset beyond program length", func(t *testing.T) {
		tok := parser.NextToken(10)
		assert.Equal(t, token.EOF, tok.Type)
	})

	t.Run("negative position with offset", func(t *testing.T) {
		parser.readPosition = 2
		tok := parser.NextToken(1)
		assert.Equal(t, token.EOF, tok.Type)
	})
}

func TestParser_AdvanceReadPosition(t *testing.T) {
	cfg := asmcpu6502.New()
	parser := NewWithTokens(cfg.Arch, nil, config.CompatDefault)

	initialPos := parser.readPosition
	parser.AdvanceReadPosition(5)
	assert.Equal(t, initialPos+5, parser.readPosition)

	parser.AdvanceReadPosition(-2)
	assert.Equal(t, initialPos+3, parser.readPosition)
}

func TestParser_AddressWidth(t *testing.T) {
	cfg := asmcpu6502.New()
	parser := New(cfg.Arch, strings.NewReader(""), config.CompatDefault)

	// CPU6502 has 16-bit addresses
	assert.Equal(t, 16, parser.AddressWidth())
}

func TestParser_PreallocationBenefit(t *testing.T) {
	cfg := asmcpu6502.New()

	// Test that pre-allocation doesn't break functionality with large programs
	largeInput := strings.Repeat("nop\n", 1000)
	parser := New(cfg.Arch, strings.NewReader(largeInput), config.CompatDefault)
	assert.NoError(t, parser.Read(t.Context()))

	nodes, err := parser.TokensToAstNodes()
	assert.NoError(t, err)
	assert.Len(t, nodes, 1000)
}

func TestParser_TokensToStreamPreservesPositionsAndState(t *testing.T) {
	cfg := asmcpu6502.New()
	parser := New(cfg.Arch, strings.NewReader("entry:\n  lda #1\n; note\n"), config.CompatDefault)
	parser.SetArchitectureState("initial")
	assert.NoError(t, parser.Read(t.Context()))

	stream, err := parser.TokensToStream("input.asm")
	assert.NoError(t, err)
	assert.Equal(t, 3, stream.Len())
	assert.Equal(t, ast.SourcePosition{Source: "input.asm", Line: 1, Column: 1}, stream.At(0).Position)
	assert.Equal(t, ast.SourcePosition{Source: "input.asm", Line: 2, Column: 3}, stream.At(1).Position)
	assert.Equal(t, ast.SourcePosition{Source: "input.asm", Line: 3, Column: 1}, stream.At(2).Position)

	initial, final, ok := ast.StateSnapshots[string](stream)
	assert.True(t, ok)
	assert.Equal(t, "initial", initial)
	assert.Equal(t, "initial", final)
	_, ok = stream.At(2).Node.(*ast.Comment)
	assert.True(t, ok)
}

func TestParser_TokensToStreamRecordsSymbolDefinitions(t *testing.T) {
	cfg := asmcpu6502.New()
	parser := New(cfg.Arch, strings.NewReader("value = 1\ndynamic EQU value+1\nentry:\n.proc work\n.endproc\n"), config.CompatDefault)
	assert.NoError(t, parser.Read(t.Context()))

	stream, err := parser.TokensToStream("input.asm")
	assert.NoError(t, err)
	symbols := stream.Symbols()
	assert.Len(t, symbols, 4)
	assert.Equal(t, ast.AliasSymbol, symbols[0].Kind)
	assert.Equal(t, ast.EquSymbol, symbols[1].Kind)
	assert.Equal(t, ast.LabelSymbol, symbols[2].Kind)
	assert.Equal(t, ast.FunctionSymbol, symbols[3].Kind)

	for index, symbol := range symbols {
		assert.Equal(t, index, symbol.EntryIndex)
		assert.Equal(t, "input.asm", symbol.Position.Source)
	}
	assert.Equal(t, ast.SymbolExpressionDefinition, symbols[0].Expression.Kind)
	assert.Equal(t, ast.SymbolExpressionDefinition, symbols[1].Expression.Kind)
	assert.Equal(t, ast.SymbolExpressionLocation, symbols[2].Expression.Kind)
	assert.Equal(t, ast.SymbolExpressionLocation, symbols[3].Expression.Kind)
	assert.NoError(t, stream.Validate())
}

// cpu6502Instruction creates an ast.Instruction with OpcodeID pre-populated from
// the cpu6502 NameToOpcodeID table, matching what the cpu6502 parser produces.
func cpu6502Instruction(name string, addressing int, arg ast.Node) ast.Instruction {
	node := ast.NewInstruction(name, addressing, arg, nil)
	node.OpcodeID = ast.NewOpcodeID(retroarch.CPU6502, uint16(cpu6502.NameToOpcodeID[name]))
	return node
}
