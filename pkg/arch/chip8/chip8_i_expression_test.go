package chip8

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/retroenv/retroasm/pkg/assembler"
	chip8cpu "github.com/retroenv/retrogolib/arch/cpu/chip8"
	"github.com/retroenv/retrogolib/assert"
)

func TestCHIP8IAddressExpressions(t *testing.T) {
	for _, test := range []struct {
		name       string
		expression string
		encoded    []byte
	}{
		{
			name:       "identifier-addition",
			expression: "base + $31",
			encoded:    []byte{0xAA, 0x31},
		},
		{
			name:       "numeric-addition",
			expression: "$F00 + $31",
			encoded:    []byte{0xAF, 0x31},
		},
		{
			name:       "parentheses",
			expression: "(base + $31)",
			encoded:    []byte{0xAA, 0x31},
		},
		{
			name:       "nested-subtraction",
			expression: "((base + $40) - 1)",
			encoded:    []byte{0xAA, 0x3F},
		},
		{
			name:       "identifier-subtraction",
			expression: "base - $10",
			encoded:    []byte{0xA9, 0xF0},
		},
		{
			name:       "last-address",
			expression: "$1000 - 1",
			encoded:    []byte{0xAF, 0xFF},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := "base = $A00\nld i, " + test.expression + "\n"
			assert.Equal(t, test.encoded, assembleChip8Source(t, source))
		})
	}
}

func TestCHIP8IAddressExpressionRejectsInvalidValuesAndSyntax(t *testing.T) {
	for _, operand := range []string{
		"(0 - 1)", "$FFF + 1", "(base + $600)", "base +", "(base + 1", "base + 1)", "base + missing",
	} {
		t.Run(operand, func(t *testing.T) {
			var output bytes.Buffer
			assembly := assembler.New(New(), &output)
			source := "base = $A00\nld i, " + operand + "\n"
			assert.Error(t, assembly.Process(t.Context(), strings.NewReader(source)))
		})
	}
}

func TestCHIP8IAddressExpressionKeepsPlainAndRegisterForms(t *testing.T) {
	assert.Equal(t, []byte{0xAA, 0x31, 0xF7, 0x1E}, assembleChip8Source(t, "ld i,$A31\nadd i,v7\n"))
	assert.Equal(t, []byte{0xA2, 0x03, 0x00, 0xE0}, assembleChip8Source(t, "ld i,v4+1\nv4:\ncls\n"))
}

func TestCHIP8IAddressExpressionWritesOnlyTheCompleteAddress(t *testing.T) {
	for _, address := range []uint16{0xA31, 0xFFF} {
		for _, leavesI := range []bool{false, true} {
			t.Run(fmt.Sprintf("address=%x/leavesI=%t", address, leavesI), func(t *testing.T) {
				source := fmt.Sprintf("base = $%X\nld v0,$A5\nld i,(base + $%X)\nld [i],v0\n",
					address&0xF00, address&0xFF)
				binary := assembleChip8Source(t, source)
				cpu := chip8cpu.New(chip8cpu.WithQuirks(chip8cpu.Quirks{
					LoadStoreLeavesI: leavesI,
				}))

				for index := range cpu.Memory {
					cpu.Memory[index] = byte(index*17 + 0x55)
				}

				copy(cpu.Memory[0x200:], binary)
				expected := cpu.Memory
				expected[address] = 0xA5

				for range 3 {
					assert.NoError(t, cpu.Step())
				}

				assert.Equal(t, expected, cpu.Memory)
				assert.Equal(t, uint8(0), cpu.SP)
			})
		}
	}
}
