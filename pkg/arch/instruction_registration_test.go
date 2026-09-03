package arch_test

import (
	"slices"
	"strings"
	"testing"

	asmarch "github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/chip8"
	"github.com/retroenv/retroasm/pkg/arch/cpu6502"
	"github.com/retroenv/retroasm/pkg/arch/cpu65816"
	cpu65816parser "github.com/retroenv/retroasm/pkg/arch/cpu65816/parser"
	"github.com/retroenv/retroasm/pkg/arch/cpu68000"
	"github.com/retroenv/retroasm/pkg/arch/sm83"
	"github.com/retroenv/retroasm/pkg/arch/z80"
	"github.com/retroenv/retrogolib/arch"
	retrocpu65816 "github.com/retroenv/retrogolib/arch/cpu/cpu65816"
	"github.com/retroenv/retrogolib/assert"
	"github.com/retroenv/retrogolib/set"
)

func TestInstructionRegistrationsAreStableAndArchitectureScoped(t *testing.T) {
	tests := []struct {
		architecture arch.Architecture
		provider     any
		dynamic      bool
	}{
		{architecture: arch.CPU6502, provider: cpu6502.New().Arch},
		{architecture: arch.CPU65816, provider: cpu65816.New().Arch},
		{architecture: arch.CPU68000, provider: cpu68000.New().Arch, dynamic: true},
		{architecture: arch.Z80, provider: z80.New().Arch},
		{architecture: arch.SM83, provider: sm83.New().Arch},
		{architecture: arch.CHIP8, provider: chip8.New().Arch},
	}

	for _, test := range tests {
		provider, ok := test.provider.(asmarch.InstructionRegistrationProvider)
		assert.True(t, ok)
		registrations := provider.InstructionRegistrations()
		assert.NotEmpty(t, registrations)
		assert.True(t, slices.IsSortedFunc(registrations, compareInstructionRegistrations))

		names := set.New[string]()
		for _, registration := range registrations {
			assert.NotEmpty(t, registration.Name)
			assert.True(t, registration.OpcodeID.ValidFor(test.architecture))
			assert.False(t, names.Contains(registration.Name))
			assert.True(t, slices.IsSorted(registration.Addressings))
			assert.True(t, slices.IsSorted(registration.RegisterForms))
			assert.True(t, slices.IsSorted(registration.RegisterPairForms))
			assert.Equal(t, test.dynamic, registration.DynamicOperands)
			names.Add(registration.Name)
		}
	}
}

func TestCPU65816InstructionRegistrationsIncludeCombinedAddressings(t *testing.T) {
	provider, ok := cpu65816.New().Arch.(asmarch.InstructionRegistrationProvider)
	assert.True(t, ok)
	registrations := make(map[string]asmarch.InstructionRegistration)
	for _, registration := range provider.InstructionRegistrations() {
		registrations[registration.Name] = registration
	}

	adc := registrations[retrocpu65816.AdcName]
	assert.True(t, slices.Contains(adc.Addressings, int(cpu65816parser.AbsoluteDirectPageAddressing)))
	assert.True(t, slices.Contains(adc.Addressings, int(cpu65816parser.XAddressing)))
	ldx := registrations[retrocpu65816.LdxName]
	assert.True(t, slices.Contains(ldx.Addressings, int(cpu65816parser.YAddressing)))
}

func compareInstructionRegistrations(left, right asmarch.InstructionRegistration) int {
	return strings.Compare(left.Name, right.Name)
}
