// Package main implements retroasm, a retro computer assembler.
// It provides command-line interface for assembling retro computer code,
// supporting 6502, Z80, M68000, and Chip-8 architectures with ca65-compatible configuration.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch/chip8"
	"github.com/retroenv/retroasm/pkg/arch/m6502"
	archm68000 "github.com/retroenv/retroasm/pkg/arch/m68000"
	archz80 "github.com/retroenv/retroasm/pkg/arch/z80"
	z80profile "github.com/retroenv/retroasm/pkg/arch/z80/profile"
	"github.com/retroenv/retroasm/pkg/assembler"
	"github.com/retroenv/retroasm/pkg/retroasm"
	"github.com/retroenv/retrogolib/app"
	"github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/buildinfo"
	"github.com/retroenv/retrogolib/log"
	"github.com/retroenv/retrogolib/set"
)

// Structured errors for validation.
var (
	ErrUnsupportedSystem = errors.New("unsupported system")
	ErrUnsupportedCPU    = errors.New("unsupported CPU architecture")
	ErrIncompatibleArch  = errors.New("incompatible system and CPU combination")
)

// Build-time variables set by go build -ldflags.
var (
	version = "dev"
	commit  = ""
	date    = ""
)

// optionFlags holds command-line options and runtime configuration.
type optionFlags struct {
	logger     *log.Logger
	config     string
	output     string
	cpu        string
	system     string
	z80Profile string
	debug      bool
	quiet      bool
}

func main() {
	options, args := readArguments()
	printBanner(options)

	logFields := buildLogFields(args[0], options)
	options.logger.Info("Assembling file...", logFields...)

	if err := assembleFile(options, args); err != nil {
		options.logger.Error("Assembling failed", log.Err(err))
		os.Exit(1)
	}

	options.logger.Info("Assembling finished successfully", log.String("output", options.output))
}

// buildLogFields creates log fields for assembly operation.
func buildLogFields(input string, options *optionFlags) []log.Field {
	fields := []log.Field{log.String("input", input)}
	if options.cpu != "" {
		fields = append(fields, log.String("cpu", options.cpu))
	}
	if options.system != "" {
		fields = append(fields, log.String("system", options.system))
	}
	if options.cpu == cpuZ80 && options.z80Profile != "" {
		fields = append(fields, log.String("z80_profile", options.z80Profile))
	}
	return fields
}

// createLogger creates a configured logger based on options.
func createLogger(options *optionFlags) *log.Logger {
	cfg := log.DefaultConfig()
	if options.debug {
		cfg.Level = log.DebugLevel
	}
	if options.quiet {
		cfg.Level = log.ErrorLevel
	}
	return log.NewWithConfig(cfg)
}

// Supported architectures and systems.
const (
	cpu6502   = string(arch.M6502)
	cpuChip8  = string(arch.CHIP8)
	cpuM68000 = string(arch.M68000)
	cpuZ80    = string(arch.Z80)

	systemChip8      = string(arch.CHIP8System)
	systemNES        = string(arch.NES)
	systemGeneric    = string(arch.Generic)
	systemGameBoy    = string(arch.GameBoy)
	systemZXSpectrum = string(arch.ZXSpectrum)
)

var supportedSystemsByCPU = map[string]set.Set[string]{
	cpu6502:   set.NewFromSlice([]string{systemNES, systemGeneric}),
	cpuChip8:  set.NewFromSlice([]string{systemChip8}),
	cpuM68000: set.NewFromSlice([]string{systemGeneric}),
	cpuZ80:    set.NewFromSlice([]string{systemGeneric, systemGameBoy, systemZXSpectrum}),
}

var defaultSystemByCPU = map[string]string{
	cpu6502:   systemNES,
	cpuChip8:  systemChip8,
	cpuM68000: systemGeneric,
	cpuZ80:    systemGeneric,
}

var defaultCPUBySystem = map[string]string{
	systemChip8:      cpuChip8,
	systemNES:        cpu6502,
	systemGeneric:    cpuZ80,
	systemGameBoy:    cpuZ80,
	systemZXSpectrum: cpuZ80,
}

var supportedSystems = set.NewFromSlice([]string{
	systemChip8,
	systemNES,
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
		return fmt.Errorf("%w: %s (supported: %s, %s, %s, %s)", ErrUnsupportedCPU, options.cpu, cpu6502, cpuChip8, cpuM68000, cpuZ80)
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
			ErrUnsupportedSystem,
			options.system,
			systemChip8,
			systemNES,
			systemGeneric,
			systemGameBoy,
			systemZXSpectrum,
		)
	}
	options.system = string(sys)

	if !supportedSystems.Contains(options.system) {
		return fmt.Errorf(
			"%w: %s (supported: %s, %s, %s, %s, %s)",
			ErrUnsupportedSystem,
			options.system,
			systemChip8,
			systemNES,
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
		return fmt.Errorf("%w: %s (supported: %s, %s, %s, %s)", ErrUnsupportedCPU, options.cpu, cpu6502, cpuChip8, cpuM68000, cpuZ80)
	}
	options.cpu = string(cpu)

	if cpu != arch.M6502 && cpu != arch.CHIP8 && cpu != arch.M68000 && cpu != arch.Z80 {
		return fmt.Errorf("%w: %s (supported: %s, %s, %s, %s)", ErrUnsupportedCPU, cpu, cpu6502, cpuChip8, cpuM68000, cpuZ80)
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

// readArguments parses command-line arguments and validates configuration.
func readArguments() (*optionFlags, []string) {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	options := &optionFlags{}

	flags.BoolVar(&options.debug, "debug", false, "enable debug logging")
	flags.StringVar(&options.config, "c", "", "assembler config file")
	flags.StringVar(&options.output, "o", "", "name of the output file")
	flags.StringVar(&options.cpu, "cpu", "", "target CPU architecture (6502, chip8, m68000, z80)")
	flags.StringVar(&options.system, "system", "", "target system (chip8, nes, generic, gameboy, zx-spectrum)")
	flags.StringVar(
		&options.z80Profile,
		"z80-profile",
		"",
		"z80 instruction profile (default, strict-documented, gameboy-z80-subset)",
	)
	flags.BoolVar(&options.quiet, "q", false, "perform operations quietly")

	err := flags.Parse(os.Args[1:])
	args := flags.Args()

	// Create logger early for error reporting
	logger := createLogger(options)
	options.logger = logger

	if err != nil || len(args) == 0 || options.output == "" {
		showUsageAndExit(options, flags)
	}

	if err := validateAndProcessArchitecture(options); err != nil {
		logger.Error("Invalid architecture configuration", log.Err(err))
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return options, args
}

// showUsageAndExit displays usage information and exits.
func showUsageAndExit(options *optionFlags, flags *flag.FlagSet) {
	printBanner(options)
	fmt.Printf("usage: retroasm [options] <file to assemble>\n\n")
	flags.PrintDefaults()
	fmt.Println()
	os.Exit(1)
}

// printBanner displays the application banner.
func printBanner(options *optionFlags) {
	if !options.quiet {
		fmt.Println("[-------------------------------------]")
		fmt.Println("[ retroasm - retro computer assembler ]")
		fmt.Printf("[-------------------------------------]\n\n")
		options.logger.Info("Build info", log.String("version", buildinfo.Version(version, commit, date)))
	}
}

// assembleFile processes the input assembly file and generates output.
func assembleFile(options *optionFlags, args []string) error {
	inputData, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("opening input file '%s': %w", args[0], err)
	}

	ctx := app.Context()

	// Chip-8 uses the direct assembler API as it is not yet supported by the retroasm high-level API.
	if options.cpu == cpuChip8 {
		return assembleChip8File(ctx, inputData, options.output)
	}

	asm := retroasm.New()
	if err := registerArchitectureForCPU(asm, options.cpu, options.z80Profile); err != nil {
		return fmt.Errorf("registering architecture '%s': %w", options.cpu, err)
	}

	input := &retroasm.TextInput{
		Source:     bytes.NewReader(inputData),
		SourceName: args[0],
		ConfigFile: options.config,
	}

	output, err := asm.AssembleText(ctx, input)
	if err != nil {
		return fmt.Errorf("assembling input file '%s': %w", args[0], err)
	}

	if err = os.WriteFile(options.output, output.Binary, 0o644); err != nil {
		return fmt.Errorf("writing output file '%s': %w", options.output, err)
	}

	return nil
}

func assembleChip8File(ctx context.Context, inputData []byte, outputFile string) error {
	cfg := chip8.New()

	var buf bytes.Buffer
	asm := assembler.New(cfg, &buf)

	if err := asm.Process(ctx, bytes.NewReader(inputData)); err != nil {
		return fmt.Errorf("assembling chip8 input: %w", err)
	}

	if err := os.WriteFile(outputFile, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing output file '%s': %w", outputFile, err)
	}

	return nil
}

func registerArchitectureForCPU(asm retroasm.Assembler, cpuName, z80ProfileName string) error {
	switch cpuName {
	case cpu6502:
		cfg := m6502.New()
		adapter := retroasm.NewArchitectureAdapter(cpu6502, cfg, cfg)
		if err := asm.RegisterArchitecture(cpu6502, adapter); err != nil {
			return fmt.Errorf("registering architecture '%s': %w", cpu6502, err)
		}
		return nil

	case cpuM68000:
		cfg := archm68000.New()
		adapter := retroasm.NewArchitectureAdapter(cpuM68000, cfg, cfg)
		if err := asm.RegisterArchitecture(cpuM68000, adapter); err != nil {
			return fmt.Errorf("registering architecture '%s': %w", cpuM68000, err)
		}
		return nil

	case cpuZ80:
		profileKind, err := z80profile.Parse(z80ProfileName)
		if err != nil {
			return fmt.Errorf("parsing z80 profile: %w", err)
		}

		cfg := archz80.New(archz80.WithProfile(profileKind))
		adapter := retroasm.NewArchitectureAdapter(cpuZ80, cfg, cfg)
		if err := asm.RegisterArchitecture(cpuZ80, adapter); err != nil {
			return fmt.Errorf("registering architecture '%s': %w", cpuZ80, err)
		}
		return nil

	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedCPU, cpuName)
	}
}
