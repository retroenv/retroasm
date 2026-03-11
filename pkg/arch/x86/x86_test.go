package x86

import (
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

func TestNewX86Config(t *testing.T) {
	cfg := New()
	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.Arch)

	// Test address width
	width := cfg.Arch.AddressWidth()
	assert.Equal(t, 16, width)
}

func TestInstructionLookup(t *testing.T) {
	cfg := New()

	// Test that we can look up a known instruction
	ins, ok := cfg.Arch.Instruction("MOV")
	assert.True(t, ok)
	assert.Equal(t, "MOV", ins.Name)

	// Test unknown instruction
	_, ok = cfg.Arch.Instruction("UNKNOWN")
	assert.False(t, ok)
}

func TestInstructionAddressingModes(t *testing.T) {
	cfg := New()

	ins, ok := cfg.Arch.Instruction("MOV")
	assert.True(t, ok)

	// MOV should support multiple addressing modes
	assert.True(t, ins.HasAddressing(RegisterAddressing))
	assert.True(t, ins.HasAddressing(ImmediateAddressing))
	assert.True(t, ins.HasAddressing(DirectAddressing))

	// Test unsupported addressing mode
	assert.False(t, ins.HasAddressing(BasedIndexedAddressing))
}
