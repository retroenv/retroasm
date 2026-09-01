package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch/cpu6502"
	"github.com/retroenv/retroasm/pkg/arch/cpu65816"
	"github.com/retroenv/retroasm/pkg/arch/cpu68000"
	"github.com/retroenv/retroasm/pkg/arch/sm83"
	"github.com/retroenv/retroasm/pkg/arch/z80"
	"github.com/retroenv/retroasm/pkg/arch/z80/profile"
	"github.com/retroenv/retroasm/pkg/assembler/config"
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

// Supported architectures and systems.
const (
	cpu6502Name  = string(arch.CPU6502)
	cpu65816Name = string(arch.CPU65816)
	cpuChip8     = string(arch.CHIP8)
	cpu68000Name = string(arch.CPU68000)
	cpuSM83      = string(arch.SM83)
	cpuZ80       = string(arch.Z80)

	systemChip8      = string(arch.CHIP8System)
	systemNES        = string(arch.NES)
	systemSNES       = string(arch.SNES)
	systemGeneric    = string(arch.Generic)
	systemGameBoy    = string(arch.GameBoy)
	systemZXSpectrum = string(arch.ZXSpectrum)
)

var supportedSystemsByCPU = map[string]set.Set[string]{
	cpu6502Name:  set.NewFromSlice([]string{systemNES, systemGeneric}),
	cpu65816Name: set.NewFromSlice([]string{systemSNES, systemGeneric}),
	cpuChip8:     set.NewFromSlice([]string{systemChip8}),
	cpu68000Name: set.NewFromSlice([]string{systemGeneric}),
	cpuSM83:      set.NewFromSlice([]string{systemGameBoy, systemGeneric}),
	cpuZ80:       set.NewFromSlice([]string{systemGeneric, systemGameBoy, systemZXSpectrum}),
}

var defaultSystemByCPU = map[string]string{
	cpu6502Name:  systemNES,
	cpu65816Name: systemSNES,
	cpuChip8:     systemChip8,
	cpu68000Name: systemGeneric,
	cpuSM83:      systemGameBoy,
	cpuZ80:       systemGeneric,
}

var defaultCPUBySystem = map[string]string{
	systemChip8:      cpuChip8,
	systemNES:        cpu6502Name,
	systemSNES:       cpu65816Name,
	systemGeneric:    cpuZ80,
	systemGameBoy:    cpuSM83,
	systemZXSpectrum: cpuZ80,
}

var supportedSystems = set.NewFromSlice([]string{
	systemChip8,
	systemNES,
	systemSNES,
	systemGeneric,
	systemGameBoy,
	systemZXSpectrum,
})

// validateAndProcessArchitecture validates CPU/system flags, applies defaults, and enforces compatibility.
func validateAndProcessArchitecture(options *optionFlags) error {
	z80ProfileRequested := normalizeArchitectureOptions(options)
	if setDefaultArchitecture(options, z80ProfileRequested) {
		return nil
	}

	if err := validateSystem(options); err != nil {
		return err
	}

	if err := validateCPU(options); err != nil {
		return err
	}

	if err := applyDerivedArchitectureDefaults(options, z80ProfileRequested); err != nil {
		return err
	}
	if err := validateArchitectureCompatibility(options); err != nil {
		return err
	}

	if err := validateZ80Profile(options, z80ProfileRequested); err != nil {
		return err
	}

	return nil
}

func normalizeArchitectureOptions(options *optionFlags) bool {
	options.cpu = strings.ToLower(strings.TrimSpace(options.cpu))
	options.system = strings.ToLower(strings.TrimSpace(options.system))
	options.z80Profile = strings.ToLower(strings.TrimSpace(options.z80Profile))
	return options.z80Profile != ""
}

func setDefaultArchitecture(options *optionFlags, z80ProfileRequested bool) bool {
	if options.cpu != "" || options.system != "" || z80ProfileRequested {
		return false
	}

	options.cpu = cpu6502Name
	options.system = systemNES
	options.z80Profile = profile.Default.String()
	return true
}

func applyDerivedArchitectureDefaults(options *optionFlags, z80ProfileRequested bool) error {
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

	if options.cpu == "" && options.system == "" && z80ProfileRequested {
		options.cpu = cpuZ80
		options.system = defaultSystemByCPU[cpuZ80]
	}

	return nil
}

func validateArchitectureCompatibility(options *optionFlags) error {
	compatibleSystems, ok := supportedSystemsByCPU[options.cpu]
	if !ok {
		return fmt.Errorf("%w: %s (supported: %s, %s, %s, %s, %s, %s)", ErrUnsupportedCPU, options.cpu, cpu6502Name, cpu65816Name, cpuChip8, cpu68000Name, cpuSM83, cpuZ80)
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
			"%w: %s (supported: %s, %s, %s, %s, %s, %s)",
			ErrUnsupportedSystem,
			options.system,
			systemChip8,
			systemNES,
			systemSNES,
			systemGeneric,
			systemGameBoy,
			systemZXSpectrum,
		)
	}
	options.system = string(sys)

	if !supportedSystems.Contains(options.system) {
		return fmt.Errorf(
			"%w: %s (supported: %s, %s, %s, %s, %s, %s)",
			ErrUnsupportedSystem,
			options.system,
			systemChip8,
			systemNES,
			systemSNES,
			systemGeneric,
			systemGameBoy,
			systemZXSpectrum,
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
		return fmt.Errorf("%w: %s (supported: %s, %s, %s, %s, %s, %s)", ErrUnsupportedCPU, options.cpu, cpu6502Name, cpu65816Name, cpuChip8, cpu68000Name, cpuSM83, cpuZ80)
	}
	options.cpu = string(cpu)

	if cpu != arch.CPU6502 && cpu != arch.CPU65816 && cpu != arch.CHIP8 && cpu != arch.CPU68000 && cpu != arch.SM83 && cpu != arch.Z80 {
		return fmt.Errorf("%w: %s (supported: %s, %s, %s, %s, %s, %s)", ErrUnsupportedCPU, cpu, cpu6502Name, cpu65816Name, cpuChip8, cpu68000Name, cpuSM83, cpuZ80)
	}

	return nil
}

func validateZ80Profile(options *optionFlags, requested bool) error {
	profileKind, err := profile.Parse(options.z80Profile)
	if err != nil {
		return fmt.Errorf("parsing z80 profile: %w", err)
	}

	options.z80Profile = profileKind.String()
	if options.cpu == cpuZ80 {
		return nil
	}

	if requested && profileKind != profile.Default {
		return fmt.Errorf(
			"%w: z80 profile '%s' requires cpu '%s'",
			ErrIncompatibleArch,
			options.z80Profile,
			cpuZ80,
		)
	}

	options.z80Profile = profile.Default.String()
	return nil
}

func registerArchitectureForCPU(asm retroasm.Assembler, cpuName, z80ProfileName string, //nolint:funlen // repetitive switch cases
	compatMode config.CompatibilityMode,
) error {

	switch cpuName {
	case cpu6502Name:
		cfg := cpu6502.New()
		cfg.CompatibilityMode = compatMode
		adapter := retroasm.NewArchitectureAdapter(cpu6502Name, cfg, cfg)
		if err := asm.RegisterArchitecture(cpu6502Name, adapter); err != nil {
			return fmt.Errorf("registering architecture '%s': %w", cpu6502Name, err)
		}
		return nil

	case cpu65816Name:
		cfg := cpu65816.New()
		cfg.CompatibilityMode = compatMode
		adapter := retroasm.NewArchitectureAdapter(cpu65816Name, cfg, cfg)
		if err := asm.RegisterArchitecture(cpu65816Name, adapter); err != nil {
			return fmt.Errorf("registering architecture '%s': %w", cpu65816Name, err)
		}
		return nil

	case cpu68000Name:
		cfg := cpu68000.New()
		cfg.CompatibilityMode = compatMode
		adapter := retroasm.NewArchitectureAdapter(cpu68000Name, cfg, cfg)
		if err := asm.RegisterArchitecture(cpu68000Name, adapter); err != nil {
			return fmt.Errorf("registering architecture '%s': %w", cpu68000Name, err)
		}
		return nil

	case cpuSM83:
		cfg := sm83.New()
		cfg.CompatibilityMode = compatMode
		adapter := retroasm.NewArchitectureAdapter(cpuSM83, cfg, cfg)
		if err := asm.RegisterArchitecture(cpuSM83, adapter); err != nil {
			return fmt.Errorf("registering architecture '%s': %w", cpuSM83, err)
		}
		return nil

	case cpuZ80:
		profileKind, err := profile.Parse(z80ProfileName)
		if err != nil {
			return fmt.Errorf("parsing z80 profile: %w", err)
		}

		cfg := z80.New(z80.WithProfile(profileKind))
		cfg.CompatibilityMode = compatMode
		adapter := retroasm.NewArchitectureAdapter(cpuZ80, cfg, cfg)
		if err := asm.RegisterArchitecture(cpuZ80, adapter); err != nil {
			return fmt.Errorf("registering architecture '%s': %w", cpuZ80, err)
		}
		return nil

	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedCPU, cpuName)
	}
}
