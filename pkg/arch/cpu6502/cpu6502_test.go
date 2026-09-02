package cpu6502

import (
	"testing"

	"github.com/retroenv/retrogolib/arch/cpu/cpu6502"
	"github.com/retroenv/retrogolib/assert"
)

func TestNew_WithVariantUsesRetrogolibInstructionRegistry(t *testing.T) {
	t.Parallel()

	nmos := New(WithVariant(cpu6502.VariantNMOS6502))
	_, ok := nmos.Arch.Instruction(cpu6502.Bbr0.Name)
	assert.False(t, ok)
	lda, ok := nmos.Arch.Instruction(cpu6502.LdaName)
	assert.True(t, ok)
	assert.Equal(t, cpu6502.LdaInst, lda)

	wdc := New(WithVariant(cpu6502.Variant65C02))
	bbr, ok := wdc.Arch.Instruction(cpu6502.Bbr0.Name)
	assert.True(t, ok)
	assert.Equal(t, cpu6502.Bbr0, bbr)
	lda, ok = wdc.Arch.Instruction(cpu6502.LdaName)
	assert.True(t, ok)
	assert.Equal(t, cpu6502.Lda65C02Inst, lda)

	synertek := New(WithVariant(cpu6502.VariantSynertek65C02))
	_, ok = synertek.Arch.Instruction(cpu6502.Bbr0.Name)
	assert.False(t, ok)
}
