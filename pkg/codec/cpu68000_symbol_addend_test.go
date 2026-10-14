package codec_test

import (
	"fmt"
	"strings"
	"testing"

	cpu68000parser "github.com/retroenv/retroasm/pkg/arch/cpu68000/parser"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
	"github.com/retroenv/retrogolib/assert"
)

func TestCPU68000CodecSymbolAddendRoundTrip(t *testing.T) {
	c := newCPU68000Codec(t)
	prefix, err := codec.BuildInstruction(c, "nop", cpu68000parser.Operands{})
	assert.NoError(t, err)
	for _, name := range []string{"A0", "d0", "ordinary"} {
		for _, long := range []bool{false, true} {
			for _, operator := range []token.Type{token.Plus, token.Minus} {
				for _, mnemonic := range []string{"move.l", "lea"} {
					t.Run(fmt.Sprintf("%s/long=%t/%v/%s", name, long, operator, mnemonic), func(t *testing.T) {
						value := ast.NewExpression(
							token.Token{
								Type:  token.Identifier,
								Value: name,
							},
							token.Token{
								Type: operator,
							},
							token.Token{
								Type:  token.Number,
								Value: "4",
							},
						)
						destination := cpu68000parser.DataRegister(0)
						if mnemonic == "lea" {
							destination = cpu68000parser.AddressRegister(0)
						}
						operands := cpu68000parser.BinaryOperands(cpu68000.SizeLong,
							cpu68000parser.Absolute(long, value), destination)
						built, err := codec.BuildInstruction(c, mnemonic, operands)
						assert.NoError(t, err)
						formatted, err := c.FormatInstruction(built)
						assert.NoError(t, err)
						parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
						assert.NoError(t, err)
						assert.NoError(t, c.ValidateInstruction(parsed))
						want, err := c.Assemble(t.Context(), []ast.Node{prefix, prefix, ast.NewLabel(name), built})
						assert.NoError(t, err)
						got, err := c.Assemble(t.Context(), []ast.Node{prefix, prefix, ast.NewLabel(name), parsed})
						assert.NoError(t, err)
						assert.Equal(t, want.Binary, got.Binary)
					})
				}
			}
		}
	}
}
