package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	z80profile "github.com/retroenv/retroasm/pkg/arch/z80/profile"
	"github.com/retroenv/retroasm/pkg/retroasm"
	"github.com/retroenv/retrogolib/assert"
)

var z80FixtureExpected = map[string][]byte{
	"basic.asm": {
		0x00,
		0x01, 0x34, 0x12,
		0x3E, 0x2A,
		0xCB, 0x5F,
		0x20, 0xFC,
	},
	"indexed.asm": {
		0xDD, 0x21, 0x34, 0x12,
		0xFD, 0x21, 0x78, 0x56,
		0xDD, 0x7E, 0x05,
		0xFD, 0x77, 0xFE,
		0xDD, 0xCB, 0x05, 0x5E,
		0xFD, 0xCB, 0xFF, 0x56,
		0xDD, 0xE9,
		0xFD, 0xE9,
		0xED, 0x56,
		0xFF,
	},
	"branches.asm": {
		0x20, 0x01,
		0x00,
		0x18, 0xFB,
		0xC2, 0x0B, 0x80,
		0xCD, 0x0B, 0x80,
		0xC9,
	},
	"io_extended.asm": {
		0x3A, 0x34, 0x12,
		0x32, 0x45, 0x23,
		0xED, 0x4B, 0x56, 0x34,
		0xED, 0x43, 0x67, 0x45,
		0xDB, 0x12,
		0xD3, 0x34,
		0xED, 0x40,
		0xED, 0x59,
	},
	"offsets.asm": {
		0xC3, 0x0E, 0x80,
		0x3A, 0x10, 0x80,
		0x32, 0x11, 0x80,
		0xDB, 0x11,
		0xD3, 0x22,
		0x00,
		0x00,
		0x11, 0x22, 0x33, 0x44,
	},
	"offsets_chained.asm": {
		0xC3, 0x0E, 0x80,
		0x3A, 0x10, 0x80,
		0x32, 0x11, 0x80,
		0xDB, 0x12,
		0xD3, 0x21,
		0x00,
		0x11, 0x22, 0x33, 0x44, 0x55,
	},
	"expressions.asm": {
		0xC3, 0x0E, 0x80,
		0x3A, 0x0E, 0x80,
		0x32, 0x0F, 0x80,
		0xDD, 0x7E, 0x01,
		0x00,
		0x10, 0x20, 0x30,
	},
	"compatibility.asm": {
		0x20, 0x03,
		0xC3, 0x0A, 0x80,
		0x3A, 0x0C, 0x80,
		0x10, 0xF6,
		0xC9,
		0x10, 0x20, 0x30,
	},
	"indexed_boundaries.asm": {
		0xDD, 0x7E, 0x80,
		0xFD, 0x77, 0x7F,
		0xDD, 0xCB, 0x7F, 0x46,
		0xFD, 0xCB, 0x80, 0xBE,
		0xDD, 0xCB, 0xFF, 0xDE,
		0xC9,
	},
	"profile_strict_documented.asm": {
		0x00,
		0xCB, 0x5F,
		0x20, 0xFB,
	},
	"profile_gameboy_subset.asm": {
		0x00,
		0x3E, 0x2A,
		0x20, 0xFB,
	},
}

func TestAssembleZ80Fixtures(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
	}{
		{name: "basic", fixture: "basic.asm"},
		{name: "indexed and prefixed opcodes", fixture: "indexed.asm"},
		{name: "branches", fixture: "branches.asm"},
		{name: "io and extended forms", fixture: "io_extended.asm"},
		{name: "offset expressions", fixture: "offsets.asm"},
		{name: "chained offset expressions", fixture: "offsets_chained.asm"},
		{name: "symbolic expressions", fixture: "expressions.asm"},
		{name: "compatibility control-flow and expressions", fixture: "compatibility.asm"},
		{name: "indexed displacement boundaries", fixture: "indexed_boundaries.asm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := assembleZ80Fixture(t, tt.fixture)
			assert.NoError(t, err)
			assert.Equal(t, z80FixtureExpected[tt.fixture], output)
		})
	}
}

func TestAssembleZ80ProfileFixtures(t *testing.T) {
	tests := []struct {
		name      string
		fixture   string
		profile   string
		expectBin []byte
	}{
		{
			name:      "strict documented profile fixture",
			fixture:   "profile_strict_documented.asm",
			profile:   z80profile.StrictDocumented.String(),
			expectBin: z80FixtureExpected["profile_strict_documented.asm"],
		},
		{
			name:      "gameboy subset profile fixture",
			fixture:   "profile_gameboy_subset.asm",
			profile:   z80profile.GameBoySubset.String(),
			expectBin: z80FixtureExpected["profile_gameboy_subset.asm"],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := assembleZ80FixtureWithProfile(t, tt.fixture, tt.profile)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectBin, output)
		})
	}
}

func TestAssembleZ80ProfileFixtures_Rejects(t *testing.T) {
	tests := []struct {
		name       string
		fixture    string
		profile    string
		errorParts []string
	}{
		{
			name:    "strict documented rejects undocumented fixture",
			fixture: "profile_strict_documented_rejects.asm",
			profile: z80profile.StrictDocumented.String(),
			errorParts: []string{
				"undocumented",
				z80profile.StrictDocumented.String(),
			},
		},
		{
			name:    "gameboy subset rejects io fixture",
			fixture: "profile_gameboy_subset_rejects.asm",
			profile: z80profile.GameBoySubset.String(),
			errorParts: []string{
				"outside profile",
				z80profile.GameBoySubset.String(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := assembleZ80FixtureWithProfile(t, tt.fixture, tt.profile)
			assert.Error(t, err)
			for _, part := range tt.errorParts {
				assert.ErrorContains(t, err, part)
			}
		})
	}
}

func TestAssembleZ80Fixtures_RelativeOverflow(t *testing.T) {
	_, err := assembleZ80Fixture(t, "branches_overflow.asm")
	assert.Error(t, err)
	assert.ErrorContains(t, err, "relative offset")
}

func TestAssembleZ80Profiles(t *testing.T) {
	t.Run("default profile allows undocumented sll", func(t *testing.T) {
		output, err := assembleZ80SourceWithProfile(
			t,
			".segment \"CODE\"\nsll a",
			z80profile.Default.String(),
		)
		assert.NoError(t, err)
		assert.Equal(t, []byte{0xCB, 0x37}, output)
	})

	t.Run("strict documented rejects undocumented sll", func(t *testing.T) {
		_, err := assembleZ80SourceWithProfile(
			t,
			".segment \"CODE\"\nsll a",
			z80profile.StrictDocumented.String(),
		)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "undocumented")
		assert.ErrorContains(t, err, z80profile.StrictDocumented.String())
	})

	t.Run("gameboy subset rejects ix instructions", func(t *testing.T) {
		_, err := assembleZ80SourceWithProfile(
			t,
			".segment \"CODE\"\nld ix,$1234",
			z80profile.GameBoySubset.String(),
		)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "gameboy-z80-subset")
		assert.ErrorContains(t, err, "unsupported prefix")
	})

	t.Run("gameboy subset rejects in/out", func(t *testing.T) {
		_, err := assembleZ80SourceWithProfile(
			t,
			".segment \"CODE\"\nin a,($12)",
			z80profile.GameBoySubset.String(),
		)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "outside profile")
	})
}

func TestAssembleZ80ResolverPaths(t *testing.T) { //nolint:funlen
	tests := []struct {
		name   string
		source string
		expect []byte
	}{
		// resolveNoOperand second pass (implied with RegisterOpcodes).
		{name: "neg", source: "neg", expect: []byte{0xED, 0x44}},
		{name: "retn", source: "retn", expect: []byte{0xED, 0x45}},

		// resolveSingleOperand second pass (value addressing with RegisterOpcodes).
		{name: "sub immediate", source: "sub $01", expect: []byte{0xD6, 0x01}},

		// resolveSingleRegisterOperand indexed fallback.
		{name: "sub (ix+3)", source: "sub (ix+3)", expect: []byte{0xDD, 0x96, 0x03}},
		{name: "sub (iy-1)", source: "sub (iy-1)", expect: []byte{0xFD, 0x96, 0xFF}},

		// resolveAluRegisterPairOperands.
		{name: "add a,b", source: "add a,b", expect: []byte{0x80}},
		{name: "add hl,bc", source: "add hl,bc", expect: []byte{0x09}},
		{name: "add hl,de", source: "add hl,de", expect: []byte{0x19}},
		{name: "add hl,sp", source: "add hl,sp", expect: []byte{0x39}},
		{name: "add ix,bc", source: "add ix,bc", expect: []byte{0xDD, 0x09}},
		{name: "add iy,de", source: "add iy,de", expect: []byte{0xFD, 0x19}},
		{name: "sbc hl,bc", source: "sbc hl,bc", expect: []byte{0xED, 0x42}},
		{name: "adc hl,de", source: "adc hl,de", expect: []byte{0xED, 0x5A}},

		// resolveIndirectLoadStoreOperands.
		{name: "ld a,(hl)", source: "ld a,(hl)", expect: []byte{0x7E}},
		{name: "ld (hl),a", source: "ld (hl),a", expect: []byte{0x77}},
		{name: "ld a,(bc)", source: "ld a,(bc)", expect: []byte{0x0A}},
		{name: "ld (de),a", source: "ld (de),a", expect: []byte{0x12}},
		{name: "ld b,(hl)", source: "ld b,(hl)", expect: []byte{0x46}},
		{name: "ld (hl),e", source: "ld (hl),e", expect: []byte{0x73}},

		// resolveIndirectImmediateOperands.
		{name: "ld (hl),$42", source: "ld (hl),$42", expect: []byte{0x36, 0x42}},
		{name: "ld (ix+5),$42", source: "ld (ix+5),$42", expect: []byte{0xDD, 0x36, 0x05, 0x42}},
		{name: "ld (iy-8),$99", source: "ld (iy-8),$99", expect: []byte{0xFD, 0x36, 0xF8, 0x99}},
		{name: "ld (ix+0),$00", source: "ld (ix+0),$00", expect: []byte{0xDD, 0x36, 0x00, 0x00}},

		// resolveSpecialRegisterPairOperands.
		{name: "ld i,a", source: "ld i,a", expect: []byte{0xED, 0x47}},
		{name: "ld r,a", source: "ld r,a", expect: []byte{0xED, 0x4F}},
		{name: "ld a,i", source: "ld a,i", expect: []byte{0xED, 0x57}},
		{name: "ld a,r", source: "ld a,r", expect: []byte{0xED, 0x5F}},
		{name: "ld sp,hl", source: "ld sp,hl", expect: []byte{0xF9}},
		{name: "ld sp,ix", source: "ld sp,ix", expect: []byte{0xDD, 0xF9}},
		{name: "ld sp,iy", source: "ld sp,iy", expect: []byte{0xFD, 0xF9}},
		{name: "ex de,hl", source: "ex de,hl", expect: []byte{0xEB}},
		{name: "ex af,af", source: "ex af,af", expect: []byte{0x08}},

		// resolveIndirectLoadStoreOperands fallback (EX (SP),rr).
		{name: "ex (sp),hl", source: "ex (sp),hl", expect: []byte{0xE3}},
		{name: "ex (sp),ix", source: "ex (sp),ix", expect: []byte{0xDD, 0xE3}},
		{name: "ex (sp),iy", source: "ex (sp),iy", expect: []byte{0xFD, 0xE3}},

		// resolveRegisterValueOperands: ALU RegA stripping.
		{name: "add a,42", source: "add a,42", expect: []byte{0xC6, 0x2A}},
		{name: "adc a,$FF", source: "adc a,$FF", expect: []byte{0xCE, 0xFF}},
		{name: "sub a,1", source: "sub 1", expect: []byte{0xD6, 0x01}},
		{name: "sbc a,$10", source: "sbc a,$10", expect: []byte{0xDE, 0x10}},
		{name: "and $0F", source: "and $0F", expect: []byte{0xE6, 0x0F}},
		{name: "xor $FF", source: "xor $FF", expect: []byte{0xEE, 0xFF}},
		{name: "or $80", source: "or $80", expect: []byte{0xF6, 0x80}},
		{name: "cp 0", source: "cp 0", expect: []byte{0xFE, 0x00}},

		// resolveRegisterValueOperands: LD A,n preserves RegA.
		{name: "ld a,42", source: "ld a,42", expect: []byte{0x3E, 0x2A}},

		// resolveExtendedMemoryFromRegister: HL preference.
		{name: "ld ($1234),hl", source: "ld ($1234),hl", expect: []byte{0x22, 0x34, 0x12}},
		{name: "ld ($1234),bc", source: "ld ($1234),bc", expect: []byte{0xED, 0x43, 0x34, 0x12}},

		// JP (HL) indirect.
		{name: "jp (hl)", source: "jp (hl)", expect: []byte{0xE9}},
		{name: "jp (ix)", source: "jp (ix)", expect: []byte{0xDD, 0xE9}},
		{name: "jp (iy)", source: "jp (iy)", expect: []byte{0xFD, 0xE9}},

		// Port I/O operations (not confused with indirect register).
		{name: "out (c),e", source: "out (c),e", expect: []byte{0xED, 0x59}},
		{name: "in b,(c)", source: "in b,(c)", expect: []byte{0xED, 0x40}},
		{name: "in a,($12)", source: "in a,($12)", expect: []byte{0xDB, 0x12}},
		{name: "out ($34),a", source: "out ($34),a", expect: []byte{0xD3, 0x34}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := assembleZ80SourceWithProfile(
				t,
				".segment \"CODE\"\n"+tt.source,
				z80profile.Default.String(),
			)
			assert.NoError(t, err)
			assert.Equal(t, tt.expect, output)
		})
	}
}

func assembleZ80Fixture(t *testing.T, fixture string) ([]byte, error) {
	t.Helper()

	return assembleZ80FixtureWithProfile(t, fixture, z80profile.Default.String())
}

func assembleZ80FixtureWithProfile(t *testing.T, fixture, profileName string) ([]byte, error) {
	t.Helper()
	sourcePath := z80FixturePath(t, fixture)
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("reading fixture '%s': %w", fixture, err)
	}

	asm := retroasm.New()
	if err := registerArchitectureForCPU(asm, cpuZ80, profileName); err != nil {
		return nil, fmt.Errorf("registering z80 architecture: %w", err)
	}

	input := &retroasm.TextInput{
		Source:     bytes.NewReader(source),
		SourceName: sourcePath,
	}
	output, err := asm.AssembleText(t.Context(), input)
	if err != nil {
		return nil, fmt.Errorf("assembling fixture '%s': %w", fixture, err)
	}

	return output.Binary, nil
}

func assembleZ80SourceWithProfile(t *testing.T, source, profileName string) ([]byte, error) {
	t.Helper()

	asm := retroasm.New()
	if err := registerArchitectureForCPU(asm, cpuZ80, profileName); err != nil {
		return nil, fmt.Errorf("registering z80 architecture: %w", err)
	}

	input := &retroasm.TextInput{
		Source:     strings.NewReader(source),
		SourceName: "inline_z80_fixture.asm",
	}
	output, err := asm.AssembleText(t.Context(), input)
	if err != nil {
		return nil, fmt.Errorf("assembling z80 source: %w", err)
	}

	return output.Binary, nil
}

func z80FixturePath(t *testing.T, fixture string) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	assert.True(t, ok)

	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	return filepath.Join(projectRoot, "tests", "z80", fixture)
}
