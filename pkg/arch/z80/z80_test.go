package z80

import (
	"testing"

	z80profile "github.com/retroenv/retroasm/pkg/arch/z80/profile"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/assert"
)

func TestNew(t *testing.T) {
	cfg := New()
	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.Arch)
	assert.Equal(t, 16, cfg.Arch.AddressWidth())
}

func TestNew_WithProfile(t *testing.T) {
	cfg := New(WithProfile(z80profile.StrictDocumented))
	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.Arch)
	assert.Equal(t, 16, cfg.Arch.AddressWidth())
}

func TestInstructionLookup(t *testing.T) {
	cfg := New()

	t.Run("ld has variants", func(t *testing.T) {
		group, ok := cfg.Arch.Instruction(cpuz80.LdName)
		assert.True(t, ok)
		assert.NotNil(t, group)
		assert.Equal(t, cpuz80.LdName, group.Name)
		assert.NotEmpty(t, group.Variants)
		assert.True(t, containsInstruction(group.Variants, cpuz80.LdImm8))
		assert.True(t, containsInstruction(group.Variants, cpuz80.LdReg8))
	})

	t.Run("lookup is case-insensitive", func(t *testing.T) {
		group, ok := cfg.Arch.Instruction("LD")
		assert.True(t, ok)
		assert.NotNil(t, group)
		assert.Equal(t, cpuz80.LdName, group.Name)
	})

	t.Run("cb family instruction is available", func(t *testing.T) {
		group, ok := cfg.Arch.Instruction(cpuz80.RlcName)
		assert.True(t, ok)
		assert.NotNil(t, group)
		assert.True(t, containsInstruction(group.Variants, cpuz80.CBRlc))
	})

	t.Run("indexed bit instruction is available", func(t *testing.T) {
		group, ok := cfg.Arch.Instruction(cpuz80.DdcbShiftName)
		assert.True(t, ok)
		assert.NotNil(t, group)
		assert.True(t, containsInstruction(group.Variants, cpuz80.DdcbShift))
	})

	t.Run("unknown instruction returns false", func(t *testing.T) {
		group, ok := cfg.Arch.Instruction("unknown")
		assert.False(t, ok)
		assert.Nil(t, group)
	})
}
