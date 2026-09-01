package parser_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/retroenv/retroasm/pkg/arch"
	asmchip8 "github.com/retroenv/retroasm/pkg/arch/chip8"
	asmcpu6502 "github.com/retroenv/retroasm/pkg/arch/cpu6502"
	asmcpu65816 "github.com/retroenv/retroasm/pkg/arch/cpu65816"
	asmcpu68000 "github.com/retroenv/retroasm/pkg/arch/cpu68000"
	cpu68000parser "github.com/retroenv/retroasm/pkg/arch/cpu68000/parser"
	asmsm83 "github.com/retroenv/retroasm/pkg/arch/sm83"
	asmz80 "github.com/retroenv/retroasm/pkg/arch/z80"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	asmparser "github.com/retroenv/retroasm/pkg/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	retroarch "github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/assert"
)

func TestParser_ParallelArchitectureOpcodeIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		architecture retroarch.Architecture
		parse        func(*testing.T) ast.OpcodeID
	}{
		{"cpu6502", retroarch.CPU6502, func(t *testing.T) ast.OpcodeID {
			t.Helper()
			return parseOpcodeID(t, asmcpu6502.New().Arch, "lda #1")
		}},
		{"cpu65816", retroarch.CPU65816, func(t *testing.T) ast.OpcodeID {
			t.Helper()
			return parseOpcodeID(t, asmcpu65816.New().Arch, "lda #1")
		}},
		{"cpu68000", retroarch.CPU68000, func(t *testing.T) ast.OpcodeID {
			t.Helper()
			return parseOpcodeID(t, asmcpu68000.New().Arch, "move.w d0,d1")
		}},
		{"sm83", retroarch.SM83, func(t *testing.T) ast.OpcodeID {
			t.Helper()
			return parseOpcodeID(t, asmsm83.New().Arch, "ld a,1")
		}},
		{"z80", retroarch.Z80, func(t *testing.T) ast.OpcodeID {
			t.Helper()
			return parseOpcodeID(t, asmz80.New().Arch, "ld a,1")
		}},
		{"chip8", retroarch.CHIP8, func(t *testing.T) ast.OpcodeID {
			t.Helper()
			return parseOpcodeID(t, asmchip8.New().Arch, "ld v0,1")
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			identity := test.parse(t)
			assert.True(t, identity.ValidFor(test.architecture))
		})
	}
}

func TestParser_CPU68000ConcurrentMnemonicResolution(t *testing.T) {
	t.Parallel()

	architecture := asmcpu68000.New().Arch
	for index := range 64 {
		mnemonic := "beq"
		if index%2 != 0 {
			mnemonic = "bne"
		}
		t.Run(fmt.Sprintf("%s-%d", mnemonic, index), func(t *testing.T) {
			t.Parallel()
			instruction := parseInstruction(t, architecture, mnemonic+" target")
			argument := instruction.Argument.(ast.InstructionArgument)
			resolved := argument.Value.(cpu68000parser.ResolvedInstruction)
			_, expected, ok := cpu68000parser.ParseConditionCode(mnemonic)
			assert.True(t, ok)
			assert.Equal(t, expected, resolved.Extra)
		})
	}
}

func parseOpcodeID[T any](t *testing.T, architecture arch.Architecture[T], source string) ast.OpcodeID {
	t.Helper()
	return parseInstruction(t, architecture, source).OpcodeID
}

func parseInstruction[T any](t *testing.T, architecture arch.Architecture[T], source string) ast.Instruction {
	t.Helper()
	assemblyParser := asmparser.New(architecture, strings.NewReader(source), config.CompatDefault)
	assert.NoError(t, assemblyParser.Read(t.Context()))
	nodes, err := assemblyParser.TokensToAstNodes()
	assert.NoError(t, err)
	assert.Len(t, nodes, 1)
	instruction, ok := ast.InstructionFromNode(nodes[0])
	assert.True(t, ok)
	return instruction
}
