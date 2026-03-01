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
}

func TestAssembleZ80Fixtures(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
	}{
		{name: "basic", fixture: "basic.asm"},
		{name: "indexed and prefixed opcodes", fixture: "indexed.asm"},
		{name: "branches", fixture: "branches.asm"},
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
