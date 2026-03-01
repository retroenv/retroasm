package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/retroenv/retroasm/pkg/retroasm"
	"github.com/retroenv/retrogolib/assert"
	"github.com/retroenv/retrogolib/log"
)

func TestBuildLogFields(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		options  *optionFlags
		expected int
	}{
		{
			name:     "input only",
			input:    "test.asm",
			options:  &optionFlags{},
			expected: 1,
		},
		{
			name:     "input with cpu",
			input:    "test.asm",
			options:  &optionFlags{cpu: "6502"},
			expected: 2,
		},
		{
			name:     "input with system",
			input:    "test.asm",
			options:  &optionFlags{system: "nes"},
			expected: 2,
		},
		{
			name:     "input with cpu and system",
			input:    "test.asm",
			options:  &optionFlags{cpu: "6502", system: "nes"},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := buildLogFields(tt.input, tt.options)
			assert.Len(t, fields, tt.expected)

			// First field should always be input
			assert.Equal(t, "input", fields[0].Key)
			assert.Equal(t, tt.input, fields[0].Value.String())
		})
	}
}

func TestCreateLogger(t *testing.T) {
	tests := []struct {
		name     string
		options  *optionFlags
		expected log.Level
	}{
		{
			name:     "default level",
			options:  &optionFlags{},
			expected: log.InfoLevel,
		},
		{
			name:     "debug level",
			options:  &optionFlags{debug: true},
			expected: log.DebugLevel,
		},
		{
			name:     "quiet level",
			options:  &optionFlags{quiet: true},
			expected: log.ErrorLevel,
		},
		{
			name:     "quiet overrides debug",
			options:  &optionFlags{debug: true, quiet: true},
			expected: log.ErrorLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := createLogger(tt.options)
			assert.NotNil(t, logger)
			assert.Equal(t, tt.expected, logger.Level())
		})
	}
}

func TestValidateSystem(t *testing.T) {
	logger := log.NewTestLogger(t)

	tests := []struct {
		name        string
		options     *optionFlags
		expectedErr error
		expectSys   string
	}{
		{
			name:        "empty system",
			options:     &optionFlags{logger: logger},
			expectedErr: nil,
		},
		{
			name:        "valid nes system",
			options:     &optionFlags{system: "nes", logger: logger},
			expectedErr: nil,
			expectSys:   systemNES,
		},
		{
			name:        "valid gameboy system",
			options:     &optionFlags{system: "gameboy", logger: logger},
			expectedErr: nil,
			expectSys:   systemGameBoy,
		},
		{
			name:        "invalid system",
			options:     &optionFlags{system: "invalid", logger: logger},
			expectedErr: ErrUnsupportedSystem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSystem(tt.options)
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectSys != "" {
				assert.Equal(t, tt.expectSys, tt.options.system)
			}
		})
	}
}

func TestValidateCPU(t *testing.T) {
	tests := []struct {
		name        string
		options     *optionFlags
		expectedErr error
	}{
		{
			name:        "empty cpu",
			options:     &optionFlags{},
			expectedErr: nil,
		},
		{
			name:        "valid 6502 cpu",
			options:     &optionFlags{cpu: "6502"},
			expectedErr: nil,
		},
		{
			name:        "valid z80 cpu",
			options:     &optionFlags{cpu: "z80"},
			expectedErr: nil,
		},
		{
			name:        "unsupported cpu",
			options:     &optionFlags{cpu: "x86"},
			expectedErr: ErrUnsupportedCPU,
		},
		{
			name:        "invalid cpu",
			options:     &optionFlags{cpu: "invalid"},
			expectedErr: ErrUnsupportedCPU,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCPU(tt.options)
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type architectureValidationCase struct {
	name        string
	options     *optionFlags
	expectedErr error
	expectCPU   string
	expectSys   string
}

func TestValidateAndProcessArchitecture(t *testing.T) {
	logger := log.NewTestLogger(t)

	for _, tt := range architectureValidationCases(logger) {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAndProcessArchitecture(tt.options)
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectCPU != "" {
				assert.Equal(t, tt.expectCPU, tt.options.cpu)
			}
			if tt.expectSys != "" {
				assert.Equal(t, tt.expectSys, tt.options.system)
			}
		})
	}
}

func architectureValidationCases(logger *log.Logger) []architectureValidationCase {
	return []architectureValidationCase{
		{
			name:        "no architecture specified",
			options:     &optionFlags{logger: logger},
			expectedErr: nil,
			expectCPU:   cpu6502,
			expectSys:   systemNES,
		},
		{
			name:        "valid nes system defaults to 6502",
			options:     &optionFlags{system: "nes", logger: logger},
			expectedErr: nil,
			expectCPU:   cpu6502,
			expectSys:   systemNES,
		},
		{
			name:        "valid 6502 cpu defaults to nes",
			options:     &optionFlags{cpu: "6502", logger: logger},
			expectedErr: nil,
			expectCPU:   cpu6502,
			expectSys:   systemNES,
		},
		{
			name:        "valid nes and 6502 combination",
			options:     &optionFlags{system: "nes", cpu: "6502", logger: logger},
			expectedErr: nil,
			expectCPU:   cpu6502,
			expectSys:   systemNES,
		},
		{
			name:        "z80 cpu defaults to generic",
			options:     &optionFlags{cpu: "z80", logger: logger},
			expectedErr: nil,
			expectCPU:   cpuZ80,
			expectSys:   systemGeneric,
		},
		{
			name:        "gameboy system defaults to z80",
			options:     &optionFlags{system: "gameboy", logger: logger},
			expectedErr: nil,
			expectCPU:   cpuZ80,
			expectSys:   systemGameBoy,
		},
		{
			name:        "incompatible nes and z80",
			options:     &optionFlags{system: "nes", cpu: "z80", logger: logger},
			expectedErr: ErrIncompatibleArch,
		},
		{
			name:        "unsupported system",
			options:     &optionFlags{system: "dos", logger: logger},
			expectedErr: ErrUnsupportedSystem,
		},
	}
}

func TestRegisterArchitectureForCPU(t *testing.T) {
	tests := []struct {
		name        string
		cpu         string
		expectedErr error
	}{
		{
			name: "register 6502",
			cpu:  cpu6502,
		},
		{
			name: "register z80",
			cpu:  cpuZ80,
		},
		{
			name:        "unsupported cpu",
			cpu:         "x86",
			expectedErr: ErrUnsupportedCPU,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asm := retroasm.New()
			err := registerArchitectureForCPU(asm, tt.cpu)
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestAssembleWithConfigFile(t *testing.T) {
	tmpFile := createTestConfigFile(t)

	tests := []struct {
		name        string
		cpu         string
		configPath  string
		expectedErr bool
	}{
		{"default config 6502", cpu6502, "", false},
		{"default config z80", cpuZ80, "", false},
		{"valid config file 6502", cpu6502, tmpFile, false},
		{"valid config file z80", cpuZ80, tmpFile, false},
		{"non-existent config file", cpu6502, "nonexistent.cfg", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runAssembleWithConfig(t.Context(), tt.cpu, tt.configPath)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func createTestConfigFile(t *testing.T) string {
	t.Helper()
	configContent := `MEMORY { CODE: start = $8000, size = $8000, fill = yes; }
SEGMENTS { CODE: load = CODE, type = rw; }`
	path := filepath.Join(t.TempDir(), "test_config.cfg")
	err := os.WriteFile(path, []byte(configContent), 0o644)
	assert.NoError(t, err)
	return path
}

func runAssembleWithConfig(ctx context.Context, cpuName, configPath string) error {
	asm := retroasm.New()
	if err := registerArchitectureForCPU(asm, cpuName); err != nil {
		return fmt.Errorf("registering architecture: %w", err)
	}
	input := &retroasm.TextInput{
		Source:     strings.NewReader(".segment \"CODE\"\nNOP"),
		SourceName: "test.asm",
		ConfigFile: configPath,
	}
	_, err := asm.AssembleText(ctx, input)
	if err != nil {
		return fmt.Errorf("assembling text: %w", err)
	}
	return nil
}
