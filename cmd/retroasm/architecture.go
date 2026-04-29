package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch/m6502"
	"github.com/retroenv/retroasm/pkg/retroasm"
	"github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/set"
)

// Structured errors for validation.
var (
	ErrUnsupportedSystem = errors.New("unsupported system")
	ErrUnsupportedCPU    = errors.New("unsupported CPU architecture")
	ErrIncompatibleArch  = errors.New("incompatible system and CPU combination")
)

// CPU and system constants — defined for all architectures so lookup tables are complete.
// Registration (registerArchitectureForCPU) is implemented per architecture wave.
const (
	cpu6502  = string(arch.M6502)
	cpuChip8 = string(arch.CHIP8)
	cpuZ80   = string(arch.Z80)

	systemChip8      = string(arch.CHIP8System)
	systemGameBoy    = string(arch.GameBoy)
	systemGeneric    = string(arch.Generic)
	systemNES        = string(arch.NES)
	systemZXSpectrum = string(arch.ZXSpectrum)
)

var supportedSystemsByCPU = map[string]set.Set[string]{
	cpu6502:  set.NewFromSlice([]string{systemNES, systemGeneric}),
	cpuChip8: set.NewFromSlice([]string{systemChip8}),
	cpuZ80:   set.NewFromSlice([]string{systemGeneric, systemGameBoy, systemZXSpectrum}),
}

var defaultSystemByCPU = map[string]string{
	cpu6502:  systemNES,
	cpuChip8: systemChip8,
	cpuZ80:   systemGeneric,
}

var defaultCPUBySystem = map[string]string{
	systemChip8:      cpuChip8,
	systemGameBoy:    cpuZ80,
	systemGeneric:    cpuZ80,
	systemNES:        cpu6502,
	systemZXSpectrum: cpuZ80,
}

var supportedSystems = set.NewFromSlice([]string{
	systemChip8,
	systemGameBoy,
	systemGeneric,
	systemNES,
	systemZXSpectrum,
})

// validateAndProcessArchitecture validates the CPU and system flags and applies defaults.
func validateAndProcessArchitecture(options *optionFlags) error {
	normalizeArchitectureOptions(options)
	if setDefaultArchitecture(options) {
		return nil
	}
	if err := validateSystem(options); err != nil {
		return err
	}
	if err := validateCPU(options); err != nil {
		return err
	}
	if err := applyDerivedArchitectureDefaults(options); err != nil {
		return err
	}
	return validateArchitectureCompatibility(options)
}

func normalizeArchitectureOptions(options *optionFlags) {
	options.cpu = strings.ToLower(strings.TrimSpace(options.cpu))
	options.system = strings.ToLower(strings.TrimSpace(options.system))
}

func setDefaultArchitecture(options *optionFlags) bool {
	if options.cpu != "" || options.system != "" {
		return false
	}
	options.cpu = cpu6502
	options.system = systemNES
	return true
}

func applyDerivedArchitectureDefaults(options *optionFlags) error {
	if options.cpu == "" && options.system != "" {
		defaultCPU, ok := defaultCPUBySystem[options.system]
		if !ok {
			return fmt.Errorf("%w: no default CPU for system '%s'", ErrIncompatibleArch, options.system)
		}
		options.cpu = defaultCPU
	}
	if options.system == "" && options.cpu != "" {
		defaultSystem, ok := defaultSystemByCPU[options.cpu]
		if !ok {
			return fmt.Errorf("%w: no default system for CPU '%s'", ErrIncompatibleArch, options.cpu)
		}
		options.system = defaultSystem
	}
	return nil
}

func validateArchitectureCompatibility(options *optionFlags) error {
	compatibleSystems, ok := supportedSystemsByCPU[options.cpu]
	if !ok {
		return fmt.Errorf("%w: %s (supported: %s, %s, %s)", ErrUnsupportedCPU, options.cpu, cpu6502, cpuChip8, cpuZ80)
	}
	if !compatibleSystems.Contains(options.system) {
		return fmt.Errorf("%w: cpu '%s' is not compatible with system '%s'", ErrIncompatibleArch, options.cpu, options.system)
	}
	return nil
}

func validateSystem(options *optionFlags) error {
	if options.system == "" {
		return nil
	}

	sys, ok := arch.SystemFromString(options.system)
	if !ok {
		return fmt.Errorf(
			"%w: %s (supported: %s, %s, %s, %s, %s)",
			ErrUnsupportedSystem, options.system,
			systemChip8, systemNES, systemGeneric, systemGameBoy, systemZXSpectrum,
		)
	}
	options.system = string(sys)
	if !supportedSystems.Contains(options.system) {
		return fmt.Errorf(
			"%w: %s (supported: %s, %s, %s, %s, %s)",
			ErrUnsupportedSystem, options.system,
			systemChip8, systemNES, systemGeneric, systemGameBoy, systemZXSpectrum,
		)
	}
	return nil
}

func validateCPU(options *optionFlags) error {
	if options.cpu == "" {
		return nil
	}

	cpu, ok := arch.FromString(options.cpu)
	if !ok {
		return fmt.Errorf("%w: %s (supported: %s, %s, %s)", ErrUnsupportedCPU, options.cpu, cpu6502, cpuChip8, cpuZ80)
	}
	options.cpu = string(cpu)
	if _, supported := supportedSystemsByCPU[options.cpu]; !supported {
		return fmt.Errorf("%w: %s (supported: %s, %s, %s)", ErrUnsupportedCPU, cpu, cpu6502, cpuChip8, cpuZ80)
	}
	return nil
}

func registerArchitectureForCPU(asm retroasm.Assembler, cpuName string) error {
	switch cpuName {
	case cpu6502:
		cfg := m6502.New()
		adapter := retroasm.NewArchitectureAdapter(cpu6502, cfg, cfg)
		if err := asm.RegisterArchitecture(cpu6502, adapter); err != nil {
			return fmt.Errorf("registering architecture '%s': %w", cpu6502, err)
		}
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedCPU, cpuName)
	}
}
