package profile

import (
	"testing"

	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		want     Kind
		wantErr  bool
		errorMsg string
	}{
		{
			name:  "empty defaults",
			value: "",
			want:  Default,
		},
		{
			name:  "default",
			value: "default",
			want:  Default,
		},
		{
			name:  "strict documented",
			value: "strict-documented",
			want:  StrictDocumented,
		},
		{
			name:  "gameboy subset",
			value: "gameboy-z80-subset",
			want:  GameBoySubset,
		},
		{
			name:     "unknown profile",
			value:    "strict",
			wantErr:  true,
			errorMsg: "unsupported z80 profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorContains(t, err, tt.errorMsg)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateInstruction_DefaultAndStrict(t *testing.T) {
	err := ValidateInstruction(
		Default,
		cpuz80.CBSll,
		cpuz80.RegisterAddressing,
		[]cpuz80.RegisterParam{cpuz80.RegA},
	)
	assert.NoError(t, err)

	err = ValidateInstruction(
		StrictDocumented,
		cpuz80.CBSll,
		cpuz80.RegisterAddressing,
		[]cpuz80.RegisterParam{cpuz80.RegA},
	)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "undocumented")
	assert.ErrorContains(t, err, "strict-documented")

	err = ValidateInstruction(
		StrictDocumented,
		cpuz80.EdIm0,
		cpuz80.ImmediateAddressing,
		[]cpuz80.RegisterParam{cpuz80.RegI},
	)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "undocumented")

	err = ValidateInstruction(
		StrictDocumented,
		cpuz80.Nop,
		cpuz80.ImpliedAddressing,
		nil,
	)
	assert.NoError(t, err)
}

func TestValidateInstruction_GameBoySubset(t *testing.T) {
	err := ValidateInstruction(
		GameBoySubset,
		cpuz80.DdLdIXnn,
		cpuz80.ImmediateAddressing,
		[]cpuz80.RegisterParam{cpuz80.RegIX},
	)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "unsupported prefix")

	err = ValidateInstruction(
		GameBoySubset,
		cpuz80.InPort,
		cpuz80.PortAddressing,
		nil,
	)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "outside profile")

	err = ValidateInstruction(
		GameBoySubset,
		cpuz80.Nop,
		cpuz80.ImpliedAddressing,
		nil,
	)
	assert.NoError(t, err)
}
