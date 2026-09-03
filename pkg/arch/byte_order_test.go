package arch_test

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/chip8"
	"github.com/retroenv/retroasm/pkg/arch/cpu6502"
	"github.com/retroenv/retroasm/pkg/arch/cpu65816"
	"github.com/retroenv/retroasm/pkg/arch/cpu68000"
	"github.com/retroenv/retroasm/pkg/arch/sm83"
	"github.com/retroenv/retroasm/pkg/arch/x86"
	"github.com/retroenv/retroasm/pkg/arch/z80"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/assert"
)

func TestArchitectureByteOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		architecture any
		order        ast.ByteOrder
	}{
		{name: "CPU6502", architecture: cpu6502.New().Arch, order: ast.ByteOrderLittle},
		{name: "CPU65816", architecture: cpu65816.New().Arch, order: ast.ByteOrderLittle},
		{name: "CPU68000", architecture: cpu68000.New().Arch, order: ast.ByteOrderBig},
		{name: "CHIP-8", architecture: chip8.New().Arch, order: ast.ByteOrderBig},
		{name: "Z80", architecture: z80.New().Arch, order: ast.ByteOrderLittle},
		{name: "SM83", architecture: sm83.New().Arch, order: ast.ByteOrderLittle},
		{name: "x86", architecture: x86.New().Arch, order: ast.ByteOrderLittle},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			orderer, ok := test.architecture.(arch.ByteOrderer)
			assert.True(t, ok)
			assert.Equal(t, test.order, orderer.ByteOrder())
		})
	}
}
