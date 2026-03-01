package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := assembleZ80Fixture(t, tt.fixture)
			assert.NoError(t, err)
			assert.Equal(t, z80FixtureExpected[tt.fixture], output)
		})
	}
}

func TestAssembleZ80Fixtures_RelativeOverflow(t *testing.T) {
	_, err := assembleZ80Fixture(t, "branches_overflow.asm")
	assert.Error(t, err)
	assert.ErrorContains(t, err, "relative offset")
}

func assembleZ80Fixture(t *testing.T, fixture string) ([]byte, error) {
	t.Helper()
	sourcePath := z80FixturePath(t, fixture)
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("reading fixture '%s': %w", fixture, err)
	}

	asm := retroasm.New()
	if err := registerArchitectureForCPU(asm, cpuZ80); err != nil {
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

func z80FixturePath(t *testing.T, fixture string) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	assert.True(t, ok)

	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	return filepath.Join(projectRoot, "tests", "z80", fixture)
}
