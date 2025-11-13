package chip8

import (
	"testing"

	"github.com/retroenv/retrogolib/arch/cpu/chip8"
	"github.com/retroenv/retrogolib/assert"
)

func TestNew(t *testing.T) {
	cfg := New()
	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.Arch)
}

func TestAddressWidth(t *testing.T) {
	cfg := New()
	width := cfg.Arch.AddressWidth()
	assert.Equal(t, 12, width)
}

func TestInstruction(t *testing.T) {
	cfg := New()

	tests := []struct {
		name        string
		instruction string
		shouldExist bool
	}{
		{"CLS exists", "cls", true},
		{"RET exists", "ret", true},
		{"JP exists", "jp", true},
		{"CALL exists", "call", true},
		{"LD exists", "ld", true},
		{"ADD exists", "add", true},
		{"DRW exists", "drw", true},
		{"Invalid instruction", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ins, ok := cfg.Arch.Instruction(tt.instruction)
			assert.Equal(t, tt.shouldExist, ok)
			if tt.shouldExist {
				assert.NotNil(t, ins)
				assert.Equal(t, tt.instruction, ins.Name)
			} else {
				assert.Nil(t, ins)
			}
		})
	}
}

func TestInstructionAddressingModes(t *testing.T) {
	cfg := New()

	tests := []struct {
		name          string
		instruction   string
		addressingCnt int
	}{
		{"CLS has implied", "cls", 1},
		{"JP has multiple modes", "jp", 2},
		{"LD has many modes", "ld", 11},
		{"ADD has multiple modes", "add", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ins, ok := cfg.Arch.Instruction(tt.instruction)
			assert.True(t, ok)
			assert.NotNil(t, ins)
			assert.Equal(t, tt.addressingCnt, len(ins.Addressing))
		})
	}
}

func TestAllInstructions(t *testing.T) {
	expectedInstructions := []string{
		"add", "and", "call", "cls", "drw", "jp", "ld", "or", "ret", "rnd",
		"se", "shl", "shr", "skp", "sknp", "sne", "sub", "subn", "xor",
	}

	for _, name := range expectedInstructions {
		t.Run("instruction_"+name, func(t *testing.T) {
			ins, ok := chip8.Instructions[name]
			assert.True(t, ok)
			assert.NotNil(t, ins)
			assert.Equal(t, name, ins.Name)
		})
	}
}
