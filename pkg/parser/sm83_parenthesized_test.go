package parser_test

import (
	"fmt"
	"testing"

	asmsm83 "github.com/retroenv/retroasm/pkg/arch/sm83"
	sm83parser "github.com/retroenv/retroasm/pkg/arch/sm83/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/assert"
)

func TestSM83ParenthesizedSymbolOffsets(t *testing.T) {
	for _, offset := range []int64{-17, -1, 1, 17} {
		for _, store := range []bool{false, true} {
			t.Run(fmt.Sprintf("offset=%d/store=%t", offset, store), func(t *testing.T) {
				address := fmt.Sprintf("(target%+d)", offset)
				source := "ld a," + address
				if store {
					source = "ld " + address + ",a"
				}
				instruction := parseInstruction(t, asmsm83.New().Arch, source)
				resolved := instruction.Argument.(ast.InstructionArgument).Value.(sm83parser.ResolvedInstruction)
				refs := resolved.InstructionReferences()
				assert.Len(t, refs, 1)
				symbol, addend, ok := ast.ParseSymbolReference(refs[0].Value.(ast.Expression).Value)
				assert.True(t, ok)
				assert.Equal(t, "target", symbol)
				assert.Equal(t, offset, addend)
				formatted, err := sm83parser.FormatInstruction(instruction)
				assert.NoError(t, err)
				reparsed := parseInstruction(t, asmsm83.New().Arch, formatted)
				assert.Equal(t, instruction.OpcodeID, reparsed.OpcodeID)
				parsedResolved := reparsed.Argument.(ast.InstructionArgument).Value.(sm83parser.ResolvedInstruction)
				parsedRefs := parsedResolved.InstructionReferences()
				assert.Len(t, parsedRefs, 1)
				parsedSymbol, parsedAddend, valid := ast.ParseSymbolReference(parsedRefs[0].Value.(ast.Expression).Value)
				assert.True(t, valid)
				assert.Equal(t, symbol, parsedSymbol)
				assert.Equal(t, addend, parsedAddend)
			})
		}
	}
}

func TestSM83ParenthesizedHLUpdates(t *testing.T) {
	for _, base := range []string{"hl", "HL"} {
		for _, suffix := range []string{"+", "-"} {
			for _, store := range []bool{false, true} {
				t.Run(fmt.Sprintf("base=%s/update=%s/store=%t", base, suffix, store), func(t *testing.T) {
					address := "(" + base + suffix + ")"
					source := "ld a," + address
					index := 1
					if store {
						source, index = "ld "+address+",a", 0
					}
					instruction := parseInstruction(t, asmsm83.New().Arch, source)
					resolved := instruction.Argument.(ast.InstructionArgument).Value.(sm83parser.ResolvedInstruction)
					kind := sm83parser.OperandHLIncrement
					if suffix == "-" {
						kind = sm83parser.OperandHLDecrement
					}
					assert.Equal(t, kind, resolved.Operands[index].Kind)
				})
			}
		}
	}
}
