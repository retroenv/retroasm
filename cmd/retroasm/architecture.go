package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch/m6502"
	archm65816 "github.com/retroenv/retroasm/pkg/arch/m65816"
	archm68000 "github.com/retroenv/retroasm/pkg/arch/m68000"
	archsm83 "github.com/retroenv/retroasm/pkg/arch/sm83"
	archz80 "github.com/retroenv/retroasm/pkg/arch/z80"
	z80profile "github.com/retroenv/retroasm/pkg/arch/z80/profile"
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
	cpu6502   = string(arch.M6502)
	cpu65816  = string(arch.M65816)
	cpuChip8  = string(arch.CHIP8)
	cpuM68000 = string(arch.M68000)
	cpuSM83   = string(arch.SM83)
	cpuZ80    = string(arch.Z80)

	systemChip8      = string(arch.CHIP8System)
	systemNES        = string(arch.NES)
	systemSNES       = string(arch.SNES)
	systemGeneric    = string(arch.Generic)
	systemGameBoy    = string(arch.GameBoy)
	systemZXSpectrum = string(arch.ZXSpectrum)
)

var supportedSystemsByCPU = map[string]set.Set[string]{
	cpu6502:   set.NewFromSlice([]string{systemNES, systemGeneric}),
	cpu65816:  set.NewFromSlice([]string{systemSNES, systemGeneric}),
	cpuChip8:  set.NewFromSlice([]string{systemChip8}),
	cpuM68000: set.NewFromSlice([]string{systemGeneric}),
	cpuSM83:   set.NewFromSlice([]string{systemGameBoy, systemGeneric}),
	cpuZ80:    set.NewFromSlice([]string{systemGeneric, systemGameBoy, systemZXSpectrum}),
}

var defaultSystemByCPU = map[string]string{
	cpu6502:   systemNES,
	cpu65816:  systemSNES,
	cpuChip8:  systemChip8,
	cpuM68000: systemGeneric,
	cpuSM83:   systemGameBoy,
	cpuZ80:    systemGeneric,
}

var defaultCPUBySystem = map[string]string{
	systemChip8:      cpuChip8,
	systemNES:        cpu6502,
	systemSNES:       cpu65816,
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

	options.cpu = cpu6502
	options.system = systemNES
	options.z80Profile = z80profile.Default.String()
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
		return fmt.Errorf("%w: %s (supported: %s, %s, %s, %s, %s, %s)", ErrUnsupportedCPU, options.cpu, cpu6502, cpu65816, cpuChip8, cpuM68000, cpuSM83, cpuZ80)
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
		return fmt.Errorf("%w: %s (supported: %s, %s, %s, %s, %s, %s)", ErrUnsupportedCPU, options.cpu, cpu6502, cpu65816, cpuChip8, cpuM68000, cpuSM83, cpuZ80)
	}
	options.cpu = string(cpu)

	if cpu != arch.M6502 && cpu != arch.M65816 && cpu != arch.CHIP8 && cpu != arch.M68000 && cpu != arch.SM83 && cpu != arch.Z80 {
		return fmt.Errorf("%w: %s (supported: %s, %s, %s, %s, %s, %s)", ErrUnsupportedCPU, cpu, cpu6502, cpu65816, cpuChip8, cpuM68000, cpuSM83, cpuZ80)
	}

	return nil
}

func validateZ80Profile(options *optionFlags, requested bool) error {
	profileKind, err := z80profile.Parse(options.z80Profile)
	if err != nil {
		return fmt.Errorf("parsing z80 profile: %w", err)
	}

	options.z80Profile = profileKind.String()
	if options.cpu == cpuZ80 {
		return nil
	}

	if requested && profileKind != z80profile.Default {
		return fmt.Errorf(
			"%w: z80 profile '%s' requires cpu '%s'",
			ErrIncompatibleArch,
			options.z80Profile,
			cpuZ80,
		)
	}

	options.z80Profile = z80profile.Default.String()
	return nil
}

func registerArchitectureForCPU(asm retroasm.Assembler, cpuName, z80ProfileName string, //nolint:funlen // repetitive switch cases
	compatMode config.CompatibilityMode,
) error {

	switch cpuName {
	case cpu6502:
		cfg := m6502.New()
		cfg.CompatibilityMode = compatMode
		adapter := retroasm.NewArchitectureAdapter(cpu6502, cfg, cfg)
		if err := asm.RegisterArchitecture(cpu6502, adapter); err != nil {
			return fmt.Errorf("registering architecture '%s': %w", cpu6502, err)
		}
		return nil

	case cpu65816:
		cfg := archm65816.New()
		cfg.CompatibilityMode = compatMode
		adapter := retroasm.NewArchitectureAdapter(cpu65816, cfg, cfg)
		if err := asm.RegisterArchitecture(cpu65816, adapter); err != nil {
			return fmt.Errorf("registering architecture '%s': %w", cpu65816, err)
		}
		return nil

	case cpuM68000:
		cfg := archm68000.New()
		cfg.CompatibilityMode = compatMode
		adapter := retroasm.NewArchitectureAdapter(cpuM68000, cfg, cfg)
		if err := asm.RegisterArchitecture(cpuM68000, adapter); err != nil {
			return fmt.Errorf("registering architecture '%s': %w", cpuM68000, err)
		}
		return nil

	case cpuSM83:
		cfg := archsm83.New()
		cfg.CompatibilityMode = compatMode
		adapter := retroasm.NewArchitectureAdapter(cpuSM83, cfg, cfg)
		if err := asm.RegisterArchitecture(cpuSM83, adapter); err != nil {
			return fmt.Errorf("registering architecture '%s': %w", cpuSM83, err)
		}
		return nil

	case cpuZ80:
		profileKind, err := z80profile.Parse(z80ProfileName)
		if err != nil {
			return fmt.Errorf("parsing z80 profile: %w", err)
		}

		cfg := archz80.New(archz80.WithProfile(profileKind))
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
