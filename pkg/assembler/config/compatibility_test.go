package config

import (
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

func TestParseCompatibilityMode(t *testing.T) {
	tests := []struct {
		input    string
		expected CompatibilityMode
		wantErr  bool
	}{
		{"default", CompatDefault, false},
		{"x816", CompatX816, false},
		{"asm6", CompatAsm6, false},
		{"ca65", CompatCa65, false},
		{"nesasm", CompatNesasm, false},
		{"X816", CompatX816, false},   // case insensitive
		{"ASM6", CompatAsm6, false},   // case insensitive
		{" ca65 ", CompatCa65, false}, // whitespace trimming
		{"invalid", CompatDefault, true},
		{"", CompatDefault, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mode, err := ParseCompatibilityMode(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, mode)
		})
	}
}

func TestCompatibilityMode_String(t *testing.T) {
	tests := []struct {
		mode     CompatibilityMode
		expected string
	}{
		{CompatDefault, "default"},
		{CompatX816, "x816"},
		{CompatAsm6, "asm6"},
		{CompatCa65, "ca65"},
		{CompatNesasm, "nesasm"},
		{CompatibilityMode(99), "CompatibilityMode(99)"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.mode.String())
	}
}

func TestCompatibilityMode_Features(t *testing.T) {
	t.Run("colon optional labels", func(t *testing.T) {
		assert.False(t, CompatDefault.ColonOptionalLabels())
		assert.True(t, CompatX816.ColonOptionalLabels())
		assert.True(t, CompatAsm6.ColonOptionalLabels())
		assert.False(t, CompatCa65.ColonOptionalLabels())
		assert.False(t, CompatNesasm.ColonOptionalLabels())
	})

	t.Run("anonymous labels", func(t *testing.T) {
		assert.False(t, CompatDefault.AnonymousLabels())
		assert.True(t, CompatX816.AnonymousLabels())
		assert.True(t, CompatAsm6.AnonymousLabels())
		assert.False(t, CompatCa65.AnonymousLabels())
		assert.False(t, CompatNesasm.AnonymousLabels())
	})

	t.Run("asterisk program counter", func(t *testing.T) {
		assert.False(t, CompatDefault.AsteriskProgramCounter())
		assert.True(t, CompatX816.AsteriskProgramCounter())
		assert.False(t, CompatAsm6.AsteriskProgramCounter())
		assert.True(t, CompatCa65.AsteriskProgramCounter())
		assert.True(t, CompatNesasm.AsteriskProgramCounter())
	})

	t.Run("bank byte operator", func(t *testing.T) {
		assert.False(t, CompatDefault.BankByteOperator())
		assert.True(t, CompatX816.BankByteOperator())
		assert.False(t, CompatAsm6.BankByteOperator())
		assert.True(t, CompatCa65.BankByteOperator())
		assert.False(t, CompatNesasm.BankByteOperator())
	})
}
