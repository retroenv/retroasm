package sm83

import (
	"testing"

	cpusm83 "github.com/retroenv/retrogolib/arch/cpu/sm83"
	"github.com/retroenv/retrogolib/assert"
)

func TestNew(t *testing.T) {
	cfg := New()
	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.Arch)
	assert.Equal(t, 16, cfg.Arch.AddressWidth())
}

func TestInstructionLookup(t *testing.T) {
	cfg := New()

	t.Run("ld has variants", func(t *testing.T) {
		group, ok := cfg.Arch.Instruction(cpusm83.LdName)
		assert.True(t, ok)
		assert.NotNil(t, group)
		assert.Equal(t, cpusm83.LdName, group.Name)
		assert.NotEmpty(t, group.Variants)
		assert.True(t, containsInstruction(group.Variants, cpusm83.LdImm8))
		assert.True(t, containsInstruction(group.Variants, cpusm83.LdReg8))
	})

	t.Run("lookup is case-insensitive", func(t *testing.T) {
		group, ok := cfg.Arch.Instruction("LD")
		assert.True(t, ok)
		assert.NotNil(t, group)
		assert.Equal(t, cpusm83.LdName, group.Name)
	})

	t.Run("cb family instruction is available", func(t *testing.T) {
		group, ok := cfg.Arch.Instruction(cpusm83.RlcName)
		assert.True(t, ok)
		assert.NotNil(t, group)
		assert.True(t, containsInstruction(group.Variants, cpusm83.CBRlc))
	})

	t.Run("implied instruction nop", func(t *testing.T) {
		group, ok := cfg.Arch.Instruction(cpusm83.NopName)
		assert.True(t, ok)
		assert.NotNil(t, group)
		assert.True(t, containsInstruction(group.Variants, cpusm83.Nop))
	})

	t.Run("sm83-specific swap instruction", func(t *testing.T) {
		group, ok := cfg.Arch.Instruction(cpusm83.SwapName)
		assert.True(t, ok)
		assert.NotNil(t, group)
		assert.True(t, containsInstruction(group.Variants, cpusm83.CBSwap))
	})

	t.Run("sm83-specific ldh instruction", func(t *testing.T) {
		group, ok := cfg.Arch.Instruction(cpusm83.LdhName)
		assert.True(t, ok)
		assert.NotNil(t, group)
		assert.NotEmpty(t, group.Variants)
	})

	t.Run("unknown instruction returns false", func(t *testing.T) {
		group, ok := cfg.Arch.Instruction("unknown")
		assert.False(t, ok)
		assert.Nil(t, group)
	})
}
