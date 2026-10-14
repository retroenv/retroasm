package codec_test

import (
	"fmt"
	"strings"
	"testing"

	cpu68000parser "github.com/retroenv/retroasm/pkg/arch/cpu68000/parser"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
	"github.com/retroenv/retrogolib/assert"
)

func TestCPU68000Codec_RegisterNamedSymbolsRoundTrip(t *testing.T) {
	c := newCPU68000Codec(t)
	for _, name := range []string{"A0", "a7", "D0", "d7", "SP", "sr", "CCR", "usp", "PC", "ordinary"} {
		for _, long := range []bool{false, true} {
			for _, mnemonic := range []string{"move.l", "jsr", "bra.w"} {
				t.Run(fmt.Sprintf("%s/%s/long=%t", mnemonic, name, long), func(t *testing.T) {
					address := cpu68000parser.Absolute(long, ast.NewLabel(name))
					operands := cpu68000parser.UnaryOperands(cpu68000.SizeWord, address)
					if mnemonic == "move.l" {
						operands = cpu68000parser.BinaryOperands(cpu68000.SizeLong, address, cpu68000parser.DataRegister(1))
					}
					built, err := codec.BuildInstruction(c, mnemonic, operands)
					assert.NoError(t, err)
					formatted, err := c.FormatInstruction(built)
					assert.NoError(t, err)
					parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
					assert.NoError(t, err)
					assert.NoError(t, c.ValidateInstruction(parsed))
					want, err := c.Assemble(t.Context(), []ast.Node{ast.NewLabel(name), built})
					assert.NoError(t, err)
					got, err := c.Assemble(t.Context(), []ast.Node{ast.NewLabel(name), parsed})
					assert.NoError(t, err)
					assert.Equal(t, want.Binary, got.Binary)
				})
			}
		}
	}
}
