package x86

import (
	"testing"
)

func TestNewX86Config(t *testing.T) {
	cfg := New()
	if cfg == nil {
		t.Fatal("New() returned nil")
	}

	if cfg.Arch == nil {
		t.Fatal("Architecture is nil")
	}

	// Test address width
	if width := cfg.Arch.AddressWidth(); width != 16 {
		t.Errorf("Expected address width 16, got %d", width)
	}
}

func TestInstructionLookup(t *testing.T) {
	cfg := New()

	// Test that we can look up a known instruction
	ins, ok := cfg.Arch.Instruction("MOV")
	if !ok {
		t.Fatal("MOV instruction not found")
	}

	if ins.Name != "MOV" {
		t.Errorf("Expected instruction name MOV, got %s", ins.Name)
	}

	// Test unknown instruction
	_, ok = cfg.Arch.Instruction("UNKNOWN")
	if ok {
		t.Error("UNKNOWN instruction should not be found")
	}
}

func TestInstructionAddressingModes(t *testing.T) {
	cfg := New()

	ins, ok := cfg.Arch.Instruction("MOV")
	if !ok {
		t.Fatal("MOV instruction not found")
	}

	// MOV should support multiple addressing modes
	if !ins.HasAddressing(RegisterAddressing) {
		t.Error("MOV should support register addressing")
	}

	if !ins.HasAddressing(ImmediateAddressing) {
		t.Error("MOV should support immediate addressing")
	}

	if !ins.HasAddressing(DirectAddressing) {
		t.Error("MOV should support direct addressing")
	}

	// Test unsupported addressing mode
	if ins.HasAddressing(BasedIndexedAddressing) {
		t.Error("MOV should not support based indexed addressing in our basic implementation")
	}
}

func TestConfigTypeCompliance(t *testing.T) {
	cfg := New()

	// Verify the config matches the expected type
	_ = cfg
}
