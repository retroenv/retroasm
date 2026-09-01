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
	retrochip8 "github.com/retroenv/retrogolib/arch/cpu/chip8"
	retro6502 "github.com/retroenv/retrogolib/arch/cpu/cpu6502"
	retro65816 "github.com/retroenv/retrogolib/arch/cpu/cpu65816"
	retro68000 "github.com/retroenv/retrogolib/arch/cpu/cpu68000"
	retrosm83 "github.com/retroenv/retrogolib/arch/cpu/sm83"
	retroz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/assert"
)

func TestArchitectureOpcodeIDRegistries(t *testing.T) {
	t.Parallel()

	t.Run("cpu6502", func(t *testing.T) {
		t.Parallel()
		assertRegisteredOpcodeIDs(t, asmcpu6502.New().Arch, retroarch.CPU6502, retro6502.NameToOpcodeID)
	})
	t.Run("cpu65816", func(t *testing.T) {
		t.Parallel()
		assertRegisteredOpcodeIDs(t, asmcpu65816.New().Arch, retroarch.CPU65816, retro65816.NameToOpcodeID)
	})
	t.Run("cpu68000", func(t *testing.T) {
		t.Parallel()
		assertRegisteredOpcodeIDs(t, asmcpu68000.New().Arch, retroarch.CPU68000, retro68000.NameToOpcodeID)
	})
	t.Run("sm83", func(t *testing.T) {
		t.Parallel()
		assertRegisteredOpcodeIDs(t, asmsm83.New().Arch, retroarch.SM83, retrosm83.NameToOpcodeID)
	})
	t.Run("z80", func(t *testing.T) {
		t.Parallel()
		assertRegisteredOpcodeIDs(t, asmz80.New().Arch, retroarch.Z80, retroz80.NameToOpcodeID)
	})
	t.Run("chip8", func(t *testing.T) {
		t.Parallel()
		assertRegisteredOpcodeIDs(t, asmchip8.New().Arch, retroarch.CHIP8, retrochip8.NameToOpcodeID)
	})
}

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

func assertRegisteredOpcodeIDs[T any, ID ~uint8](
	t *testing.T,
	architecture arch.Architecture[T],
	expectedArchitecture retroarch.Architecture,
	registered map[string]ID,
) {

	t.Helper()
	for name, expectedValue := range registered {
		instruction, ok := architecture.Instruction(name)
		if !ok {
			t.Errorf("registered mnemonic %q is not exposed by the architecture", name)
			continue
		}

		identity := architecture.OpcodeID(instruction)
		if !identity.ValidFor(expectedArchitecture) {
			t.Errorf("registered mnemonic %q has invalid identity %+v", name, identity)
			continue
		}
		if identity.Value != uint16(expectedValue) {
			t.Errorf("registered mnemonic %q has identity value %d, want %d", name, identity.Value, expectedValue)
		}
	}
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
